package parser

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestComdirectName(t *testing.T) {
	c := &comdirectParser{}
	if c.GetFormat() != Comdirect {
		t.Error("Wrong format")
	}
}

func TestComdirectParseFileNonExisting(t *testing.T) {
	v := &comdirectParser{}
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

func TestComdirectParseFileNok(t *testing.T) {
	fpath := filepath.Join("testfiles", "comdirect", "umsaetze_nok_noheader.csv")
	c := &comdirectParser{}
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

func TestComdirectParseFileNokInvalidHeader(t *testing.T) {
	fpath := filepath.Join("testfiles", "comdirect", "umsaetze_nok_invalidheader.csv")
	c := &comdirectParser{}
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

func TestComdirectParseFileNokWrongBuchungstag(t *testing.T) {
	fpath := filepath.Join("testfiles", "comdirect", "umsaetze_nok_wrongbuchungstag.csv")
	c := &comdirectParser{}
	err := c.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Errorf("DataParsingError expected, got '%s' instead", pError.ErrorType)
		}
		if pError.Line != 3 {
			t.Errorf("Expected error on line 3, got %d", pError.Line)
		}
		if pError.Field != "Buchungstag" {
			t.Errorf("Expected error on field 'Buchungstag', got %s", pError.Field)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(c.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestComdirectParseFileNokWrongUmsatz(t *testing.T) {
	fpath := filepath.Join("testfiles", "comdirect", "umsaetze_nok_wrongumsatz.csv")
	c := &comdirectParser{}
	err := c.ParseFile(fpath)
	if err == nil {
		t.Error("Should fail")
	}
	var pError *ParserError
	if errors.As(err, &pError) {
		if pError.ErrorType != DataParsingError {
			t.Errorf("DataParsingError expected, got '%s' instead", pError.ErrorType)
		}
		if pError.Line != 3 {
			t.Errorf("Expected error on line 3, got %d", pError.Line)
		}
		if pError.Field != "Umsatz in EUR" {
			t.Errorf("Expected error on field 'Umsatz in EUR', got %s", pError.Field)
		}
	} else {
		t.Error("ParserError expected")
	}
	if len(c.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestComdirectParseFileOnlyHeader(t *testing.T) {
	fpath := filepath.Join("testfiles", "comdirect", "umsaetze_onlyheader.csv")
	c := &comdirectParser{}
	err := c.ParseFile(fpath)
	if err != nil {
		t.Error("Should not fail")
	}
	if len(c.entries) != 0 {
		t.Error("Entries should be empty")
	}
}

func TestComdirectConvertRecord(t *testing.T) {

	c := comdirectRecord{
		buchungstag:      time.Date(2019, 8, 5, 0, 0, 0, 0, time.UTC),
		vorgang:          "Übertrag/Überweisung",
		fullBuchungstext: "Auftraggeber:auftragnameBuchungstext: Der Buchungstext 123 456Kto/IBAN: DE123 BLZ/BIC: ABC123",
		buchungstext:     "Der Buchungstext 123 456",
		auftraggeber:     "auftragname",
		empfaenger:       "",
		umsatz_eur:       -139.40,
	}
	h := c.convertRecord()
	if h.amount != c.umsatz_eur {
		t.Error("Amount does not match")
	}
	if h.date != "2019-08-05" {
		t.Errorf("Date does not match. h.date: %s, m.date: %s", h.date, c.buchungstag)
	}
	if h.info != "Der Buchungstext 123" {
		t.Error("Info does not match")
	}
	if h.payment != 0 {
		t.Error("Payment does not match")
	}
	if h.payee != "auftragname" {
		t.Errorf("Payee does not match. Got '%s'", h.payee)
	}
	if h.memo != c.fullBuchungstext {
		t.Error("Memo does not match")
	}
	if h.category != "" {
		t.Error("Category does not match")
	}
	if h.tags != "" {
		t.Error("Tags does not match")
	}

}

func TestSplitComdirectBuchungstextGeneral(t *testing.T) {
	fields := []string{"first", "second", "third"}
	buchungstext := "first:abcfirstsecond:abcsecond third:abcthird"
	expected := map[string]string{
		"first":  "abcfirst",
		"second": "abcsecond",
		"third":  "abcthird",
	}
	calculated := splitComdirectBuchungstext(fields, buchungstext)
	if !reflect.DeepEqual(expected, calculated) {
		t.Errorf("Expected != calculated (%v)", calculated)
	}

	calculated = splitComdirectBuchungstext(fields, "")
	if !reflect.DeepEqual(map[string]string{}, calculated) {
		t.Errorf("Expected != calculated (%v)", calculated)
	}

	calculated = splitComdirectBuchungstext([]string{}, "")
	if !reflect.DeepEqual(map[string]string{}, calculated) {
		t.Errorf("Expected != calculated (%v)", calculated)
	}

	fields = []string{"not_matching"}
	calculated = splitComdirectBuchungstext(fields, buchungstext)
	if !reflect.DeepEqual(map[string]string{}, calculated) {
		t.Errorf("Expected != calculated (%v)", calculated)
	}

	fields = []string{"not_matching", "second", "third"}
	expected = map[string]string{
		"second": "abcsecond",
		"third":  "abcthird",
	}
	calculated = splitComdirectBuchungstext(fields, buchungstext)
	if !reflect.DeepEqual(expected, calculated) {
		t.Errorf("Expected != calculated (%v)", calculated)
	}
}

func TestSplitComdirectBuchungstextChangedOrder(t *testing.T) {
	fields := []string{"third", "second", "first"}
	buchungstext := "first:abcfirstsecond:abcsecond third:abcthird"
	expected := map[string]string{
		"first":  "abcfirst",
		"second": "abcsecond",
		"third":  "abcthird",
	}
	calculated := splitComdirectBuchungstext(fields, buchungstext)
	if !reflect.DeepEqual(expected, calculated) {
		t.Errorf("Expected != calculated (%v)", calculated)
	}
}

func TestSplitComdirectBuchungstext(t *testing.T) {
	fields := []string{"Empfänger", "Auftraggeber", "Kto/IBAN", "Buchungstext"}
	buchungstext := "Kto/IBAN: MyKto/IBAN  Buchungstext: My Buchungstext"
	expected := map[string]string{
		"Kto/IBAN":     "MyKto/IBAN",
		"Buchungstext": "My Buchungstext",
	}
	calculated := splitComdirectBuchungstext(fields, buchungstext)
	if !reflect.DeepEqual(expected, calculated) {
		t.Errorf("Expected != calculated (%v)", calculated)
	}

	buchungstext = "Auftraggeber: MyAuftraggeber Buchungstext: MyBuchungstext"
	expected = map[string]string{
		"Auftraggeber": "MyAuftraggeber",
		"Buchungstext": "MyBuchungstext",
	}
	calculated = splitComdirectBuchungstext(fields, buchungstext)
	if !reflect.DeepEqual(expected, calculated) {
		t.Errorf("Expected != calculated (%v)", calculated)
	}
}

func TestGetFirstNWords(t *testing.T) {
	result := getFirstNWords(0, "")
	if result != "" {
		t.Error("Result should be empty")
	}
	result = getFirstNWords(3, "")
	if result != "" {
		t.Error("Result should be empty")
	}
	result = getFirstNWords(0, "Some string")
	if result != "" {
		t.Error("Result should be empty")
	}
	result = getFirstNWords(2, "Some string")
	if result != "Some string" {
		t.Error("Result should be 'Some string'")
	}
	result = getFirstNWords(3, "Some string")
	if result != "Some string" {
		t.Error("Result should be 'Some string'")
	}
	result = getFirstNWords(3, "Somestring")
	if result != "Somestring" {
		t.Error("Result should be 'Some string'")
	}
}

func TestComdirectParseFileOk(t *testing.T) {
	fpath := filepath.Join("testfiles", "comdirect", "umsaetze_1234567890_20231006_1804.csv")
	c := &comdirectParser{}
	if err := c.ParseFile(fpath); err != nil {
		t.Error(err)
	}
}

func TestComdirectConvertToHomebank(t *testing.T) {
	fpath := filepath.Join("testfiles", "comdirect", "umsaetze_1234567890_20231006_1804.csv")
	c := &comdirectParser{}
	err := c.ParseFile(fpath)
	if err != nil {
		t.Error(err)
	}

	tmpDir := t.TempDir()
	tmpFilepath := filepath.Join(tmpDir, "output.csv")

	err = c.ConvertToHomebank(tmpFilepath)
	if err != nil {
		t.Error(err)
	}

	expected := filepath.Join("testfiles", "comdirect", "homebank.csv")

	if !areFilesEqual(expected, tmpFilepath) {
		t.Errorf("Files are not equal %s, %s", expected, tmpFilepath)
	}
}

func TestIsValidComdirectHeader(t *testing.T) {

	headerOk := []string{
		"Buchungstag",
		"Wertstellung (Valuta)",
		"Vorgang",
		"Buchungstext",
		"Umsatz in EUR",
		"",
	}

	headerNok := []string{
		"Bezeichnung Auftragskonto",
		"IBAN Auftragskonto",
		"BIC Auftragskonto XXXXXXXXXXXXXXXXXXXXX",
		"Bankname Auftragskonto",
		"Buchungstag",
		"Valutadatum",
		"",
	}

	headerWrongLength := []string{
		"Buchungstag",
		"Wertstellung (Valuta)",
		"Vorgang",
		"Buchungstext",
		"Umsatz in EUR",
		"",
		"Additional field",
	}

	if !isValidComdirectHeader(headerOk) {
		t.Error("Header should be OK")
	}
	if isValidComdirectHeader(headerNok) {
		t.Error("Header should be NOK")
	}
	if isValidComdirectHeader(headerWrongLength) {
		t.Error("Header should be NOK (wrong length)")
	}
}
