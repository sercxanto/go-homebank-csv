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

	// Trim possible all line endings to avoid differences on Windows
	// and with git autocrlf settings
	return strings.ReplaceAll(string(file1Data), "\r", "") == strings.ReplaceAll(string(file2Data), "\r", "")
}
