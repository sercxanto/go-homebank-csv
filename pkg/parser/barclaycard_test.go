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

func TestBarclaycardParseFileNok(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze_notok.xlsx")
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

func TestBarclaycardParseFileOk(t *testing.T) {
	fpath := filepath.Join("testfiles", "barclaycard", "Umsaetze.xlsx")
	mw := &barclaycardParser{}
	if err := mw.ParseFile(fpath); err != nil {
		t.Error(err)
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
