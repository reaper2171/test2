package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"bitbucket.com/testing-cypress-server/server1/pkg/model"
	"bitbucket.com/testing-cypress-server/server1/pkg/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// filesToUpload will have list of file paths
// FIXME: deprecate this, using Go func to upload now
func UploadReportsToS3(filesToUpload []string, pathToUpload string) error {
	encodedFiles, err := utils.EncodeFileToBase64(filesToUpload)
	if err != nil {
		log.Printf("error encoding files to upload: [%v]", err)
		return err
	}
	uploadData := model.UploadReportModel{
		PathToUpload: pathToUpload,
		DataModel:    encodedFiles,
	}

	requestBody, err := json.Marshal(uploadData)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return err
	}

	resp, err := http.Post(utils.PIPELINES_API_IP+utils.UPLOAD_TO_S3_ROUTE, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Request failed with status code:", resp.StatusCode)
		return err
	}
	log.Printf("successfully uploaded file to s3")
	return nil
}

// This method will save all files of the given path passesd in filesToUpload using Go
// Without using any API
func UploadReportsToS3UsingGo(filesToUpload []string, pathToUpload string) error {

	listDataModel, err := utils.EncodeFile(filesToUpload)
	if err != nil {
		log.Printf("error encoding files to upload: [%v]", err)
		return err
	}

	var uploadFileClassModel model.UploadReportModel
	uploadFileClassModel.PathToUpload = pathToUpload
	uploadFileClassModel.DataModel = listDataModel

	awsRegion := "ap-south-1"
	awsAccessKeyID := "AKIAUFA3CXEC2XJTBZ5G"
	awsSecretAccessKey := "xLkgZBFNT51Co1l1olcV2ElQ4+tmLGYpQ6QcBHJH"
	bucketName := "docker-testing"
	s3ObjectKey := uploadFileClassModel.PathToUpload // Replace with the desired S3 object key

	// Create an AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})
	if err != nil {
		log.Fatalf("Error creating AWS session: %v", err)
	}

	// Create an S3 client
	svc := s3.New(sess)

	// Open the file to be uploaded
	var lengthOfListFile = len(uploadFileClassModel.DataModel)
	for i := 0; i < lengthOfListFile; i++ {

		data := uploadFileClassModel.DataModel[i].Details
		fileContent := []byte(data)

		s3ObjectKey = s3ObjectKey + "/" + filepath.Base(uploadFileClassModel.DataModel[i].Name)

		// Specify the parameters for the S3 upload
		params := &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(s3ObjectKey),
			Body:   strings.NewReader(string(fileContent)),
		}

		// Upload the file to S3
		_, err = svc.PutObject(params)
		if err != nil {
			log.Fatalf("Error uploading file to S3: %v", err)
		}

		fmt.Printf("File '%s' uploaded to S3 bucket '%s' with key '%s'\n", uploadFileClassModel.DataModel[i].Name, bucketName, s3ObjectKey)
		s3ObjectKey = uploadFileClassModel.PathToUpload
	}

	log.Printf("successfully uploaded file to s3")
	return nil
}
