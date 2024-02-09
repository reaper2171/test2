/*
 Hub.. It is the heart of our System
 All request comes to the hub and directed to differnet pods
 It accept response from POD and create Allure Report here
 It is connected with Server POD, AWS S3, JGIT and UI
*/

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hub/pkg/config"
	"hub/pkg/model"
	"hub/pkg/utils"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var (
	serverMap        = make(map[string]string)
	podsCreated bool = false
	serverQueue      = make(chan model.PodRequestModel, 100) // Request Channel queue present size is fixed with 100 so atmax 100 chunks can be placed in the serverQueue
	done             = make(chan bool)
	mutex       sync.Mutex
	wg          sync.WaitGroup
	delay       = 500 * time.Millisecond // In some cases delay is required to avoid read/write conflicts basicallt for Redis

	cancelRequest     bool = false // Cancel Request Boolean to handle Cancel function weather to execute or Not!!
	cancelRequestId   string
	cancelEnvironment string
	cancelModule      string
	cancelComponent   string

	module_env string
)

// This function find the free server and route the request to that server
// If all the servers are busy it enqued it back to the ServerQueue
func RouteRequestsToTestServers(queuedRequest model.PodRequestModel) {
	// rw := model.NewResponseWriter(queuedRequest.ResponseWriter)

	// finding All Server Status
	serverName, err, status_code := config.FindReadyServer(utils.REDIS_MAP_NAME)
	if err != nil {
		log.Printf("something went wrong in getting server status: [%v]", err)
	}

	podRequestBytes, err := queuedRequest.JSONMarshal()
	if err != nil {
		log.Printf("error converting model to bytes: [%v]", err)
	}

	// status_code == 0 means one of the server is free to accept the request so sending the request to that server
	if status_code != 1 {
		log.Printf("one of the server is ready - %v\n", serverName)
		log.Printf("sending GET request: %v with request: %v", serverName, queuedRequest)
		serverUrl := serverMap[serverName]

		// API called which sending the request to POD and getting response from the POD
		resp, err := http.Post(serverUrl, "application/json", bytes.NewBuffer(podRequestBytes))
		if err != nil {
			log.Printf("error in POST request to %v: [%v]", serverName, err)
		}
		defer resp.Body.Close()

		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response: %v", err)
		}

		var cypressResponse model.CypressResponseModel
		if err := json.Unmarshal(responseBody, &cypressResponse); err != nil {
			log.Printf("Error decoding response [route req]: %v", err)
		}
		UpdateTestCompletion(cypressResponse)
		// rw.WriteHeader(http.StatusOK)

	} else {
		serverQueue <- queuedRequest
	}
}

// This contain infinite for loop which will be running endlessly
// if cancelRequest is True it means cancell request is called with request id : cancelRequestId
// cancel process is written just after 'for' statement as in select-case one of the chunk is popped out and may be left during the deletion from serverQueue
func ProcessQueue() {
	for {
		if cancelRequest {
			CancelRequestFunction(cancelRequestId, cancelModule, cancelEnvironment, cancelComponent)
		}
		select {
		// since we are dequeuing the requests, if we are not able to process it then we need to queue it again
		case queuedRequest := <-serverQueue:
			go RouteRequestsToTestServers(queuedRequest)
			// again delay is added for allowing time redis to update
			time.Sleep(delay)
		}

	}
}

func CreatePodsAndUpdateStatus() error {
	wg.Add(1)
	defer wg.Done()
	log.Printf("creating pods..")

	err := CreateMultiplePods()
	if err != nil {
		log.Printf("error creating cypress pods: [%v]", err)
		panic("cypress pods not created.. exiting hub server")
	}

	log.Printf("pods created, getting pods ip")

	cypressPodsInfo, err := GetCypressPodsIP()
	if err != nil {
		log.Printf("error getting cypress pods info: [%v]", err)
		panic("cypress info not available.. exiting hub server")
	}

	utils.AddServerNameToMap(serverMap, cypressPodsInfo)

	log.Printf("updating redis status")

	err = config.SetInitialServerStatusInRedis(serverMap, utils.REDIS_MAP_NAME)
	if err != nil {
		log.Printf("error setting initial status in redis: [%v]", err)
		return err
	}
	return nil
}

