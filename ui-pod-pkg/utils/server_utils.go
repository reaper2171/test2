package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"hub/pkg/model"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func IsPortAvailable(port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}

func GetRandomPort() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(64511) + 1024
}

func GetAbsolutePath() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}
	return currentDir, nil
}

func GetFeatureFileSize(featureFiles []string, chunksModel *model.ChunksModel) error {
	for _, file := range featureFiles {
		var chunkSizeModel model.ChunksSizeModel
		fileInfo, err := os.Stat(file)
		if err != nil {
			log.Printf("error getting file stats: [%v]", err)
			return err
		}
		chunkSizeModel.Size = int(fileInfo.Size())
		chunkSizeModel.Specfile = file
		chunksModel.Chunks = append(chunksModel.Chunks, chunkSizeModel)
	}
	return nil
}

// currently every feature file has "ui-test-automation-master/" as its base directory
// this code removes that, also since we need e2e instead of integration, we replace that as well
func RemoveBaseDirectory(chunkModel model.ChunksModel) model.ChunksModel {
	ABS_PATH, err := GetAbsolutePath()
	if err != nil {
		log.Printf("error getting abs path: %v", err)
		return model.ChunksModel{}
	}
	var result model.ChunksModel
	for _, files := range chunkModel.Chunks {
		// TODO: dynamic slicing required
		length := len(ABS_PATH) + len(UI_TEST_AUTOMATION_MASTER) + 1
		newName := files.Specfile[length:] //default value = 43, can change based on the location of the ui-automation directory
		// replacedName := strings.Replace(newName, "cypress/integration", "cypress/e2e", 1)
		//replacedName := strings.Replace(newName, "cypress/integration", "cypress/e2e", 1)
		result.Chunks = append(result.Chunks, model.ChunksSizeModel{Specfile: newName, Size: files.Size})
	}
	return result
}

// it extracts the feature files from __.txt generated after searching for tags in the feature files
func ParseGrepOutputFromFile(filename string) ([]string, error) {
	var results []string

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("error opening file: %v: [%v]", filename, err)
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 0 {
			results = append(results, strings.TrimSpace(parts[0]))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("error scanning files: [%v]", err)
		return nil, err
	}

	return results, nil
}

// Normal Greedy Approach for chunking
func GreedyChunking(chunksModel *model.ChunksModel, maxLength int, chunks *model.ChunkModelFromRequest) {

	currentSize := 0
	var currentlist []string

	for i := 0; i < len(chunksModel.Chunks); i++ {
		fileSize := chunksModel.Chunks[i].Size
		fileName := chunksModel.Chunks[i].Specfile

		if fileSize+currentSize > maxLength {
			if len(currentlist) > 0 {
				chunks.Files = append(chunks.Files, currentlist)
			}

			// to set the list empty
			var newList []string
			currentlist = newList
			currentlist = append(currentlist, fileName)
			currentSize = fileSize
		} else {
			currentlist = append(currentlist, fileName)
			currentSize += fileSize
		}

	}
	if len(currentlist) > 0 {
		chunks.Files = append(chunks.Files, currentlist)
	}

}

// runs a bash script to search for list of feature file based on tags provided and stores the result in a .txt file
func RunTagSearchScript(searchDir, outputFile string, tags []string) error {
	if len(tags) == 0 {
		log.Printf("empty tags list")
		return fmt.Errorf("at least one tag must be provided")
	}

	PROJECT_ROOT, err := GetAbsolutePath()
	if err != nil {
		log.Printf("error getting abs path: %v", err)
		return err
	}

	// Create a command string with the provided tags
	cmdArgs := []string{"./search_tags.sh", searchDir, outputFile}
	cmdArgs = append(cmdArgs, tags...)

	// Command to run the Bash script
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = PROJECT_ROOT

	// Capture the script's output
	output, err := cmd.CombinedOutput()

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err != nil {
		return fmt.Errorf("error running script: %v", err)
	}

	// Print the script's output
	log.Println(string(output))

	return nil
}

