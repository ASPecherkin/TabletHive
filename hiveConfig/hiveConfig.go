package hiveConfig


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