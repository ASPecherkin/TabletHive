package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"log"
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
    ServerUrl string `json:"url"`
    TokensPath string `json:"token_path"`
}

type TabletClient struct {
	ID       string
	Token    string
	Responce string
}

// GetRide create connect amd get ride fpr that token
func (t *TabletClient) GetRide(deviceCode int, wg *sync.WaitGroup) (bool, error) {
	defer wg.Done()
	client := &http.Client{}
	url := strings.Join([]string{"url"}, string(deviceCode))
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	request.Header.Add("HTTP-AUTH-TOKEN", t.Token)
	responce, err := client.Do(request)
    if err != nil {
        log.Fatal(err)
    }
	var buf [1024]byte
   	reader := responce.Body
	n, err := reader.Read(buf[0:])
    defer responce.Body.Close()
	if err != nil {
		fmt.Println("error reading from responce Body", err)
        return false,err
	}
	resp := string(buf[0:n])
	answer, err := json.Marshal(resp)
	if err != nil {
		fmt.Println("error while marshal json from responce", err)
        return false,err
	}
	if string(answer) != "" {
		t.Responce = string(answer)
	} else {
		t.Responce = ""
	}
	if err != nil {
		return false, err
	}
	if t.Responce != "" {
		return true, nil
	}
	return false, nil
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write mem profile to file")

func main() {
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
	tokens, err := getTokens("tokens.json")
	hive := make(map[int]TabletClient)
	if err != nil {
		fmt.Println("error while read tokens.json", err)
	}
	for k, v := range tokens {
		hive[k] = TabletClient{ID: v, Token: v}
	}
	for k, v := range hive {
		wg.Add(1)
		go v.GetRide(k, &wg)
	}
	for k,v := range hive {
        if v.Responce != "" {
            fmt.Println(k, v.Responce)
        }   
    }
    fmt.Println("we done here")
}

func getTokens(path string) (tokens []string, err error) {
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