// Running a script to get absolute path of the failed cases to be run
func RunTestCaseSearchScript(searchDir, outputFile string, failedtests []string) error {
	if len(failedtests) == 0 {
		log.Printf("empty testcases list")
		return fmt.Errorf("at least one testcase must be provided")
	}

	PROJECT_ROOT, err := GetAbsolutePath()
	if err != nil {
		log.Printf("error getting abs path: %v", err)
		return err
	}

	// Create a command string with the provided testnames array
	cmdArgs := []string{"./search_testnames.sh", searchDir, outputFile}
	cmdArgs = append(cmdArgs, failedtests...)

	// Command to run the Bash script
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = PROJECT_ROOT

	// Capture the script's output
	output, err := cmd.CombinedOutput()

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err != nil {
		return fmt.Errorf("error running script: %v", err)
	}

	// Print the script's output
	log.Println(string(output))

	return nil
}

// TODO: make session based ports
func OpenAllureReport(reportsDirectory, jobID string) (string, error) {
	port := GetRandomPort()
	for !IsPortAvailable(port) {
		port = GetRandomPort()
	}

	localhost := "192.168.6.149"

	log.Printf("port available: %v", port)

	cmd := exec.Command("allure", "open", reportsDirectory, "-h", localhost, "-p", strconv.Itoa(port))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Println("starting cmd")

	go func() error {
		err := cmd.Run()
		log.Println(stderr.String())

		if err != nil {
			log.Printf("error running cmd: %v", err)
			return err
		}
		return nil
	}()

	log.Println("cmd ran")

	return fmt.Sprintf("http://%v:%v", localhost, port), nil
}

// runs the allure generate cmd in terminal to generate allure reports
func RunAllureReports(reportsDirectory, allureResultDirectory string) error {

	if err := os.MkdirAll(allureResultDirectory, 0755); err != nil {
		log.Printf("Error creating %v folder: [%v]", allureResultDirectory, err)
		return err
	}

	cmd := exec.Command("allure", "generate", reportsDirectory, "--clean", "-o", allureResultDirectory)
	err := cmd.Run()
	if err != nil {
		log.Printf("error creating allure reports: [%v]", err)
		return err
	}
	log.Printf("successfully created allure reports")
	return nil
}

// deletes a folder with filePath
func DeleteFolder(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		log.Printf("error deleting folder - %v: [%v]", filePath, err)
		return err
	}
	return nil
}

// deletes a fole with filePath
func DeleteFile(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		log.Printf("error deleting file - %v: [%v]", filePath, err)
		return err
	}
	return nil
}

// adds the dynamic pod list from kubernetes to the map
func AddServerNameToMap(serverMap map[string]string, podsInfoList model.CypressPodInfoList) {
	nodeIP := podsInfoList.NodeIP
	for _, pod := range podsInfoList.PodsInfo {
		serverMap[pod.PodName] = fmt.Sprintf("http://%v:%v/run-cypress", nodeIP, pod.NodePort)
	}
}

// downloads the current request test result files in the local in a consolidated folder to run allure reports
func SaveReportsLocally(files model.DataReportModel) error {
	if err := os.MkdirAll(CONSOLIDATED_REPORTS_FOLDER_NAME, 0755); err != nil {
		log.Println("Error creating destination folder:", err)
		return err
	}

	// Loop through the list of files and decode and save each one
	for _, file := range files.Files {

		filePath := fmt.Sprintf("./%v/%v", CONSOLIDATED_REPORTS_FOLDER_NAME, file.Name)
		err := os.WriteFile(filePath, []byte(file.Details), 0644)
		if err != nil {
			log.Printf("Error writing file %s: %v\n", file.Name, err)
			return err
		} else {
			log.Printf("File %s downloaded and saved to %s\n", file.Name, filePath)
		}
	}
	return nil
}

// check all the folders inside allure-reports for that jobID
func CheckJobFolder(rootDir, jobID string) (string, error) {
	var reportDirectoryPath string
	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v\n", path, err)
			return err
		}

		if info.IsDir() && info.Name() == jobID {
			log.Printf("Folder with job ID %s found at: %s\n", jobID, path)
			reportDirectoryPath = path
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		log.Printf("Error walking the directory: %v\n", err)
		return "", err
	}
	return reportDirectoryPath, nil
}

