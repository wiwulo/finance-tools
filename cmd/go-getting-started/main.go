package main

import (
	"bytes"
	"database/sql" //pgsql for cache
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/russross/blackfriday"
)

import (
	"encoding/json"
	//	"fmt"
	// "net/http"
	// "time"
	// "strconv"
	"io/ioutil"
	// "os"
	"sort"
	// "encoding/csv"
	"strings"
)

//struct for single contract page
/*type Page struct {
	Title         string
	ContratQuotes []SCtrPt
	//Body  []byte
	FooterData string
}*/

//TODO: log ALL events for display in webpage

type MarketDataResponse struct {
	Status struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
	Results []SResults `json:"results"`
}
type SResults struct {
	Symbol          string  `json:"symbol"`
	Exchange        string  `json:"exchange"`
	Name            string  `json:"name"`
	DayCode         string  `json:"dayCode"`
	ServerTimestamp string  `json:"serverTimestamp"`
	Mode            string  `json:"mode"`
	LastPrice       float64 `json:"lastPrice,float64"`
	/*	TradeTimestamp  string  `json:"tradeTimestamp"`
		NetChange       float64 `json:"netChange,float64"`
		PercentChange   float64 `json:"percentChange,float64"`
		UnitCode        string  `json:"unitCode"`
		Open            string  `json:"open,string"`
		High            float64 `json:"high,float64"`
		Low             float64 `json:"low,float64"`*/
	Close float64 `json:"close,float64"`
	/*	Flag            string  `json:"flag"`
		Volume          int64   `json:"volume,int64"`*/
}

// Contract delivery months https://en.wikipedia.org/wiki/Delivery_month
var MonthsCode = [12]string{"F", "G", "H", "J", "K", "M", "N", "Q", "U", "V", "X", "Z"}

//Stored values, in Capital first letter for JSON marshalling
type SCtrPt struct {
	Day        time.Time
	Data       float64
	Symbol     string
	TradingDay string
}

// ByTime implements sort.Interface for []SCtrPt based on the Day field.
type ByTime []SCtrPt

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Day.Before(a[j].Day) }

// tool to search string in string[]
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

///////////////////////////////////////////////
//webpage to show list of contracts
func repeatFunc(c *gin.Context) {
	var buffer bytes.Buffer
	//buffer.WriteString(string(len(allExistingContracts)))
	for i := 0; i < len(allExistingContracts); i++ {
		buffer.WriteString(fmt.Sprintf("%v ", allExistingContracts[i]))
	}
	c.String(http.StatusOK, buffer.String())
}

func dbClearFunc(c *gin.Context) {
	var buffer bytes.Buffer
	err := clearDB()
	if err != nil {
		buffer.WriteString(fmt.Sprintf("Error destroying table contracts: %q", err))
		return
	} else {
		buffer.WriteString("destroying table contracts ok")
	}
	c.String(http.StatusOK, buffer.String())
}

// view single contract page
func dbFunc(c *gin.Context) {
	title := c.Param("name")
	//action := c.Param("action")
	//check if page exists
	if stringInSlice(title, allExistingContracts) == false {
		c.String(http.StatusNotFound, "requested contract ("+title+") is not listed")
		return
	}
	//loads or downloads quotation
	cquotes := loadContractData(title)
	if cquotes == nil {
		c.String(http.StatusNotFound, "requested contract ("+title+") no quotations")
		return
	}
	//renderTemplate(w, "view", &p)
	c.HTML(http.StatusOK, "viewgraph.tmpl.html", gin.H{
		"Title":         title,
		"ContratQuotes": cquotes,
		"FooterData":    time.Now().Format("_2 Jan 2006 15:04:05"),
	})
}

//TODO: log the errors (for display in webpage)
func perror(err error) {
	if err != nil {
		panic(err)
	}
}

//////////////////////////////// GET & CACHE QUOTATIONS
//download JSON quotation from barchart
func get_HttpContent(symbol string) []SResults {
	url := "http://marketdata.websol.barchart.com/getQuote.json?key=5739c7e96e351d2d8c0c98f34f720965&&symbols=" + symbol

	res, err := http.Get(url)
	perror(err)
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	var data MarketDataResponse
	err = decoder.Decode(&data)
	if err != nil {
		fmt.Printf("REQ:%v\n", url)
		fmt.Printf("DATA:%+v\n", res.Body)
		fmt.Printf("STRUCT:%+v\n", data)
		perror(err)
	}
	//fmt.Printf("%+v",data)
	/*for _, element := range data.Results {
		fmt.Printf("%+v",element)
	}*/
	return data.Results
}

//convert json to struct
func chl_get_HttpContent(contractRoot string, i time.Time, chlout chan *SCtrPt) {
	var ccode = contractRoot + MonthsCode[i.Month()-1] + strconv.Itoa(i.Year()-2000)
	//fmt.Printf("(%v", ccode)
	data := get_HttpContent(ccode)
	if data != nil {
		//fmt.Printf(",%v", data[0].Close)
		//dataPts = append(dataPts, SCtrPt{i, data[0].Close})
		chlout <- &SCtrPt{i, data[0].Close, data[0].Symbol, data[0].DayCode}
		//dataName = contractRoot + "-" + data[0].Exchange + "-" + data[0].Name
	} else {
		chlout <- nil
	}
}

