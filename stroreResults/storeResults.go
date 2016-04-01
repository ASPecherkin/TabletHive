package storeResults

import (
	"sync"
)

// HiveResults stores all results of running
type HiveResults struct {
	sync.RWMutex
	When          string   `json:"when"`
	ElapsedTime   float64  `json:"elapsed_time"`
	GetResults    []Result `json:"get_results"`
	UpdateResults []Result `json:"update_results"`
	OthersResults []Result `json:"others_result"`
}

// Result store meta info about every request
type Result struct {
	RequestType   string  `json:"type"`
	AuthToken     string  `json:"token"`
	RequestURL    string  `json:"url"`
	Responce      string  `json:"responce"`
	RequestStatus int     `json:"status_code"`
	ProcessedTime float64 `json:"processed_time"`
}
