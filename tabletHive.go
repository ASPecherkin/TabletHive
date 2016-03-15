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

// GetRide create connect amd get ride for that token
func (t *TabletClient) GetRide(wg *sync.WaitGroup, cfg *HiveConfig) (int, error) {
	defer wg.Done()
	client := &http.Client{}
	url := strings.Join(append([]string{cfg.ServerURL, cfg.Endpoints.GetRides, t.DeviceID}), "")
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Add("HTTP-AUTH-TOKEN", t.Token)
	responce, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	jsonData, err := ioutil.ReadAll(responce.Body)
	defer responce.Body.Close()
	if err != nil && err != io.EOF {
		fmt.Println("error reading from responce Body", err)
		return 0, err
	}
	t.StatusCode = responce.StatusCode
	if responce.StatusCode == 404 {
		t.Rawresp = string(jsonData)
		return responce.StatusCode, nil
	} else if responce.StatusCode == 200 {
		var answer tablet.Ride
		err = json.Unmarshal([]byte(jsonData), &answer)
		if err != nil {
			fmt.Printf("err: %s  with token : %s when unmarhal this   \n", err, t.Token)
		}
		t.RespObj, t.Rawresp = answer, string(jsonData)
		return responce.StatusCode, nil
	} else {
		t.Rawresp = string(jsonData)
		return responce.StatusCode, nil
	}
}

// Func generateAuthTokens
// TODO write func for netgerate list of auth tokens

// ConsumeRidePoints func create serias of request emulates real status updating
func ConsumeRidePoints(authToken string, points []tablet.RidePoint, wg *sync.WaitGroup, cfg *HiveConfig) (bool, error) {
	defer wg.Done()
	client := &http.Client{}
	requestURL := cfg.ServerURL + cfg.Endpoints.UpdateStatus
	for _, v := range points {
		var jsonStr = []byte(`{"ride_point":{"status":"departure"}}`)
		t := strings.Join(append([]string{requestURL}, strconv.Itoa(int(v.ID))), "")
		req, err := http.NewRequest("PUT", t, bytes.NewBuffer(jsonStr))
		req.Header.Set("HTTP-AUTH-TOKEN", "wMTTN0bOUvNVkiVpYQd8AA")
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			log.Panicln(err)
            os.Exit(1)
			return false, err
		}
		resp.Body.Close()
	}
	return true, nil
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
	for k := range hive {
		wg.Add(1)
		go hive[k].GetRide(&wg, &cfg)
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
		go ConsumeRidePoints(k, ridePoints[k], &wg, &cfg)
	}
	wg.Wait()
	secs := time.Since(start).Seconds()
	fmt.Printf("we all done with: %.5fs \n", secs)
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