// function to appending parentUuid to suite.json using the cypress request from redis
func AppendToJsonReport(request model.CypressRequestModel) error {
	dirPath := ALLURE_RESULTS_FOLDER_LOCAL + "/data/"
	fileName := "suites.json"
	fullPath := filepath.Join(dirPath, fileName)

	fileContent, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("Error reading file suite.json: %v", err)
		return err
	}

	var ReportData model.SuiteReport
	err = json.Unmarshal(fileContent, &ReportData)
	if err != nil {
		log.Printf("Error unmarshaling suite.json: %v", err)
		return err
	}

	// travesing through the report data and modifying parentUid for each match from the redis hubRequest data
	for _, test := range request.Spec.Test {
		for i := range ReportData.Children {
			if ReportData.Children[i].SuiteName == test.SuiteName {
				for j := range ReportData.Children[i].Tests {
					if ReportData.Children[i].Tests[j].Name == test.TestName {
						ReportData.Children[i].Tests[j].ParentUid = test.UUID
					}
				}
			}
		}
	}

	newReportData, err := json.Marshal(ReportData)
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return err
	}

	updatedFilePath := filepath.Join(dirPath, "new_suites.json")
	err = os.WriteFile(updatedFilePath, newReportData, 0644)
	if err != nil {
		log.Printf("Error writing updated file: %v", err)
		return err
	}
	log.Printf("Updated suites.json report saved to: %v", updatedFilePath)

	return nil
}

// function for running python script to create failedcases.json from suites.json
func RunPythonScirpt() error {
	pythonScript := "./jenkinsScript.py"

	cmd := exec.Command("python3", pythonScript)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running Python script: %v", err)
		return err
	}
	log.Printf(string(output))
	log.Printf("Successfully ran the python script jenkinsScript.py")
	return nil
}

// function to process suites.json, FailSuits.json and passcases.json
func ProcessingReports() error {
	// moving suites.json file to final folder
	var source string = "allure-results/data/new_suites.json"
	var destination string = "final/suites.json"
	cmd := exec.Command("mv", source, destination)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("failed to move files: %v", err)
		return err
	}

	// appending all the passcases.json and failedcases.json to from different pod runs
	folderPath := "./cons-reports"
	failedReport, err := os.Create("./cons-reports/failedcases.json")
	if err != nil {
		log.Printf("error creating failedcases.json file: %v", err)
		return err
	}
	passedReport, err := os.Create("./final/passcases.json")
	if err != nil {
		log.Printf("error creating passcases.json file: %v", err)
		return err
	}
	defer func() {
		failedReport.Close()
		passedReport.Close()
	}()

	failedReportData := "["
	passReportData := "["
	files, err := os.ReadDir(folderPath)
	if err != nil {
		log.Printf("error reading directory for finding failcases.json and passcases.json: %v", err)
		return err
	}
	for _, file := range files {
		failMatch, err := regexp.MatchString("failedcases.json", file.Name())
		if err != nil {
			log.Printf("error in finding failescases.json files in cons-reports: %v", err)
			return err
		}
		if failMatch {
			failFileContent, err := os.ReadFile(file.Name())
			if err != nil {
				log.Printf("error reading a failedcases.json file: %v", err)
				return err
			}
			failedReportData = failedReportData + string(failFileContent) + ","
		}

		passMatch, err := regexp.MatchString("passcases.json", file.Name())
		if err != nil {
			log.Printf("error in finding passcases.json files in cons-reports: %v", err)
			return err
		}
		if passMatch {
			passFileContent, err := os.ReadFile(file.Name())
			if err != nil {
				log.Printf("error reading a passcases.json file: %v", err)
				return err
			}
			passReportData = passReportData + string(passFileContent) + ","
		}
	}
	failedReportData = failedReportData + "]"
	passReportData = passReportData + "]"

	err = os.WriteFile(failedReport.Name(), []byte(failedReportData), 0644)
	if err != nil {
		log.Printf("error writing failedcases.json file: %v", err)
		return err
	}
	err = os.WriteFile(passedReport.Name(), []byte(passReportData), 0644)
	if err != nil {
		log.Printf("error writing passcases.json file: %v", err)
		return err
	}

	// zipping new_suites.json, FailSuits.json and passedcases.json in final dir
	zipcmd := exec.Command("zip", "-r", "final.zip", "final")
	zipcmd.Stdout = os.Stdout
	zipcmd.Stderr = os.Stderr
	err = zipcmd.Run()
	if err != nil {
		log.Printf("error zipping dir: %v", err)
		return err
	}
	return nil
}

// function for hitting upload api
func UploadFile(filePath, targetURL string, params map[string]string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("myFile", filePath)
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	// Add other form fields
	for key, value := range params {
		_ = writer.WriteField(key, value)
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", targetURL, body)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	fmt.Println("Response Status:", response.Status)
	fmt.Println("Response Body:", response.Body)

	return nil
}
