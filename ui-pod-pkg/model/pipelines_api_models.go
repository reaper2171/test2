package model

// the request sent to the kubectl api for creating a new pod
type CreatePodRequestModel struct {
	ImageName     string `json:"image_name"`
	PodName       string `json:"pod_name"`
	ContainerName string `json:"container_name"`
	NodePort      int    `json:"node_port"`
	ContainerPort int    `json:"container_port"`
}

// list of file names and its base64 encoded forms to store in the cons-results folder
type DataReportModel struct {
	Files []DataModel `json:"files"`
}

// a particular file with its name and base64 encoded form from s3
type DataModel struct {
	Name    string `json:"name"`
	Details string `json:"details"`
}

// the 2d list of array called chunks, each chunk has many feature files inside hence the 2d array
type ChunkModelFromRequest struct {
	Files [][]string `json:"files"`
}

// each pod will have its nodeport, hence it contains a list of pods with its nodeports
type CypressPodInfoList struct {
	PodsInfo []CypressPodsInfo `json:"pods_info"`
	NodeIP   string            `json:"node_ip"`
}

// info for a pod with its name and its nodeport
type CypressPodsInfo struct {
	PodName  string `json:"pod_name"`
	NodePort int    `json:"node_port"`
}

// we will send tags as searchText and a filepath where it will be searched in
type TagSearchRequest struct {
	SearchText []string `json:"search_text"`
	FilePath   string   `json:"file_path"`
}

// data model for the suite.json file and unmarshalling it
type Time struct {
	Start    int64 `json:"start"`
	Stop     int64 `json:"stop"`
	Duration int   `json:"duration"`
}

type TestData struct {
	Name                string   `json:"name"`
	Uid                 string   `json:"uid"`
	ParentUid           string   `json:"parentUid"`
	Status              string   `json:"status"`
	Time                Time     `json:"time"`
	Flaky               bool     `json:"flaky"`
	NewFailed           bool     `json:"newFailed"`
	NewPassed           bool     `json:"newPassed"`
	NewBroken           bool     `json:"newBroken"`
	RetriesCount        int      `json:"retriesCount"`
	RetriesStatusChange bool     `json:"retriesStatusChange"`
	Parameters          []string `json:"parameters"`
	Tags                []string `json:"tags"`
}

type SuiteData struct {
	SuiteName string     `json:"name"`
	Tests     []TestData `json:"children"`
	SuiteUUID string     `json:"uid"`
}

type SuiteReport struct {
	Name     string      `json:"name"`
	Uid      string      `json:"uid"`
	Children []SuiteData `json:"children"`
}
