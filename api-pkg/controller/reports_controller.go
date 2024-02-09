package controller

import (
	"fmt"
	"hub/pkg/model"
	"hub/pkg/utils"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func CheckTodayReportDate(date string) bool {

	absPath, err := utils.GetAbsolutePath()
	if err != nil {
		log.Printf("error getting abs path: %v", err)
		return false
	}

	reportDirectoryToday := fmt.Sprintf("%v/%v", absPath, utils.LOCAL_DIRECTORY_TODAY)

	log.Printf("reportdirectorytoday: %v", reportDirectoryToday)

	// reading the allure report directory pass the local folder
	fileInfos, err := os.ReadDir(reportDirectoryToday)
	if err != nil {
		log.Printf("error : %v", err)
		return false
	}

	for _, fileInfo := range fileInfos {
		folderName := fileInfo.Name()
		log.Printf("foldername: %v", folderName)

		if folderName == date {
			return true
		}

	}

	// delete the current folder name today..
	// err = os.RemoveAll(reportDirectoryToday)
	// if err != nil {
	// 	log.Printf("Error in deletion of folder: %v", err)
	// 	return false
	// }

	return false
}

// Saving Allure reports to S3 and Saving it locally as well
func SaveReportDirectoryToS3AndLocal(requestId string) error {

	awsRegion := "ap-south-1"
	awsAccessKeyID := "AKIAUFA3CXEC2XJTBZ5G"
	awsSecretAccessKey := "xLkgZBFNT51Co1l1olcV2ElQ4+tmLGYpQ6QcBHJH"
	bucketName := "docker-testing"
	folder_path := utils.ALLURE_REPORT_FOLDER_S3 // Replace with the desired S3 object key

	absPath, err := utils.GetAbsolutePath()
	if err != nil {
		log.Printf("error getting abs path: %v", err)
		return err
	}

	localFolder := fmt.Sprintf("%v/%v", absPath, utils.ALLURE_RESULTS_FOLDER_LOCAL) // Specify the folder location where allure report created

	currentTime := time.Now()
	formattedDate := currentTime.Format("2006-01-02")
	fmt.Println("Current date with year:", formattedDate)

	// check weather given date folder present or not if not then we need to created it after deleting the current folder
	CheckTodayReportDate(formattedDate)

	// Initialize a new session using environment variables or AWS credentials file
	sess, er := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})
	if er != nil {
		log.Fatalf("Error creating AWS session: %v", er)
		return er
	}

	// Create a new S3 client
	svc := s3.New(sess)

	folder_path += formattedDate + "/" + requestId + "/"

	// Walk through the local folder and upload files to S3
	err = filepath.Walk(localFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			file, _ := os.Open(path)
			defer file.Close()

			// Construct the destination S3 key based on the local file path
			key := folder_path + strings.TrimPrefix(path, localFolder)

			// Upload the file to S3
			_, err := svc.PutObject(&s3.PutObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(key),
				Body:   file,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error Saving files : %v", err)
		return err
	}
	location := fmt.Sprintf("%v/%v%v/%v", absPath, utils.LOCAL_DIRECTORY_TODAY, formattedDate, requestId)

	err = copyDir(localFolder, location)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Directory copied successfully.")
	}

	return nil
}

func copyDir(src, dest string) error {
	// Ensure the source directory exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	// Create the destination directory
	err = os.MkdirAll(dest, srcInfo.Mode())
	if err != nil {
		return err
	}

	// Read the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Recursively copy each file and subdirectory
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			// If it's a subdirectory, recursively copy it
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			// If it's a file, copy it
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return destFile.Close()
}

