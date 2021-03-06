package main

import (
	//"bytes"
	"fmt"
	//    "log"
	"net/http"
	//    "os"
	"strconv"
	"time"
	//    "database/sql" //pgsql for cache

	"github.com/gin-gonic/gin"
	//    "github.com/russross/blackfriday"
	_ "github.com/lib/pq"
)

import (
	"encoding/json"
	// "io/ioutil"
	"sort"
	"strings"
)

type MarketHistoricalDataResponse struct {
	Status struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
	Results []SHResults `json:"results"`
}
type SHResults struct {
	Symbol string `json:"symbol"`
	//		Timestamp string `json:"timestamp"`
	TradingDay string `json:"tradingDay"`
	/*		Open float64 `json:"open"`
			High float64 `json:"high"`
			Low float64 `json:"low"` */
	Close float64 `json:"close"`
	/*		Volume int `json:"volume"`
			OpenInterest int `json:"openInterest"` */
}

func transposeSlice(in [][]SCtrPt) (out [][]SCtrPt) {
	if in == nil {
		return out
	}
	if in[0] == nil {
		return out
	}
	sx := len(in[0])
	sy := len(in)
	out = make([][]SCtrPt, sx)
	for i := range out {
		out[i] = make([]SCtrPt, sy)
	}
	for i, v := range in {
		for j, val := range v {
			out[j][i] = val
		}
	}
	return out
}


func extractDateNamesForTmpl(dataPts [][]SCtrPt) ([]string) {
	t:=make([]string,len(dataPts))
	for i, a := range dataPts {
		for _, b := range a {
			if strings.Compare(b.TradingDay,"")!=0 {
				t[i] = b.TradingDay
				break
			}
		}
	}
	return t
}

// view history of single contract page
func contractHistoryFunc(c *gin.Context) {
	title := c.Param("name")
	//action := c.Param("action")
	//check if page exists
	if stringInSlice(title, allExistingContracts) == false {
		c.String(http.StatusNotFound, "requested contract ("+title+") is not listed")
		return
	}
	//loads or downloads quotation
	cquotes := loadContractHistoryData(title)
	if cquotes == nil {
		c.String(http.StatusNotFound, "requested contract ("+title+") no quotations")
		return
	}
	//renderTemplate(w, "view", &p)
	renderData := transposeSlice(cquotes)
	c.HTML(http.StatusOK, "viewhistorygraph.tmpl.html", gin.H{
		"Title":         title,
		"ContratQuotes": renderData,
		"CurvesTitles":	 extractDateNamesForTmpl(renderData),
		"FooterData":    time.Now().Format("_2 Jan 2006 15:04:05"),
	})
}

var StepInMonths = 2

//try to load data from local, otherwise downloads it
func loadContractHistoryData(contractRoot string) (dataPts [][]SCtrPt) {
	//local load
	if body := loadFromDB("h-" + contractRoot /*+ ".json"*/); body != nil {
		err := json.Unmarshal(body, &dataPts)
		perror(err)
		fmt.Printf("Read %v values\n", len(dataPts))
		return
	}
	//downloads data
	//dataName := contractRoot
	chlout := make(chan *[]SCtrPt)
	nbl := 0
	for i := time.Now().AddDate(-2,0,0); i.Before(time.Now().AddDate(10, 0, 0)); i = i.AddDate(0, StepInMonths, 0) {
		go chl_get_HttpContentHistory(contractRoot, i, chlout)
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

	dataPts = transposeSlice(dataPts)
	for _, a := range dataPts {
		sort.Sort(ByTime(a))
	}
	dataPts = transposeSlice(dataPts)
	//save downloaded data
	saveToDB("h-"+contractRoot /*+".json"*/, dataPts)

	return
}

//download JSON historic from barchart & convert json to struct
func chl_get_HttpContentHistory(contractRoot string, xti time.Time, chlout chan *[]SCtrPt) {
	//the product is determined by the i time parameter
	var ccode = contractRoot + MonthsCode[xti.Month()-1] + strconv.Itoa(xti.Year()-2000)
	url := "http://marketdata.websol.barchart.com/getHistory.json?key=5739c7e96e351d2d8c0c98f34f720965&&symbol=" + ccode
	//startdate is 2 years ago (max storage of barchart)
	//enddate is today
	url = url + "&type=monthly&endDate=" + time.Now().Format("20060102") + "&startDate=" + time.Now().AddDate(-2, 0, 0).Format("20060102")
	res, err := http.Get(url)
	perror(err)
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	var data MarketHistoricalDataResponse
	err = decoder.Decode(&data)
	if err != nil {
		//no panic: most errors are caused by TradingDay = false, when there is no quotation is available at barchart
		//fmt.Printf("REQ:%v\n", url)
		//fmt.Printf("DATA:%+v\n", res.Body)
		//fmt.Printf("ERROR:%v\n", err)
		chlout <- nil
		return
	}
	if data.Results != nil {
		/*if strings.Contains(ccode, "ZWM16") {
			saveToFile("mytest.json", data)
			fmt.Printf("saved mytest.json with %vdata ", len(data.Results))
		}*/
		//fmt.Printf(",%v", len(data.Results))
		//now, 2months ago, 1 year ago, 2 years ago :
		chresult := make([]SCtrPt, 4)
		searchInMarketHistoricalDataResponse(xti, time.Now().Format("2006-01-"), data.Results, &(chresult[0]))
		searchInMarketHistoricalDataResponse(xti.AddDate(0, 2, 0), time.Now().AddDate(0, -2, 0).Format("2006-01-"), data.Results, &(chresult[1]))
		searchInMarketHistoricalDataResponse(xti.AddDate(1, 0, 0), time.Now().AddDate(-1, 0, 0).Format("2006-01-"), data.Results, &(chresult[2]))
		searchInMarketHistoricalDataResponse(xti.AddDate(2, 0, 0), time.Now().AddDate(-2, 0, 0).Format("2006-01-"), data.Results, &(chresult[3]))
		chlout <- &chresult
	} else {
		chlout <- nil
	}
}

func searchInMarketHistoricalDataResponse(i time.Time, searchString string, data []SHResults, rez *SCtrPt) {
	*rez = SCtrPt{i, 0, "", ""}
	if data == nil {
		return
	}
	for _, a := range data {
		if strings.Contains(a.TradingDay, searchString) {
			*rez = SCtrPt{i, a.Close, a.Symbol, a.TradingDay}
			return
		}
	}
}
