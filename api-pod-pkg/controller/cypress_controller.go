package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"bitbucket.com/testing-cypress-server/server1/pkg/config"
	"bitbucket.com/testing-cypress-server/server1/pkg/model"
	"bitbucket.com/testing-cypress-server/server1/pkg/utils"
)

var testFailed bool = false

func UploadReports(hubTimestamp, chunkTimestamp string) error {

	filesToUpload, err := utils.GetAbsoluteFilePaths(utils.PATH_TO_ALLURE_RESULTS)
	if err != nil {
		log.Printf("error getting absolute paths: [%v]", err)
		return err
	}

	log.Printf("Uploading report to s3..")
	err = UploadReportsToS3UsingGo(filesToUpload, fmt.Sprintf("%s/%s/%s", utils.BASE_S3_REPORT_FOLDER, hubTimestamp, chunkTimestamp))
	if err != nil {
		log.Printf("error uploading reports to s3: [%v]", err)
		return err
	}

	log.Printf("Report uploaded successfully to S3\n")

	return nil

}

func RunCypressTests(pathToSpec, pathToConfig, browser, hubTimestamp string) (string, error) {

	cmd := exec.Command("cypress", "run", "--spec", pathToSpec, "--config-file", pathToConfig, "--browser", browser, "--headless")

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("error getting stdout pipe: [%v]", err)
		return "error getting stdout pipe", err
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("error getting stderr pipe: [%v]", err)
		return "error getting stderr pipe", err
	}

	cmd.Start()

	// for logging in realtime
	go utils.StartLogging(stdOut, false)
	go utils.StartLogging(stdErr, true)

	err = cmd.Wait()

	// cypress cmds couldnt run the tests
	// TODO: we want the tests to run again
	if err != nil {
		log.Printf("Test failed for spec: %v, config: %v with error: %v\n", pathToSpec, pathToConfig, err)
	}

	log.Printf("Deleting instances of %v", browser)
	// delete all instances of chrome browser
	cmd = exec.Command("sh", "-c", fmt.Sprintf("pkill %s 2>&1 || echo \"Failed to kill Chrome processes\"", browser))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	if err != nil {
		log.Printf("error in killing instances of %v", browser)
	} else {
		log.Printf("Killed all instances of %v", browser)
	}
	result := fmt.Sprintf("Cypress Test ran for spec: %v, config: %v", pathToSpec, pathToConfig)

	currChunkTime := strconv.FormatInt(time.Now().Unix(), 10)

	log.Printf("Cypress cmd ran successfully: %v", result)

	log.Printf("UPloading report...\n")
	err = UploadReports(hubTimestamp, currChunkTime)

	if err != nil {
		log.Printf("error uploading report: %v\n", err)
		return "", err
	}

	log.Printf("removing reports directory..")
	err = utils.DeleteReportsDirectory()
	if err != nil {
		log.Printf("error removing reports directory: [%v]", err)
		return result, err
	}

	return result, nil
}

func startNewmanTest(pathToCol, pathToEnv, hubTimestamp string) (string, error) {
	cmd := exec.Command("newman", "run", pathToCol, "-e", pathToEnv, "-r", "htmlextra")

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("error getting stdout pipe: [%v]", err)
		return "error getting stdout pipe", err
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("error getting stderr pipe: [%v]", err)
		return "error getting stderr pipe", err
	}

	cmd.Start()

	// for logging in realtime
	go utils.StartLogging(stdOut, false)
	go utils.StartLogging(stdErr, true)

	err = cmd.Wait()

	if err != nil {
		log.Printf("Test failed for postman collection: %v, environment: %v with error: %v\n", pathToCol, pathToEnv, err)
	}

	result := fmt.Sprintf("Newman Test ran for collection: %v, environment: %v", pathToCol, pathToEnv)
	//currChunkTime := strconv.FormatInt(time.Now().Unix(), 10)  = to be used for uploading report with timestamp

	log.Printf("Cypress cmd ran successfully: %v", result)

	log.Printf("UPloading report...\n")
	//Have to implement report uploading here

	return result, nil
}

func StartTest(w http.ResponseWriter, r *http.Request) {
	serverName := utils.GetServerName()
	log.Printf("Connected with %v", serverName)

	var request model.RequestModel
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("can't decode request json: [%v]", err)
		http.Error(w, "Invalid request payload", http.StatusInternalServerError)
		return
	}

	log.Printf("processing req: [%v]", request)

	var podRedisModel model.PodRedisModel
	podRedisModel.RequestType = "update_server_busy"
	podRedisModel.ServerName = serverName
	config.UpdateRedis(podRedisModel)

	defer func() {
		podRedisModel.RequestType = "update_server_free"
		config.UpdateRedis(podRedisModel)
	}()

	if request.Component == "ui" {
		pathToSpec := utils.GetCombinedSpecs(request.SpecFile)
		pathToConfig := utils.PATH_TO_CONFIG + request.ConfigFile

		log.Printf("starting cypress test for spec: [%v] and config: [%v]", pathToSpec, pathToConfig)

		_, err = RunCypressTests(pathToSpec, pathToConfig, request.Browser, request.Timestamp)
		if err != nil {
			log.Printf("Error : %v", err)
			http.Error(w, fmt.Sprintf("Error running Cypress tests: %s", err), http.StatusInternalServerError)
			return
		}

		response := model.CypressResponseModel{RequestId: request.RequestId, Module: request.Module, Environment: request.Environment, Component: request.Component, Timestamp: request.Timestamp, SpecFile: request.SpecFile, ConfigFile: request.ConfigFile, Browser: request.Browser}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			log.Printf("error encoding json: [%v]", err)
			http.Error(w, fmt.Sprintf("error encdoding json: %s", err), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		log.Printf("sending json response: [%v]", string(jsonResponse))
		w.Write(jsonResponse)

	} else if request.Component == "api" {
		currDir, err := os.Getwd()
		if err != nil {
			log.Printf("error getting absolute path of collection and envConfig : %v", err)
		}
		pathToCol := currDir + "/collections/" + request.Collection
		pathToEnv := currDir + "/environment/" + request.ConfigFile

		_, err = startNewmanTest(pathToCol, pathToEnv, request.Timestamp)
		if err != nil {
			log.Printf("Error : %v", err)
			http.Error(w, fmt.Sprintf("Error running postman tests: %s", err), http.StatusInternalServerError)
			return
		}
	} else {
		log.Printf("invalid component name in request")
		return
	}

}
