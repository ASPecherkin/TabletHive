package main

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	config "github.com/ASPecherkin/TabletHive/hiveConfig"
	result "github.com/ASPecherkin/TabletHive/storeResults"
	"github.com/ASPecherkin/TabletHive/tablet"
)

func getSadiraToken(cfg *config.HiveConfig) {
	content, err := ioutil.ReadFile(cfg.DeviceCodes)
	if err != nil {
		log.Fatalf("Could not read device ids file %s with error %s ", cfg.DeviceCodes, err)
	}
	ids := deviceIds{}
	err = json.Unmarshal(content, &ids)
	tokensFile, err := os.Create("./sadiraTokens.json")
	defer tokensFile.Close()
	if err != nil {
		log.Fatalln(err)
	}
	st := make([]sadiraUser, 0, 5)
	client := &http.Client{}
	for _, id := range ids.IDs {
		url := strings.Join(append([]string{cfg.ServerURL, cfg.Endpoints["sign_in"].URL, "login=strela_operator", "&device_code=", id}), "")
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalln(err)
		}
		authHeader := "Basic " + b64.StdEncoding.EncodeToString([]byte("strela_operator:strela"))
		req.Header.Add("Authorization", authHeader)
		time.Sleep(time.Duration(cfg.Endpoints["sign_in"].Delay))
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		jsonData, err := ioutil.ReadAll(resp.Body)
		token := sadiraUser{DeviceID: id}
		err = json.Unmarshal([]byte(jsonData), &token)
		if token.Code == "ok" {
			st = append(st, token)
		}
		defer resp.Body.Close()
	}
	jsonTokens, err := json.Marshal(st)
	tokensFile.Write(jsonTokens)
}

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
func getTokens(path string) (tokens config.Authtokens, err error) {
	tokens = config.Authtokens
	content, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error while open file with tokens", err)
		return tokens, err
	}
	err = json.Unmarshal(content, &tokens)
	return tokens, nil
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
	start := time.Now()
	fmt.Fprintf(os.Stdout, "We start at: %v\n", start)
	cfg, err := config.GetConfigJSON("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		pprof.WriteHeapProfile(f)
	}
	var wg sync.WaitGroup
	tokens, err := getTokens(cfg.TokensPath)
	hive := make([]tablet.Device, 0, 3000)
	if err != nil {
		fmt.Println("error while read tokens.json", err)
	}
	getSadiraToken(&cfg)
	for k, v := range tokens.Tokens[0:] {
		hive = append(hive, tablet.Device{Token: v, ID: strconv.Itoa(k + 1)})
	}
	fmt.Printf("we have %d tokens \n", len(hive))
	res := make(chan result.Result, 5)
	testCase := result.HiveResults{When: start.String()}
	go ConsumeResults(res, &cfg, &testCase)
	for k := range hive {
		wg.Add(1)
		go hive[k].GetRide(&wg, &cfg, res)
	}
	wg.Wait()
	ridePoints := make(map[string][]tablet.RidePoint)
	for _, tablerClient := range hive {
		for _, factRides := range tablerClient.RespObj.FactRides {
			if len(factRides.RidePoints) != 0 {
				ridePoints[tablerClient.Token] = factRides.RidePoints
			}
		}
	}
	fmt.Println("Start updating statuses")
	for k := range ridePoints {
		wg.Add(1)
		go ConsumeRidePoints(k, ridePoints[k], &wg, &cfg, res)
	}
	wg.Wait()
	secs := time.Since(start).Seconds()
	fmt.Printf("we all done with: %.5fs \n", secs)
	testCase.ElapsedTime = secs
	testCase.Lock()
	jsondata, err := json.Marshal(testCase)
	testCase.Unlock()
	if err != nil {
		fmt.Println(err)
	}
	ResultFile, err := os.Create("./TestCase.json")
	defer ResultFile.Close()
	if err != nil {
		fmt.Println(err)
	}
	ResultFile.Write(jsondata)
}
