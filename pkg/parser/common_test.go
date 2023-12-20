package parser

import (
	"os"
	"strings"
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

	// Trim CRs to avoid differences on Windows
	return strings.Trim(string(file1Data), "\r") == strings.Trim(string(file2Data), "\r")
}
