package parser

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestDkbName(t *testing.T) {
	c := &dkbParser{}
	if c.GetFormat() != DKB {
		t.Error("Wrong format")
	}
}

func TestDkbParseFileNonExisting(t *testing.T) {
	v := &dkbParser{}
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

func TestDkbParseFileNok(t *testing.T) {
	fpath := filepath.Join("testfiles", "dkb", "dkb_nok_noheader.csv")
	c := &dkbParser{}
	err := c.ParseFile(fpath)
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
	if len(c.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestDkbParseFileNokInvalidHeader(t *testing.T) {
	fpath := filepath.Join("testfiles", "dkb", "dkb_nok_invalidheader.csv")
	c := &dkbParser{}
	err := c.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != HeaderError {
			t.Errorf("HeaderError expected, got '%s' instead", pError.ErrorType)
		}
		if pError.Line != 0 {
			t.Errorf("Expected error on line 0, got %d", pError.Line)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(c.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestDkbParseFileNokWrongBuchungsdatum(t *testing.T) {
	fpath := filepath.Join("testfiles", "dkb", "dkb_nok_wrongbuchungsdatum.csv")
	c := &dkbParser{}
	err := c.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Errorf("DataParsingError expected, got '%s' instead", pError.ErrorType)
		}
		if pError.Line != 5 {
			t.Errorf("Expected error on line 5, got %d", pError.Line)
		}
		if pError.Field != "Buchungsdatum" {
			t.Errorf("Expected error on field 'Buchungsdatum', got '%s'", pError.Field)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(c.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestDkbParseFileNokWrongWertstellung(t *testing.T) {
	fpath := filepath.Join("testfiles", "dkb", "dkb_nok_wrongwertstellung.csv")
	c := &dkbParser{}
	err := c.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Errorf("DataParsingError expected, got '%s' instead", pError.ErrorType)
		}
		if pError.Line != 5 {
			t.Errorf("Expected error on line 5, got %d", pError.Line)
		}
		if pError.Field != "Wertstellung" {
			t.Errorf("Expected error on field 'Wertstellung', got '%s'", pError.Field)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(c.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestDkbParseFileNokWrongBetrag(t *testing.T) {
	fpath := filepath.Join("testfiles", "dkb", "dkb_nok_wrongbetrag.csv")
	c := &dkbParser{}
	err := c.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Errorf("DataParsingError expected, got '%s' instead", pError.ErrorType)
		}
		if pError.Line != 5 {
			t.Errorf("Expected error on line 5, got %d", pError.Line)
		}
		if pError.Field != "Betrag (€)" {
			t.Errorf("Expected error on field 'Betrag (€)', got '%s'", pError.Field)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(c.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestDkbParseFileOnlyHeader(t *testing.T) {
	fpath := filepath.Join("testfiles", "dkb", "dkb_onlyheader.csv")
	c := &dkbParser{}
	err := c.ParseFile(fpath)
	if err != nil {
		t.Errorf("Should not fail: %v", err)
	}
	if len(c.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestDkbConvertRecord(t *testing.T) {
	d := dkbRecord{
		buchungsdatum:       time.Date(2024, 12, 13, 0, 0, 0, 0, time.UTC),
		wertstellung:        time.Date(2024, 12, 14, 0, 0, 0, 0, time.UTC),
		status:              "Gebucht",
		zahlungspflichtiger: "Name1",
		zahlungsempfaenger:  "Name2",
		verwendungszweck:    "Verwendungszweck",
		umsatztyp:           "Ausgang",
		iban:                "DE12345678901234567890",
		betrag_eur:          -1000.0,
		glaeubigerId:        "DE98ZZZ09999999999",
		mandatsreferenz:     "Mandatsreferenz",
		kundenreferenz:      "Kundenreferenz",
	}
	h := d.convertRecord()
	if h.amount != d.betrag_eur {
		t.Errorf("Expected amount to be %f, got %f", d.betrag_eur, h.amount)
	}
	if h.date != "2024-12-13" {
		t.Errorf("Expected date to be 2024-12-13, got '%s'", h.date)
	}
	if h.payment != 0 {
		t.Errorf("Expected payment to be 0, got %d", h.payment)
	}
	if h.payee != d.zahlungsempfaenger {
		t.Errorf("Expected payee to be '%s', got '%s'", d.zahlungsempfaenger, h.payee)
	}
	if h.memo != d.verwendungszweck {
		t.Errorf("Expected memo to be '%s', got '%s'", d.verwendungszweck, h.memo)
	}
	if h.info != "" {
		t.Errorf("Expected info to be empty, got '%s'", h.info)
	}
	if h.category != "" {
		t.Errorf("Expected category to be empty, got '%s'", h.category)
	}
	if h.tags != "" {
		t.Errorf("Expected tags to be empty, got '%s'", h.tags)
	}
}

func TestDkbParseFileOk(t *testing.T) {
	fpath := filepath.Join("testfiles", "dkb", "dkb.csv")
	d := &dkbParser{}
	if err := d.ParseFile(fpath); err != nil {
		t.Error(err)
	}
}

func TestDkbConvertToHomebank(t *testing.T) {
	fpath := filepath.Join("testfiles", "dkb", "dkb.csv")
	d := &dkbParser{}
	err := d.ParseFile(fpath)
	if err != nil {
		t.Error(err)
	}

	tmpDir := t.TempDir()
	tmpFilepath := filepath.Join(tmpDir, "output.csv")

	err = d.ConvertToHomebank(tmpFilepath)
	if err != nil {
		t.Error(err)
	}

	expected := filepath.Join("testfiles", "dkb", "homebank.csv")

	if !areFilesEqual(expected, tmpFilepath) {
		t.Errorf("Files are not equal %s, %s", expected, tmpFilepath)
	}
}

func TestIsValidDkbHeader(t *testing.T) {
	validHeader := []string{
		"Buchungsdatum",
		"Wertstellung",
		"Status",
		"Zahlungspflichtige*r",
		"Zahlungsempfänger*in",
		"Verwendungszweck",
		"Umsatztyp",
		"IBAN",
		"Betrag (€)",
		"Gläubiger-ID",
		"Mandatsreferenz",
		"Kundenreferenz",
	}

	invalidHeader := []string{
		"Datum",
		"Wertstellung",
		"Status",
		"Zahlungspflichtige*r",
		"Zahlungsempfänger*in",
		"Verwendungszweck",
		"Umsatztyp",
		"IBAN",
		"Betrag (€)",
		"Gläubiger-ID",
		"Mandatsreferenz",
		"Kundenreferenz",
	}

	if !isValidDkbHeader(validHeader) {
		t.Errorf("Expected valid header to be valid")
	}

	if isValidDkbHeader(invalidHeader) {
		t.Errorf("Expected invalid header to be invalid")
	}
}
