package controller

import (
	"fmt"
	"hub/pkg/model"
	"hub/pkg/utils"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// this searches the tags using the grep cmd of linux, since we already have all the feature files in our local
// we can search using grep rather than hitting api calls
func GetChunkFromGREP(requestID, filePath string, searchText []string) (model.ChunkModelFromRequest, error) {
	outputFileName := fmt.Sprintf("%v.txt", requestID)
	err := utils.RunTagSearchScript(utils.CYPRESS_FEATURE_SEARCH_DIRECTORY, outputFileName, searchText)
	if err != nil {
		log.Printf("error searching tags: [%v]", err)
		return model.ChunkModelFromRequest{}, err
	}

	PROJECT_ROOT := os.Getenv("PROJECT_ROOT")
	if err != nil {
		log.Printf("error getting abs path: %v", err)
		return model.ChunkModelFromRequest{}, err
	}
	log.Printf("abs path: %v", PROJECT_ROOT)
	fileNameWithPath := fmt.Sprintf("%v/%v", PROJECT_ROOT, outputFileName)
	featureFiles, err := utils.ParseGrepOutputFromFile(fileNameWithPath)
	if err != nil {
		log.Printf("error parsing files from output file: [%v]", err)
		return model.ChunkModelFromRequest{}, err
	}

	var featureFilesWithAbsolutePath []string
	for _, val := range featureFiles {
		absPath := fmt.Sprintf("%v/%v", PROJECT_ROOT, val)
		featureFilesWithAbsolutePath = append(featureFilesWithAbsolutePath, absPath)
	}

	// log.Printf("absolute path: [%v]", featureFilesWithAbsolutePath)

	var chunksModel model.ChunksModel
	err = utils.GetFeatureFileSize(featureFilesWithAbsolutePath, &chunksModel)
	if err != nil {
		log.Printf("error getting file sizes: [%v]", err)
		return model.ChunkModelFromRequest{}, err
	}

	// log.Printf("chunksModel: [%v]", chunksModel)

	//sorting the chunksModel on the Size based
	sortBySize := func(i, j int) bool {
		return chunksModel.Chunks[i].Size < chunksModel.Chunks[j].Size
	}

	sort.Slice(chunksModel.Chunks, sortBySize)
	// log.Printf("sorted chunksModel: [%v]", chunksModel)

	MAX_CHUNK_SIZE := chunksModel.Chunks[len(chunksModel.Chunks)-1].Size + 1

	log.Printf("MAX SIZE OF THE CHUNK : %v", MAX_CHUNK_SIZE)

	removedBaseDirectoryFiles := utils.RemoveBaseDirectory(chunksModel)
	// log.Printf("removedbasedirectory files: [%v]", removedBaseDirectoryFiles)

	var chunks model.ChunkModelFromRequest
	utils.GreedyChunking(&removedBaseDirectoryFiles, MAX_CHUNK_SIZE, &chunks)
	// utils.DeleteFile(fileNameWithPath)

	log.Printf("chunks after algorithm: %v", chunks)
	log.Printf("no. of chunks- %v", len(chunks.Files))

	return chunks, nil
}

// Getting chunks for rerun with spec files with absolute path and greedy chunking
func GetChunkForRerun(requestID, filePath string, failedTests []string) (model.ChunkModelFromRequest, error) {
	outputFileName := fmt.Sprintf("%v.txt", requestID)

	//Here will the script for searching the failed testcases in the cypress test directory
	err := utils.RunTestCaseSearchScript(utils.CYPRESS_FEATURE_SEARCH_DIRECTORY, outputFileName, failedTests)
	if err != nil {
		log.Printf("error searching tests: [%v]", err)
		return model.ChunkModelFromRequest{}, err
	}

	// PROJECT_ROOT := os.Getenv("PROJECT_ROOT")
	// if err != nil {
	// 	log.Printf("error getting abs path: %v", err)
	// 	return model.ChunkModelFromRequest{}, err
	// }
	// log.Printf("abs path: %v", PROJECT_ROOT)
	fileNameWithPath := fmt.Sprintf("%v", outputFileName)
	featureFiles, err := utils.ParseGrepOutputFromFile(fileNameWithPath)
	if err != nil {
		log.Printf("error parsing files from output file: [%v]", err)
		return model.ChunkModelFromRequest{}, err
	}

	var featureFilesWithAbsolutePath []string
	for _, val := range featureFiles {
		absPath := fmt.Sprintf("%v", val)
		featureFilesWithAbsolutePath = append(featureFilesWithAbsolutePath, absPath)
	}

	// log.Printf("absolute path: [%v]", featureFilesWithAbsolutePath)

	var chunksModel model.ChunksModel
	err = utils.GetFeatureFileSize(featureFilesWithAbsolutePath, &chunksModel)
	if err != nil {
		log.Printf("error getting file sizes: [%v]", err)
		return model.ChunkModelFromRequest{}, err
	}

	// log.Printf("chunksModel: [%v]", chunksModel)

	//sorting the chunksModel on the Size based
	sortBySize := func(i, j int) bool {
		return chunksModel.Chunks[i].Size < chunksModel.Chunks[j].Size
	}

	sort.Slice(chunksModel.Chunks, sortBySize)
	// log.Printf("sorted chunksModel: [%v]", chunksModel)

	MAX_CHUNK_SIZE := chunksModel.Chunks[len(chunksModel.Chunks)-1].Size + 1

	log.Printf("MAX SIZE OF THE CHUNK : %v", MAX_CHUNK_SIZE)

	removedBaseDirectoryFiles := utils.RemoveBaseDirectory(chunksModel)
	// log.Printf("removedbasedirectory files: [%v]", removedBaseDirectoryFiles)

	var chunks model.ChunkModelFromRequest
	utils.GreedyChunking(&removedBaseDirectoryFiles, MAX_CHUNK_SIZE, &chunks)
	utils.DeleteFile(fileNameWithPath)

	log.Printf("chunks after algorithm: %v", chunks)
	log.Printf("no. of chunks- %v", len(chunks.Files))

	return chunks, nil
}

func GetTestReportFromS3UsingGo(file_path string) error {
	// declare S3 details here...
	awsRegion := utils.AWS_REGION
	awsAccessKeyID := utils.AWS_ACCESS_KEY_ID
	awsSecretAccessKey := utils.AWS_SECRET_ACCESS_KEY
	bucketName := utils.BUCKET_NAME

	absFilePath := fmt.Sprintf("%v/%v", utils.BASE_S3_REPORT_FOLDER, file_path)

	// Create an AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})
	if err != nil {
		log.Fatalf("Error creating AWS session: %v", err)
		return err
	}

	// Create an S3 client
	svc := s3.New(sess)

	// List objects in the S3 bucket with the specified prefix (path)
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(absFilePath), // check weather file path is correct or Result/ should be added before...
	}

	resp, err := svc.ListObjectsV2(input)
	if err != nil {
		log.Fatalf("Error listing objects in S3: %v", err)
		return err
	}

	var pairList []model.DataModel

	// Iterate through the objects and fetch their content
	for _, obj := range resp.Contents {
		fileKey := *obj.Key

		// Get the object (file) from S3
		getInput := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKey),
		}

		getResp, err := svc.GetObject(getInput)
		if err != nil {
			log.Fatalf("Error fetching file from S3: %v", err)
			return err
		}

		// Read the content of the file
		fileContent, err := io.ReadAll(getResp.Body)
		if err != nil {
			log.Fatalf("Error reading file content: %v", err)
			return err
		}

		var Data model.DataModel
		Data.Name = filepath.Base(fileKey)
		Data.Details = string(fileContent)

		if len(Data.Details) > 1 {
			pairList = append(pairList, Data)
		}

	}

	var finalResult model.DataReportModel

	finalResult.Files = pairList

	log.Printf("Report Contents Size: [%v]", len(finalResult.Files))

	err = utils.SaveReportsLocally(finalResult)
	if err != nil {
		log.Printf("error in saving reports to local directory: [%v]", err)
		return err
	}
	return nil

}
