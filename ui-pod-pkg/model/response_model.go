package model

type Job struct {
	ID   string `json:"id"`
	Date string `json:"date"`
}

type JobResponseModel struct {
	Jobs []Job `json:"jobs"`
}

type SelectedDates struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Hash      string `json:"hash"`
}

type CypressResponseModel struct {
	RequestId   string   `json:"request_id"`
	Module      string   `json:"module"`
	Environment string   `json:"environment"`
	Component   string   `json:"component"`
	Timestamp   string   `json:"time_stamp"`
	SpecFile    []string `json:"spec_file"`
	ConfigFile  string   `json:"config_file"`
	Browser     string   `json:"browser"`
}
