package tablet

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	config "github.com/ASPecherkin/TabletHive/hiveConfig"
	"github.com/davecgh/go-spew/spew"
)

//Ride - basic struct for Mobile controller responce
type Ride struct {
	ID        uint        `json:"id"`
	Number    string      `json:"number"`
	Duration  uint        `json:"duration"`
	Distance  float32     `json:"distance"`
	FactRides []FactRides `json:"fact_rides"`
}

// FactRides - struct for json unmarshal FactRides in responce
type FactRides struct {
	ID         uint        `json:"id"`
	TimeStart  string      `json:"time_start"`
	RidePoints []RidePoint `json:"ride_points"`
}

// RidePoint - struct for json unmarshal RidePoint in responce
type RidePoint struct {
	ID          uint    `json:"id"`
	Number      uint    `json:"number"`
	Lat         float32 `json:"lat"`
	Lng         float32 `json:"lng"`
	AddressText string  `json:"address_text"`
	Status      string  `json:"status"`
	Kind        string  `json:"kind"`
	Order       `json:"order"`
}

// Order - struct for json unmarshal Order in responce
type Order struct {
	ID            uint   `json:"id"`
	Status        string `json:"status"`
	ServiceType   string `json:"service_type"`
	ServiceObject `json:"service_object"`
}

// ServiceObject - struct for json unmarshal ServiceObject in responce
type ServiceObject struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	ObjType string `json:"type"`
	TimeT   string `json:"time_t"`
	Phones  string `json:"phones"`
	TimeG1  string `json:"time_g1"`
	TimeG2  string `json:"time_g2"`
}

// Device one unit of hive
type Device struct {
	ID         string
	Token      string
	RespObj    Ride
	Rawresp    string
	StatusCode int
	ch         chan string
}

// GetRide create connect amd get ride for that token
func (t *Device) GetRide(wg *sync.WaitGroup, cfg *config.HiveConfig, res chan Result) error {
	defer wg.Done()
	client := &http.Client{}
	url := strings.Join(append([]string{cfg.ServerURL, cfg.Endpoints["get_rides"].URL, t.ID}), "")
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	request.Header.Add("HTTP-AUTH-TOKEN", t.Token)
	time.Sleep(time.Duration(cfg.Endpoints["get_rides"].Delay))
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
		var answer Ride
		err = json.Unmarshal([]byte(jsonData), &answer)
		if err != nil {
			spew.Printf("err: %s  with token : %v when unmarhal this  \n", err, t)
		}
		t.RespObj, t.Rawresp = answer, string(jsonData)
		return nil
	} else {
		t.Rawresp = string(jsonData)
		return nil
	}
}
