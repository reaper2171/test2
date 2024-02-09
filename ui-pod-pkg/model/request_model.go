package model

import (
	"encoding/json"
	"log"
	"net/http"
)

// it contains all the feature files with its name and its size
type ChunksModel struct {
	Chunks []ChunksSizeModel `json:"chunks"`
}

// for test uuid
type Test struct {
	FeatureName string `json:"feature_name"`
	SuiteName   string `json:"suite_name"`
	TestName    string `json:"test_name"`
	UUID        string `json:"uuid"`
	JobID       string `json:"jobid"`
}

// check if the request is a tag or contains the whole folder
type Spec struct {
	IsTag   bool     `json:"is_tag"`
	Tags    []string `json:"tags"`
	Folders []string `json:"folders"`
	Test    []Test   `json:"tests"`
}

// the request model which comes to the hub for testing
type CypressRequestModel struct {
	RequestID   string `json:"request_id"`
	Module      string `json:"module"`
	Environment string `json:"environment"`
	Component   string `json:"component"`
	ConfigFile  string `json:"config_file"`
	Spec        Spec   `json:"spec"`
	Browser     string `json:"browser"`
}

// a cancellation request with a request id to cancel
type CancelRequestModel struct {
	RequestId   string `json:"request_id"`
	Module      string `json:"module"`
	Environment string `json:"environment"`
	Component   string `json:"component"`
}

// this contains each feature file and its size
type ChunksSizeModel struct {
	Specfile string `json:"spec_file"`
	Size     int    `json:"size"`
}

// this is what goes to each pod for processing of request
type PodRequestModel struct {
	RequestId      string   `json:"request_id"`
	Module         string   `json:"module"`
	Environment    string   `json:"environment"`
	Component      string   `json:"component"`
	SpecFile       []string `json:"spec_file"`
	ConfigFile     string   `json:"config_file"`
	Browser        string   `json:"browser"`
	TimeStamp      string   `json:"time_stamp"`
	ResponseWriter http.ResponseWriter
}

func (r *PodRequestModel) JSONMarshal() ([]byte, error) {
	res, err := json.Marshal(r)
	if err != nil {
		log.Printf("error in json marshal: [%v]", err)
		return []byte{}, err
	}
	return res, nil
}

func (c *CypressRequestModel) JSONMarshal() ([]byte, error) {
	jsonVal, err := json.Marshal(c)
	if err != nil {
		log.Printf("error parsing json: [%v]", err)
		return []byte{}, err
	}
	return jsonVal, nil
}
