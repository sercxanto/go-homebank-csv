package parser

import (
	"errors"
	"io"
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
	if len(lines) != 1 {
		t.Fatalf("Expected one record, got %d lines: %q", len(lines), lines)
	}
	for i, line := range lines {
		if fields := strings.Count(line, homebankFieldSeparator) + 1; fields != 8 {
			t.Errorf("Line %d has %d fields instead of 8: %s", i+1, fields, line)
		}
	}

	expected := "2024-01-01;0;info,with,separator;Payee, Name;memo,with line break;-1.500000;cat,egory;tag,one"
	if lines[0] != expected {
		t.Errorf("Expected:\n%s\ngot:\n%s", expected, lines[0])
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

func TestParseErrorMessage(t *testing.T) {
	cases := []struct {
		err      ParseError
		expected string
	}{
		{ParseError{Type: HeaderError}, "HeaderError"},
		{ParseError{Type: DataParsingError, Line: 3, Field: "Betrag"},
			"DataParsingError in line 3 in field name 'Betrag'"},
		{ParseError{Type: IOError, Err: errors.New("permission denied")},
			"IOError: permission denied"},
		{ParseError{Type: DataParsingError, Line: 5, Field: "Betrag", Err: errors.New("invalid syntax")},
			"DataParsingError in line 5 in field name 'Betrag': invalid syntax"},
	}

	for _, c := range cases {
		if got := c.err.Error(); got != c.expected {
			t.Errorf("Expected %q, got %q", c.expected, got)
		}
	}
}

func TestParseErrorUnwrap(t *testing.T) {
	cause := errors.New("cause")
	var err error = &ParseError{Type: IOError, Err: cause}
	if !errors.Is(err, cause) {
		t.Error("Expected the cause to be found with errors.Is")
	}

	err = &ParseError{Type: HeaderError}
	if errors.Unwrap(err) != nil {
		t.Error("Expected no underlying error")
	}
}

// Every parser keeps the cause of a failed file access, so that callers can
// tell e.g. a missing file from a file without permission
func TestParseFileNonExistingKeepsCause(t *testing.T) {
	for _, format := range GetSourceFormats() {
		err := GetParser(format).ParseFile(filepath.Join("testfiles", "non_existing_file"))
		var pError *ParseError
		if !errors.As(err, &pError) || pError.Type != IOError {
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

// testHomebankRecords returns a single record for the tests of
// writeHomeBankRecords
func testHomebankRecords() []homebankRecord {
	return []homebankRecord{{date: "2024-01-01", memo: "memo", amount: -1.5}}
}

// HomeBank warns about a header line on import, so only the data lines are
// written
func TestWriteHomeBankCSVNoHeader(t *testing.T) {
	var b strings.Builder
	if err := writeHomeBankCSV(&b, testHomebankRecords()); err != nil {
		t.Fatal(err)
	}
	expected := "2024-01-01;0;;;memo;-1.500000;;\n"
	if b.String() != expected {
		t.Errorf("Expected %q, got %q", expected, b.String())
	}
}

// A successful write leaves only the output file, no temporary file
func TestWriteHomeBankRecordsNoTempFileLeft(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "output.csv")
	if err := writeHomeBankRecords(testHomebankRecords(), fpath); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "output.csv" {
		t.Errorf("Expected only 'output.csv' in the output directory, got %v", entries)
	}
}

// An existing output file is replaced, as before the temporary file was used
func TestWriteHomeBankRecordsReplacesExistingFile(t *testing.T) {
	fpath := filepath.Join(t.TempDir(), "output.csv")
	if err := os.WriteFile(fpath, []byte("old content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeHomeBankRecords(testHomebankRecords(), fpath); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(fpath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(content), "2024-01-01;0;") {
		t.Errorf("Expected the existing file to be replaced, got %q", content)
	}
}

// If the output file cannot be put in place, here because a directory has
// its name, the temporary file is removed again
func TestWriteHomeBankRecordsRemovesTempFileOnError(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "output.csv")
	if err := os.Mkdir(fpath, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := writeHomeBankRecords(testHomebankRecords(), fpath); err == nil {
		t.Error("Expected an error when the output path is a directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "output.csv" {
		t.Errorf("Expected the temporary file to be removed, got %v", entries)
	}
}

// Files which have the name of a temporary file, e.g. left behind by an
// aborted run, neither block writing the output file nor are they changed
func TestWriteHomeBankRecordsKeepsExistingTempFiles(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "output.csv")
	tempPaths := []string{
		fpath + ".tmp", // name used by earlier versions
		fpath + ".123456.tmp",
	}
	for _, tempPath := range tempPaths {
		if err := os.WriteFile(tempPath, []byte("unrelated\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if err := writeHomeBankRecords(testHomebankRecords(), fpath); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	content, err := os.ReadFile(fpath)
	if err != nil || !strings.HasPrefix(string(content), "2024-01-01;0;") {
		t.Errorf("Expected the output file to be written, got %q, %v", content, err)
	}
	for _, tempPath := range tempPaths {
		content, err := os.ReadFile(tempPath)
		if err != nil || string(content) != "unrelated\n" {
			t.Errorf("Expected '%s' to be kept, got %q, %v", tempPath, content, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(tempPaths)+1 {
		t.Errorf("Expected no further temporary file, got %v", entries)
	}
}

// If the temporary file cannot be created, here because the output
// directory does not exist, the error is passed on
func TestWriteHomeBankRecordsMissingDirectory(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "missing", "output.csv")

	if err := writeHomeBankRecords(testHomebankRecords(), fpath); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Expected fs.ErrNotExist, got %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected an empty directory, got %v", entries)
	}
}

// failingWriter fails all writes after the first okWrites ones
type failingWriter struct {
	okWrites int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.okWrites <= 0 {
		return 0, errors.New("write failed")
	}
	w.okWrites--
	return len(p), nil
}

// An error while writing a record is passed on instead of being ignored
func TestWriteHomeBankCSVWriteError(t *testing.T) {
	for okWrites := range 1 {
		err := writeHomeBankCSV(&failingWriter{okWrites: okWrites}, testHomebankRecords())
		if err == nil {
			t.Errorf("Expected the write error to be returned after %d successful writes", okWrites)
		}
	}
}

// If writing the content fails, neither the output file nor the temporary
// file is left behind and the error is passed on
func TestWriteFileAtomicallyWriteError(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "output.csv")
	writeErr := errors.New("write failed")

	err := writeFileAtomically(fpath, func(w io.Writer) error {
		if _, err := io.WriteString(w, "partial content"); err != nil {
			return err
		}
		return writeErr
	})

	if !errors.Is(err, writeErr) {
		t.Errorf("Expected the write error, got %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected an empty output directory, got %v", entries)
	}
}
