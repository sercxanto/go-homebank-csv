package parser

import (
	"os"
	"strings"
	"testing"
)

// mustNew returns a new parser for the given format and fails the test if
// there is none
func mustNew(t *testing.T, format SourceFormat) Parser {
	t.Helper()
	p, err := New(format)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

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
