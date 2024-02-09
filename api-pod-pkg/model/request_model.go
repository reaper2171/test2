package model

import (
	"encoding/json"
	"log"
)

type RequestModel struct {
	RequestId   string   `json:"request_id"`
	Component   string   `json:"component"`
	Module      string   `json:"module"`
	Environment string   `json:"environment"`
	Timestamp   string   `json:"time_stamp"`
	Collection  string   `json:"collection"`
	SpecFile    []string `json:"spec_file"`
	ConfigFile  string   `json:"config_file"`
	Browser     string   `json:"browser"`
}

func (r *RequestModel) JSONMarshal() ([]byte, error) {
	res, err := json.Marshal(r)
	if err != nil {
		log.Printf("error in json marshal: [%v]", err)
		return []byte{}, err
	}
	return res, nil
}
