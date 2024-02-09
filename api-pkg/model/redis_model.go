package model

import (
	"encoding/json"
	"log"
)

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