// This Method Basically add all the chunks to the ServerQueue which is a channel which is continuously running in ProcessQueue func
func AddingRequestToServerQueue(request model.TestRequestModel, specFile []string, w http.ResponseWriter, requestId string, timestamp string) {

	select {
	case serverQueue <- model.PodRequestModel{RequestId: requestId, Environment: request.Environment, Module: request.Module, Component: request.Component, ResponseWriter: w, ConfigFile: request.ConfigFile, SpecFile: specFile, Browser: request.Browser, TimeStamp: timestamp}:
		log.Printf("enqueuing request with id: [%v]", requestId)
	default:
		log.Printf("queue is full, cannot enqueue the request")
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	done <- true

}

func StartTest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Hub connected")

	var request model.TestRequestModel
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("can't decode request json: [%v]", err)
		http.Error(w, "Invalid request payload", http.StatusInternalServerError)
		return
	}

	requestId := uuid.New().String()
	request.RequestID = requestId
	log.Printf("got cypress request: [%v]", request)
	log.Printf("RequestId created: %v", requestId)

	// need to sort tags for later use case of same Environment Module Request Cancellation
	sort.Strings(request.Spec.Tags)
	log.Printf("After Sorting Tags arrangement are : [%v]", request.Spec.Tags)

	// check condition if this request tags already running or not
	if EnvironmentTagsPressence(request) {
		log.Printf("This Request is already running in the System...")
		return
	}

	config.SettingModuleEnvironment(request.Module+":"+request.Environment+":"+request.Component, request.RequestID, request.Spec.Tags)

	mutex.Lock()
	if !podsCreated {

		err = CreatePodsAndUpdateStatus()
		if err != nil {
			log.Printf("error creating pods and updating status: [%v]", err)
			http.Error(w, "Error creating test pods", http.StatusInternalServerError)
			panic("error creating pods.. exiting hub")
		}
		podsCreated = true

		// TODO: call JGit and upload in s3
	}
	mutex.Unlock()
	wg.Wait()

	// Here we are creating a unique Folder Name under which we are going to save the given request Results along with Time Stampso we can fetch this result on time basis
	currentTimeStamp := strconv.FormatInt(time.Now().Unix(), 10)
	currentTimeStamp += "_" + request.RequestID
	log.Printf("Time Samp with Unique Folder Name %v\n", currentTimeStamp)

	if request.Spec.IsTag {

		go ChunkCreation(request, currentTimeStamp, w)

	} else {
		// Setting init status of Spec file
		error := config.SetInitialRequestStatus(requestId, 1, 0, currentTimeStamp)
		if error != nil {
			log.Printf("Error Encounter while setting Request Status In Redis %v", error)
			return
		}
		AddingRequestToServerQueue(request, request.Spec.Folders, w, requestId, currentTimeStamp)
	}

}

// Rerun code
func StartRerunTest(w http.ResponseWriter, r *http.Request) { //have to take array of testnames in body
	fmt.Println("Hub connected")

	var request model.TestRequestModel
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("can't decode request json: [%v]", err)
		http.Error(w, "Invalid request payload", http.StatusInternalServerError)
		return
	}

	if request.Component != "ui" && request.Component == "api" {
		log.Printf("Invalid component type in the request")
		return
	}

	requestId := uuid.New().String()
	request.RequestID = requestId
	log.Printf("got http request: [%v]", request)
	log.Printf("RequestId created: %v", requestId)

	// log.Printf("req.spec.test is : %v", request.Spec.Test)

	//For ui, to check the presence of a request we check list of feature file name
	if request.Component == "ui" {
		//for adding the tests feature file in an array from key value mapping
		var tests []string
		for _, test := range request.Spec.Test {
			tests = append(tests, test.RunFile)
		}

		log.Printf("The test to be run are : [%v]", tests)

		// check condition if this request env is already running or not
		if EnvironmentTagsPressenceForRerun(request) {
			log.Printf("This Request is already running in the System...")
			return
		}
		config.SettingModuleEnvironment(request.Module+":"+request.Environment+":"+request.Component, request.RequestID, tests)
		mutex.Lock()

		//for api, we need to check whether suitname and testname combination is present in the redis env map with requestid
	} else {
		var testdetails []string
		for _, testdetail := range request.Spec.Test {
			testdetails = append(testdetails, testdetail.SuiteName+"_"+testdetail.TestName)
		}

		log.Printf("The test to be run are : [%v]", testdetails)

		// check condition if this request env is already running or not
		if EnvironmentTagsPressenceForRerun(request) {
			log.Printf("This Request is already running in the System...")
			return
		}
		config.SettingModuleEnvironment(request.Module+":"+request.Environment+":"+request.Component, request.RequestID, testdetails)
		mutex.Lock()
	}

	if !podsCreated {

		err = CreatePodsAndUpdateStatus()
		if err != nil {
			log.Printf("error creating pods and updating status: [%v]", err)
			http.Error(w, "Error creating test pods", http.StatusInternalServerError)
			panic("error creating pods.. exiting hub")
		}
		podsCreated = true

		// TODO: call JGit and upload in s3
	}
	mutex.Unlock()
	wg.Wait()

	// Here we are creating a unique Folder Name under which we are going to save the given request Results along with Time Stampso we can fetch this result on time basis
	currentTimeStamp := strconv.FormatInt(time.Now().Unix(), 10)
	currentTimeStamp += "_" + request.RequestID
	log.Printf("Time Samp with Unique Folder Name %v\n", currentTimeStamp)

	if request.Component == "ui" {
		go ChunkCreation(request, currentTimeStamp, w)
	} else if request.Component == "api" {
		go GetPostmanCollection(request)
	}
}

