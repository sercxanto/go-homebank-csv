package parser

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestBarclaycardName(t *testing.T) {
	mw := &barclaycardParser{}
	if mw.GetFormat() != Barclaycard {
		t.Error("Wrong format")
	}
}

func TestBarclaycardParseFileNonExisting(t *testing.T) {
	bc := &barclaycardParser{}
	err := bc.ParseFile("non_existing_file.csv")
	if err == nil {
		t.Error("Non existing file should return error")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != IOError {
			t.Error("Expected IOError")
		}
	} else {
		t.Error("Expected ParserError")
	}
	if bc.GetNumberOfEntries() != 0 {
		t.Error("Entries should be empty")
	}
}

func TestBarclaycardParseFileNokNoHeader(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze_nok_noheader.xlsx")
	bc := &barclaycardParser{}
	err := bc.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != HeaderError {
			t.Errorf("HeaderError expected, got '%s' instead", pError.ErrorType)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(bc.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestBarclaycardParseFileNokNoSheet1(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze_nok_nosheet1.xlsx")
	bc := &barclaycardParser{}
	err := bc.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != HeaderError {
			t.Errorf("HeaderError expected, got '%s' instead", pError.ErrorType)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(bc.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestBarclaycardParseFileNokWrongDate1(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze_nok_wrongdate1.xlsx")
	bc := &barclaycardParser{}
	err := bc.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Errorf("DataParsingError expected, got '%s' instead", pError.ErrorType)
		}
		if pError.Line != 14 {
			t.Errorf("Expected line 14, got %d", pError.Line)
		}
		if pError.Field != "Buchungsdatum(1)/Transaktionsdatum" {
			t.Errorf("Expected field 'Buchungsdatum(1)/Transaktionsdatum', got '%s' instead", pError.Field)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(bc.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestBarclaycardParseFileNokWrongDate2(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze_nok_wrongdate2.xlsx")
	bc := &barclaycardParser{}
	err := bc.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Errorf("DataParsingError expected, got '%s' instead", pError.ErrorType)
		}
		if pError.Line != 14 {
			t.Errorf("Expected line 14, got %d", pError.Line)
		}
		if pError.Field != "Buchungsdatum" {
			t.Errorf("Expected field 'Buchungsdatum', got '%s' instead", pError.Field)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(bc.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestBarclaycardParseFileNokWrongAmount(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze_nok_wrongamount.xlsx")
	bc := &barclaycardParser{}
	err := bc.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Errorf("DataParsingError expected, got '%s' instead", pError.ErrorType)
		}
		if pError.Line != 14 {
			t.Errorf("Expected line 14, got %d", pError.Line)
		}
		if pError.Field != "Betrag" {
			t.Errorf("Expected field 'Betrag', got '%s' instead", pError.Field)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(bc.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestBarclaycardCell(t *testing.T) {
	row := []string{"first", "second"}
	if barclaycardCell(row, 0) != "first" {
		t.Errorf("Expected 'first', got '%s'", barclaycardCell(row, 0))
	}
	if barclaycardCell(row, 1) != "second" {
		t.Errorf("Expected 'second', got '%s'", barclaycardCell(row, 1))
	}
	// Columns beyond the row are reported as empty instead of panicking
	if barclaycardCell(row, 2) != "" {
		t.Errorf("Expected empty string, got '%s'", barclaycardCell(row, 2))
	}
	if barclaycardCell(row, 14) != "" {
		t.Errorf("Expected empty string, got '%s'", barclaycardCell(row, 14))
	}
	if barclaycardCell([]string{}, 0) != "" {
		t.Errorf("Expected empty string for empty row")
	}
}

func TestIsEmptyBarclaycardRow(t *testing.T) {
	if !isEmptyBarclaycardRow([]string{}) {
		t.Error("Empty row should be reported as empty")
	}
	if !isEmptyBarclaycardRow([]string{"", "  ", ""}) {
		t.Error("Row with blank columns only should be reported as empty")
	}
	if isEmptyBarclaycardRow([]string{"", "value"}) {
		t.Error("Row with a value should not be reported as empty")
	}
}

func TestParseBarclaycardAmountWithThousandsSeparator(t *testing.T) {
	amount, err := parseBarclaycardAmount("+2.456,54 €")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if amount != 2456.54 {
		t.Errorf("Expected 2456.54, got %f", amount)
	}
}

func TestBarclaycardParseFileOk(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze.xlsx")
	mw := &barclaycardParser{}
	if err := mw.ParseFile(fpath); err != nil {
		t.Error(err)
	}
}

// Rows can be shorter than the header as excelize omits trailing empty cells,
// see the fixture Umsaetze_shortrows.xlsx
func TestBarclaycardParseFileShortRows(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze_shortrows.xlsx")
	bc := &barclaycardParser{}
	if err := bc.ParseFile(fpath); err != nil {
		t.Fatal(err)
	}

	// The "vorgemerkt" entry without "Buchungsdatum" and the empty row are skipped
	if bc.GetNumberOfEntries() != 3 {
		t.Fatalf("Expected 3 entries, got %d", bc.GetNumberOfEntries())
	}

	expectedDescriptions := []string{"DetailA", "", "DetailB"}
	for i, expected := range expectedDescriptions {
		if bc.entries[i].description != expected {
			t.Errorf("Entry %d: expected description '%s', got '%s'",
				i, expected, bc.entries[i].description)
		}
	}

	expectedDates := []time.Time{
		time.Date(2020, 9, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2020, 9, 19, 0, 0, 0, 0, time.UTC),
		time.Date(2020, 9, 12, 0, 0, 0, 0, time.UTC),
	}
	for i, expected := range expectedDates {
		if !bc.entries[i].transactionDate.Equal(expected) {
			t.Errorf("Entry %d: expected transaction date %s, got %s",
				i, expected, bc.entries[i].transactionDate)
		}
	}

	expectedValues := []float64{-64.14, -15.00, -3.98}
	for i, expected := range expectedValues {
		if bc.entries[i].value != expected {
			t.Errorf("Entry %d: expected value %f, got %f", i, expected, bc.entries[i].value)
		}
	}
}

func TestBarclaycardConvertRecord(t *testing.T) {
	m := &barclaycardRecord{
		transactionDate: time.Date(2014, 2, 1, 0, 0, 0, 0, time.UTC),
		bookingDate:     time.Date(2014, 3, 2, 0, 0, 0, 0, time.UTC),
		value:           10.0,
		description:     "description",
	}
	h := m.convertRecord()
	if h.amount != m.value {
		t.Error("Amount does not match")
	}
	if h.date != "2014-02-01" {
		t.Errorf("Date does not match. h.date: %s, m.date: %s", h.date, m.transactionDate)
	}
	if h.info != m.description {
		t.Error("Info does not match")
	}
	if h.payment != 1 {
		t.Error("Payment does not match")
	}
	if h.payee != "" {
		t.Error("Payee does not match")
	}
	if h.memo != "" {
		t.Error("Memo does not match")
	}
	if h.category != "" {
		t.Error("Category does not match")
	}
	if h.tags != "" {
		t.Error("Tags does not match")
	}
	if h.amount != m.value {
		t.Error("Amount does not match")
	}
}

func TestBarclaycardConvertToHomebank(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze.xlsx")
	b := &barclaycardParser{}
	err := b.ParseFile(fpath)
	if err != nil {
		t.Error(err)
	}
	tmpDir := t.TempDir()
	tmpFilepath := filepath.Join(tmpDir, "output.csv")

	err = b.ConvertToHomebank(tmpFilepath)
	if err != nil {
		t.Error(err)
	}

	expected := filepath.Join("testfiles", "barclaycard", "Umsaetze.csv")
	if !areFilesEqual(expected, tmpFilepath) {
		t.Error("Files are not equal")
	}
}

func TestBarclaycardConvertToHomebankShortRows(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze_shortrows.xlsx")
	b := &barclaycardParser{}
	err := b.ParseFile(fpath)
	if err != nil {
		t.Error(err)
	}
	tmpDir := t.TempDir()
	tmpFilepath := filepath.Join(tmpDir, "output.csv")

	err = b.ConvertToHomebank(tmpFilepath)
	if err != nil {
		t.Error(err)
	}

	expected := filepath.Join("testfiles", "barclaycard", "Umsaetze_shortrows.csv")
	if !areFilesEqual(expected, tmpFilepath) {
		t.Error("Files are not equal")
	}
}
