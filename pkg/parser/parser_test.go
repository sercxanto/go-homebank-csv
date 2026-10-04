package parser

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestSanitizeHomebankField(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"a plain memo", "a plain memo"},
		{"", ""},
		{"Rechnung; Nr 123", "Rechnung, Nr 123"},
		{"a;b;c", "a,b,c"},
		{"line1\r\nline2", "line1 line2"},
		{"line1\nline2", "line1 line2"},
		{"line1\rline2", "line1 line2"},
		{"mixed;value\nwith both", "mixed,value with both"},
	}

	for _, c := range cases {
		got := sanitizeHomebankField(c.input)
		if got != c.expected {
			t.Errorf("Input %q: expected %q, got %q", c.input, c.expected, got)
		}
	}
}

// A separator or a line break inside a text field must not shift the fields of
// the written record
func TestWriteHomeBankRecordsSanitizesFields(t *testing.T) {
	records := []homebankRecord{
		{
			date:     "2024-01-01",
			payment:  0,
			info:     "info;with;separator",
			payee:    "Payee; Name",
			memo:     "memo;with\nline break",
			amount:   -1.5,
			category: "cat;egory",
			tags:     "tag;one",
		},
	}

	fpath := filepath.Join(t.TempDir(), "output.csv")
	if err := writeHomeBankRecords(records, fpath); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(fpath)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected a header and one record, got %d lines: %q", len(lines), lines)
	}
	for i, line := range lines {
		if fields := strings.Count(line, homebankFieldSeparator) + 1; fields != 8 {
			t.Errorf("Line %d has %d fields instead of 8: %s", i+1, fields, line)
		}
	}

	expected := "2024-01-01;0;info,with,separator;Payee, Name;memo,with line break;-1.500000;cat,egory;tag,one"
	if lines[1] != expected {
		t.Errorf("Expected:\n%s\ngot:\n%s", expected, lines[1])
	}
}

// The order of the returned formats has to be stable: it determines the output
// of the "list-formats" command and the order in which GetGuessedParser tries
// the parsers.
func TestGetSourceFormatsOrder(t *testing.T) {
	expected := []SourceFormat{MoneyWallet, Barclaycard, Volksbank, Comdirect, DKB}

	formats := GetSourceFormats()
	if !slices.Equal(formats, expected) {
		t.Errorf("Expected %v, got %v", expected, formats)
	}

	// Repeated calls keep the order. Without sorting this fails, as the
	// iteration order of a map is randomized.
	for i := 0; i < 100; i++ {
		if !slices.Equal(GetSourceFormats(), expected) {
			t.Fatalf("Call %d returned a different order: %v", i, GetSourceFormats())
		}
	}
}

func TestGetParser(t *testing.T) {
	for _, f := range GetSourceFormats() {
		p := GetParser(f)
		if p == nil {
			t.Fatal("Parser not found")
		}
		if p.GetFormat() != f {
			t.Error("Parser mismatch")
		}
	}
	p := GetParser(999999999)
	if p != nil {
		t.Fatal("Expected nil parser")
	}
}

func TestSourceFormatString(t *testing.T) {
	for _, f := range GetSourceFormats() {
		s := SourceFormat(f).String()
		if s == "" || s == "unknown format" {
			t.Errorf("Expected valid string, got: %s", s)
		}
	}
	s := SourceFormat(999999999).String()
	if s != "unknown format" {
		t.Errorf("Expected 'unknown format', got: %s", s)
	}
}

func TestUnmarshalSourceFormatText(t *testing.T) {
	for key, value := range sourceFormats {
		var s SourceFormat
		err := s.UnmarshalText([]byte(value))
		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
		if s != key {
			t.Errorf("Expected: %v, got: %v", key, s)
		}
	}

	var s SourceFormat
	err := s.UnmarshalText([]byte("no valid format"))
	if err == nil {
		t.Error("Expected error")
	}
}

func TestNewSourceFormat(t *testing.T) {
	for _, f := range GetSourceFormats() {
		s := NewSourceFormat(f)
		if s == nil {
			t.Error("Expected non nil pointer")
		}
	}
}

func TestGetGuessedParser(t *testing.T) {

	nilFilepath := filepath.Join("testfiles", "moneywallet", "converted_1.csv")
	p := GetGuessedParser(nilFilepath)
	if p != nil {
		t.Errorf("Expected: nil, got: %v, %s", p, p.GetFormat())
	}

	formats := map[string]SourceFormat{
		filepath.Join("testfiles", "moneywallet", "MoneyWallet_export_1.csv"):                     MoneyWallet,
		filepath.Join("testfiles", "barclaycard", "Umsaetze.xlsx"):                                Barclaycard,
		filepath.Join("testfiles", "volksbank", "Umsaetze_DE12345678901234567890_2023.10.04.csv"): Volksbank,
		filepath.Join("testfiles", "comdirect", "umsaetze_1234567890_20231006_1804.csv"):          Comdirect,
		filepath.Join("testfiles", "dkb", "dkb.csv"):                                              DKB,
	}

	for testfile, format := range formats {
		p := GetGuessedParser(testfile)
		if p == nil {
			t.Errorf("Parser not found for file: %s", testfile)
		}
		if p != nil && p.GetFormat() != format {
			t.Errorf("Parser not correct, expected: %s, got: %s", format, p.GetFormat())
		}
	}
}

func TestParserErrorMessage(t *testing.T) {
	cases := []struct {
		err      ParserError
		expected string
	}{
		{ParserError{ErrorType: HeaderError}, "HeaderError"},
		{ParserError{ErrorType: DataParsingError, Line: 3, Field: "Betrag"},
			"DataParsingError in line 3 in field name 'Betrag'"},
		{ParserError{ErrorType: IOError, Err: errors.New("permission denied")},
			"IOError: permission denied"},
		{ParserError{ErrorType: DataParsingError, Line: 5, Field: "Betrag", Err: errors.New("invalid syntax")},
			"DataParsingError in line 5 in field name 'Betrag': invalid syntax"},
	}

	for _, c := range cases {
		if got := c.err.Error(); got != c.expected {
			t.Errorf("Expected %q, got %q", c.expected, got)
		}
	}
}

func TestParserErrorUnwrap(t *testing.T) {
	cause := errors.New("cause")
	var err error = &ParserError{ErrorType: IOError, Err: cause}
	if !errors.Is(err, cause) {
		t.Error("Expected the cause to be found with errors.Is")
	}

	err = &ParserError{ErrorType: HeaderError}
	if errors.Unwrap(err) != nil {
		t.Error("Expected no underlying error")
	}
}

// Every parser keeps the cause of a failed file access, so that callers can
// tell e.g. a missing file from a file without permission
func TestParseFileNonExistingKeepsCause(t *testing.T) {
	for _, format := range GetSourceFormats() {
		err := GetParser(format).ParseFile(filepath.Join("testfiles", "non_existing_file"))
		var pError *ParserError
		if !errors.As(err, &pError) || pError.ErrorType != IOError {
			t.Errorf("%s: expected IOError, got %v", format, err)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: expected fs.ErrNotExist in the error chain, got %v", format, err)
		}
	}
}

// A malformed value keeps the error of the conversion function as cause
func TestParseFileDataParsingErrorKeepsCause(t *testing.T) {
	err := GetParser(DKB).ParseFile(filepath.Join("testfiles", "dkb", "dkb_nok_wrongbetrag.csv"))
	var numError *strconv.NumError
	if !errors.As(err, &numError) {
		t.Errorf("Expected strconv.NumError in the error chain, got %v", err)
	}
}