// This Method basically created to handle multiple request simultaniously
// because GetChunksAPI can take time around 15-30 seconds to respond
// So with this below code GetChunkAPI func called at the same time
func ChunkCreation(request model.TestRequestModel, currentTimeStamp string, w http.ResponseWriter) {

	// Creating Chunks From the Files in S3
	log.Printf("chunks Grep HIT for Request Id : %v ", request.RequestID)

	var chunks model.ChunkModelFromRequest
	var err error
	if request.Spec.IsTag {
		chunks, err = GetChunkFromGREP(request.RequestID, utils.CYPRESS_FEATURE_SEARCH_DIRECTORY, request.Spec.Tags)
	} else {
		//for adding the tests feature file in an array from key value mapping
		var tests []string
		for _, test := range request.Spec.Test {
			tests = append(tests, test.RunFile)
		}
		log.Printf("tests in the request: %v", tests)
		chunks, err = GetChunkForRerun(request.RequestID, utils.CYPRESS_FEATURE_SEARCH_DIRECTORY, tests)
	}

	if err != nil {
		log.Printf("error getting chunks from api: [%v]", err)
		http.Error(w, "error getting chunks from api", http.StatusInternalServerError)
		return
	}
	log.Printf("chunks created : [%v]", chunks)

	//Locking this Method to avoid Read/Write Conflict and Setting init status of Chunks data
	mutex.Lock()
	error := config.SetInitialRequestStatus(request.RequestID, len(chunks.Files), 0, currentTimeStamp)
	mutex.Unlock()

	if error != nil {
		log.Printf("Error Encounter while setting Request Status In Redis %v", error)
		return
	}

	for _, innerChunk := range chunks.Files {
		log.Printf("curr chunk: [%v]", innerChunk)
		go AddingRequestToServerQueue(request, innerChunk, w, request.RequestID, currentTimeStamp)
		// delay is required for updating the Redis Dashboard otherwise multiple request goes to the same pod at the same time
		time.Sleep(delay)
	}

}

// Getting the postman collection to run based on the suitenames passed
func GetPostmanCollection(request model.TestRequestModel) {
	tests := request.Spec.Test
	testDetails := []model.TestData{}
	for _, val := range tests {
		var testval model.TestData
		testval.Suitename = val.SuiteName
		testval.Testname = val.TestName
		testDetails = append(testDetails, testval)
	}

	CreatePostmanCollection(testDetails, utils.POSTMAN_SEARCH_DIRECTORY)

}

// Cancel Request API called for cancelling the Request
func CancelRequest(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	var request model.CancelRequestModel
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("can't decode request json: [%v]", err)
		http.Error(w, "Invalid request payload", http.StatusInternalServerError)
		return
	}
	cancelRequest = true
	cancelRequestId = request.RequestId
	cancelEnvironment = request.Environment
	cancelModule = request.Module
	cancelComponent = request.Component
}

