package parser

import (
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The line numbers count empty lines and the lines of a field spanning
// several lines
func TestReadCSVRecordsLines(t *testing.T) {
	content := "\n" + // line 1: empty
		"a;b\n" + // line 2
		"\n" + // line 3: empty
		"c;\"first\n" + // line 4: field spanning lines 4 and 5
		"second\";d\n" + // line 5: the field "d" starts here
		"e;f\n" // line 6
	records, err := readCSVRecords(newSemicolonReader(content))
	if err != nil {
		t.Fatal(err)
	}
	expected := []csvRecord{
		{fields: []string{"a", "b"}, lines: []int{2, 2}},
		{fields: []string{"c", "first\nsecond", "d"}, lines: []int{4, 4, 5}},
		{fields: []string{"e", "f"}, lines: []int{6, 6}},
	}
	if len(records) != len(expected) {
		t.Fatalf("Expected %d records, got %d: %v", len(expected), len(records), records)
	}
	for i := range expected {
		if !slices.Equal(records[i].fields, expected[i].fields) || !slices.Equal(records[i].lines, expected[i].lines) {
			t.Errorf("Record %d: expected %v, got %v", i, expected[i], records[i])
		}
	}
}

// A malformed CSV file is reported as IOError with the line of the error
func TestReadCSVRecordsSyntaxError(t *testing.T) {
	content := "a;b\n" +
		"\n" +
		"c;d\"e\n" // line 3: a quote in an unquoted field
	_, err := readCSVRecords(newSemicolonReader(content))
	var pError *ParseError
	if !errors.As(err, &pError) {
		t.Fatalf("Expected ParseError, got %v", err)
	}
	if pError.Type != IOError || pError.Line != 3 {
		t.Errorf("Expected IOError in line 3, got %v", err)
	}
	var csvErr *csv.ParseError
	if !errors.As(err, &csvErr) {
		t.Errorf("Expected csv.ParseError in the error chain, got %v", err)
	}
}

// The line of a malformed value is the line in the file, also after empty
// lines and a field spanning several lines
func TestParseFileLineCountsAllLines(t *testing.T) {
	content := "wallet,currency,category,datetime,money,description\n" +
		"\n" +
		"Wallet,EUR,Category,2024-01-02 10:00:00,-12.34,\"multi\nline\"\n" +
		"Wallet,EUR,Category,02.01.2024 10:00:00,-12.34,Description\n" // line 5
	fpath := filepath.Join(t.TempDir(), "moneywallet.csv")
	if err := os.WriteFile(fpath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	err := mustNew(t, MoneyWallet).ParseFile(fpath)
	var pError *ParseError
	if !errors.As(err, &pError) || pError.Type != DataParsingError || pError.Line != 5 {
		t.Errorf("Expected DataParsingError in line 5, got %v", err)
	}
}

func newSemicolonReader(content string) *csv.Reader {
	r := csv.NewReader(strings.NewReader(content))
	r.Comma = ';'
	r.FieldsPerRecord = -1
	return r
}
