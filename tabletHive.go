package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
)

type ServiceObject struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	ObjType string `json:"type"`
	TimeT   string `json:"time_t"`
	Phones  string `json:"phones"`
	TimeG1  string `json:"time_g1"`
	TimeG2  string `json:"time_g2"`
}

type Order struct {
	ID            uint          `json:"name"`
	Status        string        `json:"name"`
	ServiceType   string        `json:"name"`
	ServiceObject ServiceObject `json:"name"`
}

type RidePoint struct {
	ID          uint    `json:"name"`
	Number      uint    `json:"name"`
	Lat         float32 `json:"name"`
	Lng         float32 `json:"name"`
	AddressText string  `json:"name"`
	Status      string  `json:"name"`
	Kind        string  `json:"name"`
	Order       Order   `json:"order"`
}

type FactRides struct {
	ID         uint        `json:"name"`
	TimeStart  string      `json:"name"`
	RidePoints []RidePoint `json:"name"`
}

type Ride struct {
	ID        uint      `json:"id"`
	Number    string    `json:"name"`
	Duration  uint      `json:"name"`
	Distance  float32   `json:"name"`
	FactRides FactRides `json:"name"`
}

type HiveConfig struct {
	ServerEndpoint string `json:"endpoint_url"`
	TokensPath     string `json:"token_file_path"`
}

func GetConfigJSON(jsonFile string) (cfg HiveConfig, err error) {
	jsonDoc, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		log.Fatalf("Could not read config file: %s ", err)
		return
	}
	err = json.Unmarshal(jsonDoc, &cfg)
	return cfg, err
}

type TabletClient struct {
	ID       string
	Token    string
	DeviceID int
	RespObj  ServiceObject
    Rawresp  string
	ch       chan string
}

// GetRide create connect amd get ride for that token
func (t *TabletClient) GetRide(wg *sync.WaitGroup, cfg *HiveConfig, ch chan<- string) (bool, error) {
	defer wg.Done()
	client := &http.Client{}
	url := strings.Join([]string{cfg.ServerEndpoint}, string(t.DeviceID))
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	request.Header.Add("HTTP-AUTH-TOKEN", t.Token)
    fmt.Println(t.Token)
	responce, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	jsonData, err := ioutil.ReadAll(responce.Body)
	defer responce.Body.Close()
	if err != nil && err != io.EOF {
		fmt.Println("error reading from responce Body", err)
		return false, err
	}
    var answer ServiceObject
    if responce.Status == "200" {
       err = json.Unmarshal([]byte(jsonData), &answer)
       t.RespObj = answer
    } else {
       t.Rawresp = string(jsonData)
       ch <- fmt.Sprintf("DeviceID %d responce: %v  \n", t.DeviceID, t.Rawresp)
    }
	if err != nil {
		fmt.Fprintf(os.Stderr,"error: %v while marshal json from responce", err)
		return false, err
	}
	if err != nil {
		return false, err
	}
	return false, nil
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
	ch := make(chan string)
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
	for k, v := range tokens[0:]{
		hive = append(hive, TabletClient{ID: v, Token: v, DeviceID: k+1})
	}
	fmt.Printf("we have %d tokens \n", len(hive))
	for _, v := range hive {
		wg.Add(1)
		go v.GetRide(&wg, &cfg, ch)
        if str, ok := <-ch; ok {
		fmt.Printf("from chan: %s \n", str)
	}
	}
	
	secs := time.Since(start).Seconds()
	fmt.Printf("we all done with: %.5fs \n", secs)
}

func getTokens(path string) (tokens []string, err error) {
	tokens = make([]string, 0, 0)
    content, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error while open file with tokens", err)
		return nil, err
	}
	tokens = strings.Split(string(content), ",")
	return tokens, nil
}

func getChatset(responce *http.Response) string {
	contentType := responce.Header.Get("Content-Type")
	if contentType == "" {
		return "UTF-8"
	}
	idx := strings.Index(contentType, "charset:")
	if idx == -1 {
		return "UTF-8"
	}
	return strings.Trim(contentType[idx:], " ")
}
