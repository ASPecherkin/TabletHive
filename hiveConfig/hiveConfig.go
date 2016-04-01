package hiveConfig

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

// HiveConfig gather all of needed configs
type HiveConfig struct {
	ServerURL    string               `json:"server"`
	TokensPath   string               `json:"token_file_path"`
	DeviceCodes  string               `json:"device_codes_file_path"`
	SecondsDelay int64                `json:"delay"`
	Endpoints    map[string]Endpoints `json:"endpoints"`
}

// Endpoints stores all urls for requests
type Endpoints struct {
	URL   string `json:"url"`
	Delay int    `json:"delay"`
}

// Authtokens stores json represent array of tokens
type Authtokens struct {
	Tokens []string `json:"tokens"`
}

// SadirLogins stores logins
type SadirLogins struct {
	Logins []string `json:"sadira_logins"`
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

// GetLogins read avaible logins and return it
func GetLogins(loginsFile string) (logins SadirLogins, err error) {
	jsonDoc, err := ioutil.ReadFile(loginsFile)
	if err != nil {
		log.Fatalf("Could not read config file: %s ", err)
		return
	}
	err = json.Unmarshal(jsonDoc, &logins)
	return
}