//cache to file
func saveToFile(filename string, ld interface{}) { //}[]SCtrPt) {
	b, err := json.Marshal(ld)
	perror(err)
	err = ioutil.WriteFile(filename, b, 0600)
	perror(err)
}

//loads existing file, or return nil
func loadFromFile(filename string) (body []byte) { //[]SCtrPt) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return
	}
	body, err := ioutil.ReadFile(filename)
	perror(err)
	//err = json.Unmarshal(body, &ldata)
	//perror(err)
	return
}
func clearDB() error {
	_, err := db.Exec("DROP TABLE contracts;")
	return err
}

//db cache
func saveToDB(filename string, ld interface{}) { //[]SCtrPt) {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS contracts (code varchar PRIMARY KEY, dday date, data text)"); err != nil {
		fmt.Printf("Error creating database table contracts: %q", err)
		return
	}
	b, err := json.Marshal(ld)
	perror(err)
	if _, err := db.Exec(`UPDATE contracts SET data='` + string(b) + `' , dday = current_date WHERE code='` + filename + `' ;
INSERT INTO contracts (code, dday, data) 
 SELECT '` + filename + `' , current_date , '` + string(b) + `' 
 WHERE NOT EXISTS (SELECT 1 FROM contracts WHERE code='` + filename + `');`); err != nil {
		fmt.Printf("Error writing data to pg cache: %q", err)
		return
	}
}
func loadFromDB(filename string) (body []byte) { //[]SCtrPt) {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS contracts (code varchar PRIMARY KEY, dday date, data text)"); err != nil {
		fmt.Printf("Error creating database table contracts: %q", err)
		return
	}
	rows, err := db.Query("SELECT data FROM contracts WHERE code = '" + filename + "' AND dday = current_date")
	if err != nil {
		fmt.Printf("Error reading ticks: %q", err)
		return
	}
	defer rows.Close()
	rows.Next() //for rows.Next() {
	//var body []byte
	if err := rows.Scan(&body); err != nil {
		fmt.Printf("Error scanning data: %q", err)
		return
	}
	//err = json.Unmarshal(body, &ldata)
	//perror(err)

	return
}

//try to load data from local, otherwise downloads it
func loadContractData(contractRoot string) (dataPts []SCtrPt) {
	//local load
	if body := loadFromDB(contractRoot /*+ ".json"*/); body != nil {
		err := json.Unmarshal(body, &dataPts)
		perror(err)
		fmt.Printf("Read %v values\n", len(dataPts))
		return
	}
	//downloads data
	//dataName := contractRoot
	chlout := make(chan *SCtrPt)
	nbl := 0
	for i := time.Now(); i.Before(time.Now().AddDate(10, 0, 0)); i = i.AddDate(0, 1, 0) {
		go chl_get_HttpContent(contractRoot, i, chlout)
		nbl++
	}
	for ; nbl > 0; nbl-- {
		select {
		case ldata := <-chlout:
			if ldata != nil { //when we have some quotation
				dataPts = append(dataPts, *ldata) //append quotation
			}
			break
		case <-time.After(30 * time.Second):
			nbl = 0 //break out loop
			break
		}
	}

	sort.Sort(ByTime(dataPts))
	//save downloaded data
	saveToDB(contractRoot /*+".json"*/, dataPts)

	return
}

//////////////// Load constants
//load []strings from CSV file
func loadTableFromSring(fileName string) []string {
	/*f, err := os.Open(fileName)
	    perror(err)
	    defer f.Close()
	    reader := csv.NewReader(strings.NewReader(fileName))
		bytes, err := reader.ReadAll()
		perror(err)*/
	bytes := strings.Split(fileName, ",")
	if bytes != nil {
		return bytes //[0]
	}
	return nil
}

//////////////////////////////////////// MAIN
var (
	repeat               int
	db                   *sql.DB  = nil
	allExistingContracts []string //= loadTableFromCSVfile("contracts-list.csv")
)

func main() {
	var err error
	var errd error

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	db, errd = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if errd != nil {
		log.Fatalf("Error opening database: %q", errd)
	}

	tCtrslst := os.Getenv("CONTRACTSLIST")
	if tCtrslst == "" {
		log.Fatal("no CONTRACTSLIST in environment variables")
	}
	allExistingContracts = loadTableFromSring(tCtrslst)
	log.Printf("Found %v contracts in .env file\n", len(allExistingContracts))

	tStr := os.Getenv("REPEAT")
	repeat, err = strconv.Atoi(tStr)
	if err != nil {
		log.Print("Error converting $REPEAT to an int: %q - Using default", err)
		repeat = 5
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	router.GET("/mark", func(c *gin.Context) {
		c.String(http.StatusOK, string(blackfriday.MarkdownBasic([]byte("**hi!**"))))
	})

	//view list of contracts
	router.GET("/contracts", repeatFunc)
	//clear db
	router.GET("/clear", dbClearFunc)
	//view single contract
	router.GET("/contract/:name/*action", dbFunc)
	//view history of contract
	router.GET("/contract-history/:name/*action", contractHistoryFunc)

	router.Run(":" + port)
}
