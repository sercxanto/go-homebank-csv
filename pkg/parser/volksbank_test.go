package parser

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestVolksbankName(t *testing.T) {
	v := &volksbankParser{}
	if v.GetFormat() != Volksbank {
		t.Error("Wrong format")
	}
}

func TestVolksbankParseFileNonExisting(t *testing.T) {
	v := &volksbankParser{}
	err := v.ParseFile("non_existing_file.csv")
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
	if v.GetNumberOfEntries() != 0 {
		t.Error("Entries should be empty")
	}
}

func TestVolksbankParseFileNok(t *testing.T) {
	fpath := filepath.Join("testfiles", "moneywallet", "MoneyWallet_export_1.csv")
	v := &volksbankParser{}
	err := v.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != HeaderError && pError.ErrorType != IOError {
			t.Errorf("HeaderError or IOError expected, got '%s' instead", pError.ErrorType)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(v.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestVolksbankConvertRecord(t *testing.T) {
	v := volksbankRecord{
		buchungstag:             time.Date(2014, 2, 1, 0, 0, 0, 0, time.UTC),
		verwendungszweck:        "My Verwendungs-Zweck",
		nameZahlungsbeteiligter: "Name Zahlungsbeteiligter",
		betrag:                  200.123,
	}
	h := v.convertRecord()
	if h.amount != v.betrag {
		t.Error("Amount does not match")
	}
	if h.date != "2014-02-01" {
		t.Errorf("Date does not match. h.date: %s, m.date: %s", h.date, v.buchungstag)
	}
	if h.info != "" {
		t.Error("Info does not match")
	}
	if h.payment != 0 {
		t.Error("Payment does not match")
	}
	if h.payee != v.nameZahlungsbeteiligter {
		t.Error("Payee does not match")
	}
	if h.memo != v.verwendungszweck {
		t.Error("Memo does not match")
	}
	if h.category != "" {
		t.Error("Category does not match")
	}
	if h.tags != "" {
		t.Error("Tags does not match")
	}

}

func TestVolksbankParseFileOk(t *testing.T) {
	fpath := filepath.Join("testfiles", "volksbank", "Umsaetze_DE12345678901234567890_2023.10.04.csv")
	v := &volksbankParser{}
	if err := v.ParseFile(fpath); err != nil {
		t.Error(err)
	}
}

func TestVolksbankConvertToHomebank(t *testing.T) {
	fpath := filepath.Join("testfiles", "volksbank", "Umsaetze_DE12345678901234567890_2023.10.04.csv")
	v := &volksbankParser{}
	err := v.ParseFile(fpath)
	if err != nil {
		t.Error(err)
	}

	tmpDir := t.TempDir()
	tmpFilepath := filepath.Join(tmpDir, "output.csv")

	err = v.ConvertToHomebank(tmpFilepath)
	if err != nil {
		t.Error(err)
	}

	expected := filepath.Join("testfiles", "volksbank", "homebank.csv")

	if !areFilesEqual(expected, tmpFilepath) {
		t.Errorf("Files are not equal %s, %s", expected, tmpFilepath)
	}
}

func TestIsValidVolksbankHeader(t *testing.T) {

	headerOk := []string{
		"Bezeichnung Auftragskonto",
		"IBAN Auftragskonto",
		"BIC Auftragskonto",
		"Bankname Auftragskonto",
		"Buchungstag",
		"Valutadatum",
		"Name Zahlungsbeteiligter",
		"IBAN Zahlungsbeteiligter",
		"BIC (SWIFT-Code) Zahlungsbeteiligter",
		"Buchungstext",
		"Verwendungszweck",
		"Betrag",
		"Waehrung",
		"Saldo nach Buchung",
		"Bemerkung",
		"Kategorie",
		"Steuerrelevant",
		"Glaeubiger ID",
		"Mandatsreferenz",
	}

	headerNok := []string{
		"Bezeichnung Auftragskonto xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		"IBAN Auftragskonto",
		"BIC Auftragskonto",
		"Bankname Auftragskonto",
		"Buchungstag",
		"Valutadatum",
		"Name Zahlungsbeteiligter",
		"IBAN Zahlungsbeteiligter",
		"BIC (SWIFT-Code) Zahlungsbeteiligter",
		"Buchungstext",
		"Verwendungszweck",
		"Betrag",
		"Waehrung",
		"Saldo nach Buchung",
		"Bemerkung",
		"Kategorie",
		"Steuerrelevant",
		"Glaeubiger ID",
		"Mandatsreferenz",
	}

	headerWrongLength := []string{
		"Bezeichnung Auftragskonto",
		"IBAN Auftragskonto",
		"BIC Auftragskonto",
		"Bankname Auftragskonto",
		"Buchungstag",
		"Valutadatum",
	}

	if !isValidVolksbankHeader(headerOk) {
		t.Error("Header should be OK")
	}
	if isValidVolksbankHeader(headerNok) {
		t.Error("Header should be NOK")
	}
	if isValidVolksbankHeader(headerWrongLength) {
		t.Error("Header should be NOK (wrong length)")
	}
}
