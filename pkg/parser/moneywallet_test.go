package parser

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestMoneywalletName(t *testing.T) {
	mw := &moneywalletParser{}
	if mw.GetFormat() != MoneyWallet {
		t.Error("Wrong format")
	}
}

func TestMoneywalletParseFileNonExisting(t *testing.T) {
	mw := &moneywalletParser{}
	err := mw.ParseFile("non_existing_file.csv")
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
	if mw.GetNumberOfEntries() != 0 {
		t.Error("Entries should be empty")
	}
}

/*
func TestMoneywalletParseFileNok(t *testing.T) {
	fpath := filepath.Join("testfiles", "moneywallet", "converted_1.csv")
	mw := &moneywalletParser{}
	err := mw.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != HeaderError {
			t.Error("Expected HeaderError")
		}
		if pError.Line != 1 {
			t.Error("Expected HeaderError on first line")
		}
	} else {
		t.Error("Expected ParserError")
	}

	if len(mw.entries) != 0 {
		t.Error("Entries should be empty")
	}
}
*/

func TestMoneywalletParseFileNokNoHeader(t *testing.T) {
	fpath := filepath.Join("testfiles", "moneywallet", "MoneyWallet_nok_noheader.csv")
	mw := &moneywalletParser{}
	err := mw.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != HeaderError {
			t.Error("Expected HeaderError")
		}
		if pError.Line != 0 {
			t.Errorf("Expected HeaderError on line 0, got %d", pError.Line)
		}
	} else {
		t.Error("Expected ParserError")
	}

	if len(mw.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestMoneywalletParseFileNokWrongDatetime(t *testing.T) {
	fpath := filepath.Join("testfiles", "moneywallet", "MoneyWallet_nok_wrongdatetime.csv")
	mw := &moneywalletParser{}
	err := mw.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Error("Expected DataParsingError")
		}
		if pError.Line != 1 {
			t.Errorf("Expected DataParsingError on line 1, got %d", pError.Line)
		}
		if pError.Field != "datetime" {
			t.Errorf("Expected field 'datetime', got '%s' instead", pError.Field)
		}
	} else {
		t.Error("Expected ParserError")
	}

	if len(mw.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestMoneywalletParseFileNokWrongMoney(t *testing.T) {
	fpath := filepath.Join("testfiles", "moneywallet", "MoneyWallet_nok_wrongmoney.csv")
	mw := &moneywalletParser{}
	err := mw.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Error("Expected DataParsingError")
		}
		if pError.Line != 1 {
			t.Errorf("Expected DataParsingError on line 1, got %d", pError.Line)
		}
		if pError.Field != "money" {
			t.Errorf("Expected field 'money', got '%s' instead", pError.Field)
		}
	} else {
		t.Error("Expected ParserError")
	}

	if len(mw.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestMoneywalletParseFileOnlyHeader(t *testing.T) {
	fpath := filepath.Join("testfiles", "moneywallet", "MoneyWallet_onlyheader.csv")
	mw := &moneywalletParser{}
	err := mw.ParseFile(fpath)
	if err != nil {
		t.Fatal("Should not fail")
	}

	if len(mw.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestMoneywalletParseFileOk(t *testing.T) {
	fpath := filepath.Join("testfiles", "moneywallet", "MoneyWallet_export_1.csv")
	mw := &moneywalletParser{}
	if err := mw.ParseFile(fpath); err != nil {
		t.Error(err)
	}
}

func TestMoneywalletConvertToHomebank(t *testing.T) {
	fpath := filepath.Join("testfiles", "moneywallet", "MoneyWallet_export_1.csv")
	mw := &moneywalletParser{}
	err := mw.ParseFile(fpath)
	if err != nil {
		t.Error(err)
	}

	tmpDir := t.TempDir()
	tmpFilepath := filepath.Join(tmpDir, "output.csv")

	err = mw.ConvertToHomebank(tmpFilepath)
	if err != nil {
		t.Error(err)
	}

	expected := filepath.Join("testfiles", "moneywallet", "converted_1.csv")

	if !areFilesEqual(expected, tmpFilepath) {
		t.Errorf("Files are not equal %s, %s", expected, tmpFilepath)
	}
}

func TestMoneywalletConvertRecord(t *testing.T) {
	m := &moneywalletRecord{
		wallet:      "wallet",
		currency:    "EUR",
		category:    "category",
		datetime:    time.Date(2014, 2, 1, 0, 0, 0, 0, time.UTC),
		money:       10.0,
		description: "description",
	}

	h := m.convertRecord()

	if h.amount != m.money {
		t.Error("Wrong amount")
	}
	if h.category != m.category {
		t.Error("Wrong category")
	}
	if h.date != "2014-02-01" {
		t.Error("Wrong date")
	}
	if h.info != m.description {
		t.Error("Wrong info")
	}
	if h.payment != 0 {
		t.Error("Wrong payment")
	}
	if h.payee != "" {
		t.Error("Wrong payee")
	}
	if h.tags != "" {
		t.Error("Wrong tags")
	}
	if h.memo != "" {
		t.Error("Wrong memo")
	}
}

func TestIsValidMoneyWalletHeader(t *testing.T) {

	headerOk := []string{
		"wallet",
		"currency",
		"category",
		"datetime",
		"money",
		"description",
	}

	headerNok := []string{
		"wallet",
		"currency",
		"category",
		"datetime",
		"money",
		"description xxx",
	}

	headerWrongLength := []string{
		"wallet",
		"currency",
		"category",
		"datetime",
		"money",
	}

	if !isValidMoneyWalletHeader(headerOk) {
		t.Error("Header should be OK")
	}
	if isValidMoneyWalletHeader(headerNok) {
		t.Error("Header should be NOK")
	}
	if isValidMoneyWalletHeader(headerWrongLength) {
		t.Error("Header should be NOK (wrong length)")
	}
}
