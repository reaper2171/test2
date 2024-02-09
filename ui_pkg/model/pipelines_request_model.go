package model

type DataModel struct {
	Name    string `json:"name"`
	Details string `json:"details"`
}

type UploadReportModel struct {
	PathToUpload string      `json:"path_to_upload"`
	DataModel    []DataModel `json:"datamodel"`
}
