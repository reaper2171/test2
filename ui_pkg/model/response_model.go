package model

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

type PodRedisModel struct {
	ServerName  string `json:"server_name"`
	RequestType string `json:"request_type"`
}
