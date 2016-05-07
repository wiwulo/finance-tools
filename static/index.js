
"use strict";

var ticker = 'AAPL';
//sign up to quandl and retrieve you key
var key = 'RyRk2YQmUS4zxUJSMpfs';
var url = 'https://www.quandl.com/api/v3/datasets/CURRFX/'+ticker+'.json?column_index=1&auth_token='+key;
//var shorturl = 'https://www.quandl.com/api/v3/datasets/CURRFX/USD';
var endurl = '.json?column_index=1&auth_token='+key;

var mydata;
var nbCurrencies = 28;
var refCurr = "USD";

function logArrayElements(element, index, array) 
{
	var url = this.shorturl;
	var i = 0;
	this.date.forEach(function(ldate, i)
	{
		var xmlhttp = new XMLHttpRequest();
		//console.log('a[' + index + '] = ' + element);
		//console.log( url+element+endurl+"&end_date="+ldate+"&start_date="+ldate );
		xmlhttp.open("GET", url+element+endurl+"&end_date="+ldate+"&start_date="+ldate , true);
		//console.log(url+element+endurl+"&end_date="+ldate+"&start_date="+ldate);
		xmlhttp.setRequestHeader("Content-Type", "application/jsonp");
		
		xmlhttp.onload = function() {
			var result = JSON.parse(this.responseText);
			//console.log(element+" "+i+" "+result.dataset.data[0][0]+" "+result.dataset.data[0][1]);
			//result.dataset.myShowData();
			//store record: create new element, if unknown
			if(  mydata.get(element) === undefined )	mydata.set(element, new Array(NaN,NaN,NaN,NaN,NaN));
			var e = mydata.get(element);
			if(result.dataset.data.length > 0) e[i]=result.dataset.data[0][1];
			if(element == "GOLD") e[i]=100/e[i]; // GOLD quotation is inverted (and very low) !!!
			//mydata.get(element).doCalcDiff();
			//result.data
			//        .map(   function(i){ return i[1]; })
						//compute ref currency unit
			doCalcRefUnits(mydata); 
			doAnalysis(mydata);
			//xmlhttp.abort();
		}; 
		xmlhttp.send();
	});
}

function doCalcRefUnits(m)
{
	//Compute all the quotes in Ref unit: mydata[3] , [4]
	//rcv = ref currency. When ref currency == "USD", computation is unit
	var rcv = ( refCurr == "USD" )?  new Array(1,1,0,1,1) : m.get(refCurr);
	if(rcv == null) return;
	if( isNaN(rcv[0]) || isNaN(rcv[1]) ) return;

	m.forEach(function(mv, mk, mmap) {
		mv[3] = mv[0] / rcv[0];
		mv[4] = mv[1] / rcv[1];
		//compute perf
		if( (!isNaN(mv[3])) && (!isNaN(mv[4])) )
			//mv[2] = ( mv[3] - mv[4] ) / mv[4];
			mv[2] = ( mv[4] / mv[3] ) -1;
	});
	//correction of the base currency
	rcv[3] = 1 / rcv[0];
	rcv[4] = 1 / rcv[1];
	rcv[2] = ( rcv[4] / rcv[3] ) - 1;	//( rcv[3] - rcv[4] ) / rcv[4];
}

function doAnalysis(m)
{
	//sort map, by performance
	var keys = [];
	for (let key of m.keys()) {	keys.push(key);	}
	keys.sort(function(a, b) {
			return m.get(b)[2] - m.get(a)[2];
    });

    //empty result display
    var x = document.getElementById("mainContent");
    while(x.rows.length > 1)  x.deleteRow(x.rows.length-1);
    //show results
    keys.forEach(function(k, index) { 
    	var elt=m.get(k); 
    	//console.log(k+" "+elt[2]+" :"+elt[0]+"; "+elt[1] ); 
    
    	var new_row = x.rows[0].cloneNode(true);
    	new_row.cells[0].innerText = index;
    	new_row.cells[1].innerText = (k == refCurr) ? "USD" : k;//inversion Curr<>Ref
    	new_row.cells[2].innerText = (isNaN(elt[2]) ? "no data" : (parseFloat(elt[2])*100).toFixed(2) )+"%";
    	/*new_row.cells[4].innerText = (isNaN(elt[1]) ? "no data" : parseFloat(elt[1]).toFixed(4)) +"->"+
    								(isNaN(elt[0]) ? "no data" : parseFloat(elt[0]).toFixed(4));
    	new_row.cells[3].innerText = (isNaN(elt[4]) ? "no data" : parseFloat(elt[4]).toFixed(4)) +"->"+
    								(isNaN(elt[3]) ? "no data" : parseFloat(elt[3]).toFixed(4));*/
    	new_row.cells[3].innerText = (isNaN(elt[4]) ? "no data" : parseFloat(elt[4]).toFixed(4));
    	new_row.cells[4].innerText = (isNaN(elt[3]) ? "no data" : parseFloat(elt[3]).toFixed(4));
    	x.appendChild( new_row );
    });    
}

/*
Object.prototype.myShowData = function()
{
	var x = document.getElementById("mainContent");
    var new_row = x.rows[0].cloneNode(true);
    //var len = x.rows.length;
    //new_row.cells[0].innerHTML = len;
	// grab the input from the first cell and update its ID and value
    new_row.cells[0].innerText = this.dataset_code;
    
	this.data.forEach( function(i){	new_row.cells[2].innerText = i[1].toLocaleString(); }  );

	x.appendChild( new_row );
};
*/

function MainIn()
{
	//empty results and display
	mydata = new Map();
    var x = document.getElementById("mainContent");
    while(x.rows.length > 1)  x.deleteRow(x.rows.length-1);
    
    //Get parameters:
	//Today
	var date1 = document.getElementById("today_date").value;
	var date2 = new Date(date1);
	//check date
	var day = date2.getDay();
	if (day >= new Date() ) {
		document.getElementById("date_comment").innerText = "No trading data from the future, yet!"; return; }
	if( (day == 6) || (day == 0) ) {    // 6 = Saturday, 0 = Sunday
		document.getElementById("date_comment").innerText = "No trading during the weekend"; return; }
	document.getElementById("date_comment").innerText = "";	
	//computes start date
	var ds = document.getElementById("time_period");
	switch(document.getElementById("time_period").value) {
	case "3M":
			date2.setMonth(date2.getMonth() -3 );	break;
	case "1Y":
			date2.setFullYear(date2.getFullYear() -1 );	break;
	case "YTD":
			date2.setMonth(0,0);	break;
	case "1M":
	default:
			date2.setMonth(date2.getMonth() -1 );	break;
	}
	document.getElementById("end_date").innerText = date2.toISOString().slice(0,10);
	var params = {	date : [ date1, date2.toISOString().slice(0,10) ],
					shorturl : 'https://www.quandl.com/api/v3/datasets/CURRFX/USD' };
	var currList = new Array("AUD","BRL","CAD","CHF","CLP","CNY","COP",
			"CZK","DKK","NGN","GBP","HKD","INR","JPY","KRW","MXN","MYR","NOK",
			"NZD","PEN","PLN","RUB","SEK","SGD","TRY","EUR","ZAR");
	//Ref currency
	refCurr = document.getElementById("ref_currency").value;
	//Start the computations	
	currList.forEach(logArrayElements, params);
	
	var params2 = {	date : [ date1, date2.toISOString().slice(0,10) ],
				shorturl : 'https://www.quandl.com/api/v3/datasets/LBMA/' };
	["GOLD"].forEach(logArrayElements, params2);

}

