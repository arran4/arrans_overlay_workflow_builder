package util

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func DownloadUrlToTempFile(url string) (string, error) {
	// Create a temporary file
	file, err := os.CreateTemp("", "download-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Temp file close issue: %s", err)
		}
	}(file)
	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("File download close issue: %s", err)
		}
	}(response.Body)

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", fmt.Errorf("writing %s to file %s: %v", url, file.Name(), err)
	}

	return file.Name(), nil
}
