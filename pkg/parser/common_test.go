package parser

import (
	"os"
)

func areFilesEqual(file1, file2 string) bool {
	file1Data, err := os.ReadFile(file1)
	if err != nil {
		return false
	}
	file2Data, err := os.ReadFile(file2)
	if err != nil {
		return false
	}
	return string(file1Data) == string(file2Data)
}
