package model

import (
	"encoding/json"
	"log"
)

type PodRedisModel struct {
	ServerName  string `json:"server_name"`
	RequestType string `json:"request_type"`
}

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

type RedisRequestReportModel struct {
	TotalChunks      int    `json:"total_chunks"`
	CurrentCompleted int    `json:"current_completed"`
	TimeStamp        string `json:"timestamp"`
}

type RedisEnvironmentModule struct {
	RequestId string   `json:"request_id"`
	Tags      []string `json:"tags"`
}

type RedisEnvironmentModuleList struct {
	List_Tags []RedisEnvironmentModule `json:"list_tags"`
}

func (report *RedisRequestReportModel) JSONMarshal() ([]byte, error) {
	bytes, err := json.Marshal(report)
	if err != nil {
		log.Printf("error marshalling redis report map: [%v]", err)
		return []byte{}, err
	}
	return bytes, nil
}

func (report *RedisEnvironmentModuleList) JSONMarshal() ([]byte, error) {
	bytes, err := json.Marshal(report)
	if err != nil {
		log.Printf("error marshalling redis report map: [%v]", err)
		return []byte{}, err
	}
	return bytes, nil
}

// model for the hub request model
type HubRequestModel struct {
	RequestId string              `json:"request_id"`
	Request   CypressRequestModel `json:"request"`
}
