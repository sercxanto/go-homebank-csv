package parser

import (
	"path/filepath"
	"testing"
)

func TestGetParser(t *testing.T) {
	for _, f := range GetSourceFormats() {
		p := GetParser(f)
		if p.GetFormat() != f {
			t.Error("Parser mismatch")
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
	}

	for testfile, format := range formats {
		p := GetGuessedParser(testfile)
		if p.GetFormat() != format {
			t.Errorf("Parser not correct, expected: %s, got: %s", format, p.GetFormat())
		}
	}
}
