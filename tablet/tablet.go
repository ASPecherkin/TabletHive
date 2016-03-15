package tablet

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