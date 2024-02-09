package utils

import (
	"encoding/base64"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"bitbucket.com/testing-cypress-server/server1/pkg/model"
)

func GetServerName() string {
	serverName := os.Getenv(SERVER_NAME_ENV)
	return serverName
}

func DeleteReportsDirectory() error {
	cmd := exec.Command("rm", "-r", PATH_TO_ALLURE_RESULTS)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		log.Printf("error removing reports directory: [%v]", err)
		return err
	}
	return nil
}

func EncodeFileToBase64(filePath []string) ([]model.DataModel, error) {
	var encodedFiles []model.DataModel
	for _, files := range filePath {
		fileName := ExtractFileName(files)
		fileData, err := os.ReadFile(files)
		if err != nil {
			log.Printf("Error reading file: %v", err)
			return []model.DataModel{}, err
		}

		// Encode the file data to base64
		encodedData := base64.StdEncoding.EncodeToString(fileData)
		encodedFiles = append(encodedFiles, model.DataModel{Name: fileName, Details: encodedData})
	}
	return encodedFiles, nil
}

func EncodeFile(filePath []string) ([]model.DataModel, error) {
	var encodedFiles []model.DataModel
	for _, files := range filePath {
		fileName := ExtractFileName(files)
		fileData, err := os.ReadFile(files)
		if err != nil {
			log.Printf("Error reading file: %v", err)
			return []model.DataModel{}, err
		}

		encodedFiles = append(encodedFiles, model.DataModel{Name: fileName, Details: string(fileData)})
	}
	return encodedFiles, nil
}

func ExtractFileName(path string) string {
	return filepath.Base(path)
}

func GetAbsoluteFilePaths(folderPath string) ([]string, error) {
	var fileList []string

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			absPath, _ := filepath.Abs(path)
			fileList = append(fileList, absPath)
		}
		return nil
	})

	if err != nil {
		log.Println("Error getting absolute paths: ", err)
		return []string{}, err
	}
	log.Printf("got absolute paths")
	return fileList, nil
}

func GetCombinedSpecs(specFiles []string) string {
	l := len(specFiles)
	specs := ""
	for i, spec := range specFiles {
		// specs += PATH_TO_SPEC
		specs += spec
		if i == (l - 1) {
			break
		}
		specs += ","
	}
	return specs
}