// For Fetching All Allure reports from given range of dates
func GetAllAllureReportsOfGivenRange(startDate string, endDate string, hash string) (model.JobResponseModel, error) {

	// declare S3 details here...
	awsRegion := utils.AWS_REGION
	awsAccessKeyID := utils.AWS_ACCESS_KEY_ID
	awsSecretAccessKey := utils.AWS_SECRET_ACCESS_KEY
	bucketName := utils.BUCKET_NAME

	var result model.JobResponseModel

	// Create an AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})
	if err != nil {
		log.Fatalf("Error creating AWS session: %v", err)
		return result, err
	}

	absPath, err := utils.GetAbsolutePath()
	if err != nil {
		log.Printf("error getting abs path: %v", err)
		return result, err
	}

	reportDirectorySelected := fmt.Sprintf("%v/%v%v/", absPath, utils.LOCAL_DIRECTORY_SELECTED_DATE, hash)

	// delete the current folder name with given hash if present..
	_ = os.RemoveAll(reportDirectorySelected)

	// Create an S3 client
	svc := s3.New(sess)

	// List objects in the S3 bucket with the specified prefix (path)
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(utils.ALLURE_REPORT_FOLDER_S3), // check weather file path is correct or Result/ should be added before...
	}

	resp, err := svc.ListObjectsV2(input)
	if err != nil {
		log.Fatalf("Error listing objects in S3: %v", err)
		return result, err
	}

	lastFolder := "abcd"

	// Iterate through the objects and fetch their content
	for _, obj := range resp.Contents {
		fileKey := *obj.Key

		//check if key lies in the range from start date and end date
		// Parse the date strings into time.Time objects
		start, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			fmt.Println("Error parsing start date:", err)
			return result, err
		}

		end, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			fmt.Println("Error parsing end date:", err)
			return result, err
		}

		parts := strings.Split(fileKey, "/")

		givenDate, err := time.Parse("2006-01-02", parts[1])
		if err != nil {
			fmt.Println("Error parsing given date:", err)
			return result, err
		}

		fileKey = parts[1]

		// Check if the given date is between the start and end dates
		if (givenDate.Equal(start) || givenDate.After(start)) && (givenDate.Equal(end) || givenDate.Before(end)) && lastFolder != fileKey {
			lastFolder = fileKey
			fmt.Printf("The given date is within the range. %v", fileKey)

			destinationFilePath := reportDirectorySelected + fileKey

			s3Path := utils.ALLURE_REPORT_FOLDER_S3 + fileKey + "/"

			directoryPath := reportDirectorySelected + parts[1]

			log.Printf("File Location : %v", directoryPath)

			// creating the folder
			err := os.MkdirAll(directoryPath, 0755)
			if err != nil {
				fmt.Println("Error:", err)
				return result, err
			}

			rest, err := FetchObjectsOfSelectedDate(sess, bucketName, s3Path, destinationFilePath)
			if err != nil {
				fmt.Println("Error fetching folder from S3:", err)
				return result, err
			}
			result.Jobs = append(result.Jobs, rest.Jobs...)
		}
	}

	log.Printf("All files name fetched successfully")
	return result, nil
}

func FetchObjectsOfSelectedDate(sess *session.Session, bucketName, s3Path, localPath string) (model.JobResponseModel, error) {
	downloader := s3manager.NewDownloader(sess)
	svc := s3.New(sess)

	lastFolder := "abcd"

	var finalListOfJobId model.JobResponseModel
	var jobs []model.Job

	listObjectsInput := &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(s3Path),
	}

	err := svc.ListObjectsPages(listObjectsInput,
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			for _, obj := range page.Contents {
				s3Key := *obj.Key

				parts := strings.Split(s3Key, "/")

				if lastFolder != parts[2] {
					var job model.Job
					job.Date = parts[1]
					job.ID = parts[2]
					jobs = append(jobs, job)
				}

				lastFolder = parts[2]

				localFile := filepath.Join(localPath, strings.TrimPrefix(s3Key, s3Path))

				if err := os.MkdirAll(filepath.Dir(localFile), 0755); err != nil {
					fmt.Println("Error creating local directory:", err)
					return false
				}

				file, err := os.Create(localFile)
				if err != nil {
					fmt.Println("Error creating local file:", err)
					return false
				}
				defer file.Close()

				_, err = downloader.Download(file, &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(s3Key),
				})
				if err != nil {
					fmt.Println("Error downloading S3 object:", err)
					return false
				}
			}
			return true
		})

	finalListOfJobId.Jobs = jobs

	log.Printf("size of jobs : %v", len(jobs))

	if err != nil {
		return finalListOfJobId, err
	}
	return finalListOfJobId, nil
}
