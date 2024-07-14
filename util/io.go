package util

import (
	"fmt"
	"io"
	"log"
	"os"
)

func SaveReaderToTempFile(reader io.Reader) (string, error) {
	// Create a temporary file
	file, err := os.CreateTemp("", "extracted-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Temp file close issue: %s", err)
		}
	}(file)
	_, err = io.Copy(file, reader)
	if err != nil {
		return "", fmt.Errorf("writing to file: %v: %s", file.Name(), err)
	}

	return file.Name(), nil
}

type ReaderCloser struct {
	Closer func() error
	io.Reader
}

func (oc *ReaderCloser) Close() error {
	return oc.Closer()
}
