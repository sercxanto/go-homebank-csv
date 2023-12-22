package parser

import (
	"path/filepath"
	"testing"
)

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
	}

	for testfile, format := range formats {
		p := GetGuessedParser(testfile)
		if p.GetFormat() != format {
			t.Errorf("Parser not correct, expected: %s, got: %s", format, p.GetFormat())
		}
	}
}
