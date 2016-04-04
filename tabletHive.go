package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	config "github.com/ASPecherkin/TabletHive/hiveConfig"
	result "github.com/ASPecherkin/TabletHive/storeResults"
	tablet "github.com/ASPecherkin/TabletHive/tablet"
)

// ConsumeResults will store all of results in one HiveResults
func ConsumeResults(input chan result.Result, cfg *config.HiveConfig, testResult *result.HiveResults) {
	for i := range input {
		testResult.Lock()
		switch i.RequestType {
		case "CONSUME":
			testResult.UpdateResults = append(testResult.UpdateResults, i)
			testResult.Unlock()
		case "GET_RIDE":
			testResult.GetResults = append(testResult.GetResults, i)
			testResult.Unlock()
		default:
			testResult.OthersResults = append(testResult.OthersResults, i)
			testResult.Unlock()
		}
	}
}

// getTokens read file from path and stores they in tokens
func getLogins(path string) (logins config.SadirLogins, err error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error while open file with tokens", err)
		return
	}
	err = json.Unmarshal(content, &logins)
	return
}

// ConsumeRidePoints func create serias of request emulates real status updating
func ConsumeRidePoints(authToken string, points []tablet.RidePoint, wg *sync.WaitGroup, cfg *config.HiveConfig, res chan result.Result) error {
	defer wg.Done()
	client := &http.Client{}
	requestURL := cfg.ServerURL + cfg.Endpoints["update_status"].URL
	for _, v := range points {
		var jsonStr = []byte(`{"ride_point":{"status":"departure"}}`)
		t := strings.Join(append([]string{requestURL}, strconv.Itoa(int(v.ID))), "")
		req, err := http.NewRequest("PUT", t, bytes.NewBuffer(jsonStr))
		req.Header.Set("HTTP-AUTH-TOKEN", authToken)
		req.Header.Set("Content-Type", "application/json")
		time.Sleep(time.Duration(cfg.Endpoints["update_status"].Delay))
		start := time.Now()
		resp, err := client.Do(req)
		jsonData, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatalln(err)
		}
		res <- result.Result{RequestType: "CONSUME", AuthToken: authToken, RequestURL: t, RequestStatus: resp.StatusCode, Responce: string(jsonData), ProcessedTime: time.Since(start).Seconds()}
		if err != nil {
			log.Panicln(err)
			os.Exit(1)
			return err
		}
	}
	return nil
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write mem profile to file")

func main() {
	cfg, err := config.GetConfigJSON("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	logins, err := getLogins(cfg.LoginsPath)
	if err != nil {
		log.Fatalf("error while parse logins %s", err)
	}
	Devises := make([]tablet.Device, 0, 5)
	for k, v := range logins.Logins {
		Devises = append(Devises, tablet.Device{ID: strconv.Itoa(k+1), Login: v})
	}
	for k := range Devises {
		Devises[k].InitDevice(&cfg)
		Devises[k].GetSadiraToken(&cfg)
	}
    activeDevices := make([]tablet.Device,0,5)
    for k := range Devises {
        if Devises[k].StatusCode == 200 {
           activeDevices = append(activeDevices, Devises[k])
        } 
    }
    for k:= range activeDevices {
        fmt.Printf("Login %s with token: %s \n",Devises[k].Login,Devises[k].Token)
    }
}

// func main() {
// 	start := time.Now()
// 	fmt.Fprintf(os.Stdout, "We start at: %v\n", start)
// 	cfg, err := config.GetConfigJSON("./config.json")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	flag.Parse()
// 	if *cpuprofile != "" {
// 		f, err := os.Create(*cpuprofile)
// 		if err != nil {
// 			fmt.Println("Error: ", err)
// 		}
// 		pprof.StartCPUProfile(f)
// 		defer pprof.StopCPUProfile()
// 	}
// 	if *memprofile != "" {
// 		f, err := os.Create(*memprofile)
// 		if err != nil {
// 			fmt.Println("Error: ", err)
// 		}
// 		pprof.WriteHeapProfile(f)
// 	}
// 	var wg sync.WaitGroup
// 	// tokens, err := getTokens(cfg.TokensPath)
// 	hive := make([]tablet.Device, 0, 3000)
// 	if err != nil {
// 		fmt.Println("error while read tokens.json", err)
// 	}
// 	// getSadiraToken(&cfg)
// 	for k, v := range tokens.Tokens[0:] {
// 		hive = append(hive, tablet.Device{Token: v, ID: strconv.Itoa(k + 1)})
// 	}
// 	fmt.Printf("we have %d tokens \n", len(hive))
// 	res := make(chan result.Result, 5)
// 	testCase := result.HiveResults{When: start.String()}
// 	go ConsumeResults(res, &cfg, &testCase)
// 	for k := range hive {
// 		wg.Add(1)
// 		go hive[k].GetRide(&wg, &cfg, res)
// 	}
// 	wg.Wait()
// 	ridePoints := make(map[string][]tablet.RidePoint)
// 	for _, tablerClient := range hive {
// 		for _, factRides := range tablerClient.RespObj.FactRides {
// 			if len(factRides.RidePoints) != 0 {
// 				ridePoints[tablerClient.Token] = factRides.RidePoints
// 			}
// 		}
// 	}
// 	fmt.Println("Start updating statuses")
// 	for k := range ridePoints {
// 		wg.Add(1)
// 		go ConsumeRidePoints(k, ridePoints[k], &wg, &cfg, res)
// 	}
// 	wg.Wait()
// 	secs := time.Since(start).Seconds()
// 	fmt.Printf("we all done with: %.5fs \n", secs)
// 	testCase.ElapsedTime = secs
// 	testCase.Lock()
// 	jsondata, err := json.Marshal(testCase)
// 	testCase.Unlock()
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	ResultFile, err := os.Create("./TestCase.json")
// 	defer ResultFile.Close()
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	ResultFile.Write(jsondata)
// }
