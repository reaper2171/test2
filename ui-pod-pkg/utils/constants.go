package utils

const (
	AWS_REGION                       = "ap-south-1"
	SERVER_NAME_ENV                  = "MY_NAME"
	REDIS_TEST_RESULTS               = "pod-report"
	AWS_ACCESS_KEY_ID                = "AKIAUFA3CXEC2XJTBZ5G"
	AWS_SECRET_ACCESS_KEY            = "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"
	BUCKET_NAME                      = "docker-testing"
	REDIS_MAP_NAME                   = "pod-map"
	REDIS_REQUEST_MAP_NAME           = "request-map"
	REDIS_ENVIRONMENT_MAP_NAME       = "environment-map"
	REDIS_HUB_REQUEST_MAP_NAME       = "hub-request-map"
	KUBERNETES_API_IP                = "http://localhost:8000/"
	TEST_FILES_PATH                  = "cypress/integration/features/A2A"
	CHUNKS_API_ROUTE                 = "chunk-spec-files"
	CREATE_CYPRESS_POD_ROUTE         = "create-pods"
	BASE_S3_REPORT_FOLDER            = "resulttest1"
	GET_CYPRESS_POD_INFO_ROUTE       = "pod-info"
	DELETE_CYPRESS_POD_ROUTE         = "delete-pods"
	CONSOLIDATED_REPORTS_FOLDER_NAME = "cons-reports"
	FETCH_RESULTS_ROUTE              = "fetch-results"
	ALLURE_RESULTS_FOLDER_LOCAL      = "allure-results"
	LOCAL_DIRECTORY_SELECTED_DATE    = "report_directory/selected_date/"
	LOCAL_DIRECTORY_TODAY            = "report_directory/today/"
	ALLURE_REPORT_FOLDER_S3          = "allure-reports/"

	PROJECT_ROOT = "/home/amankumarsingh/hub-spoke/qa-tools"

	// TODO: need to fix such that we dont pass the base ui-test folder in search script
	CYPRESS_FEATURE_SEARCH_DIRECTORY = "ui-test-automation-master/cypress/e2e/features"
	UI_TEST_AUTOMATION_MASTER        = "ui-test-automation-master/"
)
