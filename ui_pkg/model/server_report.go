package model

import (
	"encoding/json"
	"log"
)

type ServerReport struct {
	TestPassed int `json:"test_passed"`
	TestFailed int `json:"test_failed"`
	TotalTest  int `json:"total_tests"`
}

func (report *ServerReport) JsonMarshal() ([]byte, error) {
	json, err := json.Marshal(report)
	if err != nil {
		log.Printf("error marshalling json: [%v]", err)
		return []byte{}, err
	}
	return json, nil
}
