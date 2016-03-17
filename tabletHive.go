package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ASPecherkin/TabletHive/tablet"
)

// Authtokens stores json represent array of tokens
type Authtokens struct {
	Tokens []string `json:"tokens"`
}

// HiveConfig gather all of needed configs
type HiveConfig struct {
	ServerURL  string `json:"server"`
	TokensPath string `json:"token_file_path"`
	Endpoints  `json:"endpoints"`
}

// Endpoints stores all urls for requests
type Endpoints struct {
	GetRides     string `json:"get_rides"`
	UpdateStatus string `json:"update_status"`
}

// GetConfigJSON func get full path to config.json and store it in HiveConfig struct
func GetConfigJSON(jsonFile string) (cfg HiveConfig, err error) {
	jsonDoc, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		log.Fatalf("Could not read config file: %s ", err)
		return
	}
	err = json.Unmarshal(jsonDoc, &cfg)
	return cfg, err
}

// TabletClient one unit of hive
type TabletClient struct {
	ID         string
	Token      string
	DeviceID   string
	RespObj    tablet.Ride
	Rawresp    string
	StatusCode int
	ch         chan string
}

// Result store meta info about every request
type Result struct {
	RequestType   string  `json:"type"`
	AuthToken     string  `json:"token"`
	RequestURL    string  `json:"url"`
	RequestStatus int     `json:"status_code"`
	ProcessedTime float64 `json:"processed_time"`
}

// HiveResults stores all results of running
type HiveResults struct {
	When          string   `json:"when"`
	ElapsedTime   float64  `json:"elapsed_time"`
	GetResults    []Result `json:"get_results"`
	UpdateResults []Result `json:"update_results"`
	OthersResults []Result `json:"others_result"`
}

// GetRide create connect amd get ride for that token
func (t *TabletClient) GetRide(wg *sync.WaitGroup, cfg *HiveConfig, res chan Result) error {
	defer wg.Done()
	client := &http.Client{}
	url := strings.Join(append([]string{cfg.ServerURL, cfg.Endpoints.GetRides, t.DeviceID}), "")
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	request.Header.Add("HTTP-AUTH-TOKEN", t.Token)
	start := time.Now()
	responce, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	jsonData, err := ioutil.ReadAll(responce.Body)
	res <- Result{RequestType: "GET_RIDE", AuthToken: t.Token, RequestURL: url, RequestStatus: responce.StatusCode, ProcessedTime: time.Since(start).Seconds()}
	defer responce.Body.Close()
	if err != nil && err != io.EOF {
		fmt.Println("error reading from responce Body", err)
		return err
	}
	t.StatusCode = responce.StatusCode
	if responce.StatusCode == 404 {
		t.Rawresp = string(jsonData)
		return nil
	} else if responce.StatusCode == 200 {
		var answer tablet.Ride
		err = json.Unmarshal([]byte(jsonData), &answer)
		if err != nil {
			fmt.Printf("err: %s  with token : %s when unmarhal this   \n", err, t.Token)
		}
		t.RespObj, t.Rawresp = answer, string(jsonData)
		return nil
	} else {
		t.Rawresp = string(jsonData)
		return nil
	}
}

// ConsumeResults will store all of results in one HiveResults
func ConsumeResults(input chan Result, cfg *HiveConfig, testResult *HiveResults) {
	for i := range input {
		switch i.RequestType {
		case "CONSUME":
			testResult.UpdateResults = append(testResult.UpdateResults, i)
		case "GET_RIDE":
			testResult.GetResults = append(testResult.GetResults, i)
		default:
			testResult.OthersResults = append(testResult.OthersResults, i)
		}
	}
}

// Func generateAuthTokens
// TODO write func for netgerate list of auth tokens

// ConsumeRidePoints func create serias of request emulates real status updating
func ConsumeRidePoints(authToken string, points []tablet.RidePoint, wg *sync.WaitGroup, cfg *HiveConfig, res chan Result) error {
	defer wg.Done()
	client := &http.Client{}
	requestURL := cfg.ServerURL + cfg.Endpoints.UpdateStatus
	for _, v := range points {
		var jsonStr = []byte(`{"ride_point":{"status":"departure"}}`)
		t := strings.Join(append([]string{requestURL}, strconv.Itoa(int(v.ID))), "")
		req, err := http.NewRequest("PUT", t, bytes.NewBuffer(jsonStr))
		req.Header.Set("HTTP-AUTH-TOKEN", "wMTTN0bOUvNVkiVpYQd8AA")
		req.Header.Set("Content-Type", "application/json")
		start := time.Now()
		resp, err := client.Do(req)
		resp.Body.Close()
		res <- Result{RequestType: "CONSUME", AuthToken: authToken, RequestURL: t, RequestStatus: resp.StatusCode, ProcessedTime: time.Since(start).Seconds()}
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
	cfg, err := GetConfigJSON("./config.json")
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
	hive := make([]TabletClient, 0, 1000)
	if err != nil {
		fmt.Println("error while read tokens.json", err)
	}
	for k, v := range tokens.Tokens[0:] {
		hive = append(hive, TabletClient{ID: v, Token: v, DeviceID: strconv.Itoa(k + 1)})
	}
	fmt.Printf("we have %d tokens \n", len(hive))
	res := make(chan Result, 5)
	testCase := HiveResults{When: start.String()}
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
	for k := range ridePoints {
		wg.Add(1)
		go ConsumeRidePoints(k, ridePoints[k], &wg, &cfg, res)
	}
	wg.Wait()
	secs := time.Since(start).Seconds()
	fmt.Printf("we all done with: %.5fs \n", secs)
	testCase.ElapsedTime = secs
	jsondata, err := json.Marshal(testCase)
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

func getTokens(path string) (tokens Authtokens, err error) {
	tokens = Authtokens{}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error while open file with tokens", err)
		return tokens, err
	}
	err = json.Unmarshal(content, &tokens)
	return tokens, nil
}