// After Completion of Pod this method is called
// Update Redis data of that Request ID
// When all chunks of that request completed create allure report and delete its trace from redis
func UpdateTestCompletion(response model.CypressResponseModel) {

	log.Printf("Number of Request Left in Queue %v", len(serverQueue))
	log.Printf("Response Method called from Hub for RequesId %v\n", response.RequestId)

	// fetch the current completion and total chunk from Redis
	// if return is 0 it means all chunks of that Request Id is completed so generate allure report
	// if return -1 then this Request Id might be Deleted because of Cancel Request
	// if return > 0 means some of the chunks left to complete of that Request Id

	module_env = response.Module + ":" + response.Environment + ":" + response.Component

	numberOfChunkLeft, timeStampId := config.FetchRequestStatus(response.RequestId, module_env)
	if numberOfChunkLeft == 0 {
		log.Printf("Successfull completion of RequestId: %v and TimeStampId: %v ", response.RequestId, timeStampId)

		// fetching Results From S3
		err := GetTestReportFromS3UsingGo(timeStampId)
		//err := GetTestReportFromS3(timeStampId)

		if err != nil {
			log.Printf("error getting reports from s3: [%v]", err)
		}
		err = utils.RunAllureReports(utils.CONSOLIDATED_REPORTS_FOLDER_NAME, utils.ALLURE_RESULTS_FOLDER_LOCAL)
		if err != nil {
			log.Printf("error creating allure reports: [%v]", err)
		}
		log.Printf("successfully created allure reports")

		// Save Allure Report Directory in S3
		err = SaveReportDirectoryToS3AndLocal(response.RequestId)
		if err != nil {
			log.Printf("error uploading report directory to S3: [%v]", err)
		}
		log.Printf("successfully uploaded report directory")

	} else if numberOfChunkLeft < 0 {
		log.Printf("This Request is Deleted from Redis, Might be RequestId : %v is Cancelled", response.RequestId)
	} else {
		log.Printf("Number of Request left : %v for TimeStampId %v ", numberOfChunkLeft, timeStampId)
	}

}

// logic for cancellation of request
// It delete all chunks from the Queue Channel of given Request ID
// After deletion it remove it from the Redis
func CancelRequestFunction(RequestId string, module string, environment string, component string) {
	log.Printf("Cancelation Request Function Called for Request Id : %v \n", RequestId)
	length_of_loop := len(serverQueue)
	log.Printf("Number of Chunks left in the Queue: %v before Cancellation\n", length_of_loop)
	for i := 0; i < length_of_loop; i++ {
		select {
		case queuedRequest := <-serverQueue:
			if queuedRequest.RequestId == RequestId {
				log.Printf("Removing Request ID %v from ServerQueue \n", queuedRequest.RequestId)
			} else {
				log.Printf("enqueuing request with id: %v", queuedRequest.RequestId)
				serverQueue <- queuedRequest
			}
		default:
			log.Printf("empty serverQueue")
		}
	}

	module_env = module + ":" + environment + ":" + component

	log.Printf("Presnet Number of Chunks left in the Queue: %v after Cancellation\n", len(serverQueue))
	err := config.DeleteRedisData(RequestId, module_env)
	if err != nil {
		log.Printf("Somthing went wrong %v", err)
	}
	cancelRequest = false

}

// checking if same environment_module hash a subset of current Tags or not
// We don't want to run those on repeat
func EnvironmentTagsPressence(request model.TestRequestModel) bool {

	module_name := request.Module + ":" + request.Environment + ":" + request.Component

	// firstly need to fetch previously data on which i have to add another request
	list, err := config.FetchModuleEnvironment(module_name)
	if err != nil {
		log.Printf("error marshalling redis report map in Fetching Module Environment Might not be present in the Redis: [%v]", err)
	}

	log.Printf("Checking Environment Pressence for: %v", module_name)

	currTagList := request.Spec.Tags

	for i := 0; i < len(list.List_Tags); i++ {
		value := list.List_Tags[i]
		var count int = 0
		if len(value.Tags) >= len(currTagList) {
			for j := 0; j < len(value.Tags); j++ {
				if currTagList[count] == value.Tags[j] {
					count++
				}
			}
		}
		if count == len(currTagList) {
			return true
		}
	}

	return false

}

// env check for rerun
func EnvironmentTagsPressenceForRerun(request model.TestRequestModel) bool {

	module_name := request.Module + ":" + request.Environment + ":" + request.Component

	// firstly need to fetch previously data on which i have to add another request
	list, err := config.FetchModuleEnvironment(module_name)
	if err != nil {
		log.Printf("error marshalling redis report map in Fetching Module Environment Might not be present in the Redis: [%v]", err)
	}

	log.Printf("Checking Environment Pressence for: %v", module_name)

	//for ui we can take the list of feature files
	//adding all the test in a single array
	var currTagList []string
	for _, test := range request.Spec.Test {
		currTagList = append(currTagList, test.RunFile)
	}

	for i := 0; i < len(list.List_Tags); i++ {
		value := list.List_Tags[i]
		var count int = 0
		if len(value.Tags) >= len(currTagList) {
			for j := 0; j < len(value.Tags); j++ {
				if currTagList[count] == value.Tags[j] {
					count++
				}
			}
		}
		if count == len(currTagList) {
			return true
		}
	}
	return false
}

func GetReports(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	startDate := vars["startDate"]
	endDate := vars["endDate"]

	log.Printf("startDate: %v, endDate: %v", startDate, endDate)

	// TODO: change hash from email password of user
	data, err := GetAllAllureReportsOfGivenRange(startDate, endDate, "1234")
	if err != nil {
		log.Printf("error getting range reports: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	log.Printf("returned jobs: %v", data)

	jsonResponseForJobId, err := json.Marshal(data.Jobs)
	if err != nil {
		log.Printf("error marshalling json: %v", err)
		http.Error(w, "error marshalling json: "+err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonResponseForJobId))

}

// we need to check inside the allure reports folder locally, which will have multiple date folders
// we will check inside each date folder and find that jobId
func GetJobReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]

	absFilePath, err := utils.GetAbsolutePath()
	if err != nil {
		log.Printf("error getting abs path: [%v]", err)
		http.Error(w, "Error getting abs path: "+err.Error(), http.StatusBadRequest)
		return
	}

	reportAbsPath := fmt.Sprintf("%v/report_directory/", absFilePath)

	reportFilePath, err := utils.CheckJobFolder(reportAbsPath, jobID)
	if err != nil {
		log.Printf("error searching job id: %v locally: [%v]", jobID, err)
		http.Error(w, "couldn't find job_id: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("report file path: %v", reportFilePath)

	url, err := utils.OpenAllureReport(reportFilePath, jobID)
	if err != nil {
		log.Printf("error opening allure report: %v", err)
		http.Error(w, "error opening allure report: "+err.Error(), http.StatusBadRequest)
	}

	log.Printf("url: %v", url)

	jsonResponse := fmt.Sprintf("{\"url\": \"%v\"}", url)
	log.Println(reportFilePath)

	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response
	w.Write([]byte(jsonResponse))

	log.Printf("JOb Id : %v", jobID)

}

// fetching all jobId of current day
func GetJobReportToday(w http.ResponseWriter, r *http.Request) {

	currentTime := time.Now()
	formattedDate := currentTime.Format("2006-01-02")
	log.Println("Current date with year:", formattedDate)

	var checker = CheckTodayReportDate(formattedDate)
	log.Printf("checker : %v", checker)

	absPath, err := utils.GetAbsolutePath()
	if err != nil {
		log.Printf("error getting abs path: %v", err)
		http.Error(w, "error getting abs path: "+err.Error(), http.StatusInternalServerError)

		return
	}

	var listOfJobId model.JobResponseModel

	if checker {
		reportDirectory := fmt.Sprintf("%v/%v%v", absPath, utils.LOCAL_DIRECTORY_TODAY, formattedDate)

		dir, err := os.Open(reportDirectory)
		if err != nil {
			log.Println("Error opening directory:", err)
			return
		}
		defer dir.Close()

		// Read the directory entries
		fileInfos, err := dir.Readdir(0)
		if err != nil {
			log.Println("Error reading directory:", err)
			return
		}

		// Iterate over the file info and print the file names
		for _, fileInfo := range fileInfos {
			job := model.Job{ID: fileInfo.Name()}
			listOfJobId.Jobs = append(listOfJobId.Jobs, job)
		}

	}

	// pass this to the user who hit the API
	log.Printf("List Of Job Id : %v", listOfJobId)

	jsonResponseForJobId, err := json.Marshal(listOfJobId.Jobs)
	if err != nil {
		log.Printf("error marshalling json: %v", err)
		http.Error(w, "error marshalling json: "+err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonResponseForJobId))

}

func GetAllureReportsOfSelectedDates(w http.ResponseWriter, r *http.Request) {
	var request model.SelectedDates
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("can't decode request json: [%v]", err)
		http.Error(w, "Invalid request payload", http.StatusInternalServerError)
		return
	}

	var result model.JobResponseModel

	Uid := request.Hash

	result, er := GetAllAllureReportsOfGivenRange(request.StartDate, request.EndDate, Uid)
	if er != nil {
		log.Printf("Error in fetching jobs name : %v", er)
	}

	log.Printf("Results fetched : %v", result)

}
