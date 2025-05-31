package parser

import (
	"encoding/csv"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Single record of volksbank data, all data is stored as quoted string in the CSV file
type volksbankRecord struct {
	buchungstag             time.Time
	verwendungszweck        string
	nameZahlungsbeteiligter string
	betrag                  float64
}

type volksbankParser struct {
	entries []volksbankRecord
}

func (m *volksbankParser) ParseFile(filepath string) error {
	m.entries = make([]volksbankRecord, 0)
	infile, err := os.Open(filepath)
	if err != nil {
		return &ParserError{ErrorType: IOError}
	}
	defer infile.Close()
	csvReader := csv.NewReader(infile)
	csvReader.Comma = ';'
	records, err := csvReader.ReadAll()
	if err != nil {
		return &ParserError{ErrorType: IOError}
	}
	if len(records) == 0 {
		return &ParserError{ErrorType: HeaderError}
	}

	if !isValidVolksbankHeader(records[0]) {
		return &ParserError{
			ErrorType: HeaderError,
			Line:      1,
		}
	}

	// Only header found, no entries
	if len(records) == 1 {
		return nil
	}

	for lineNr, row := range records[1:] {
		date, err := time.Parse("02.01.2006", row[4])
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      lineNr + 2,
				Field:     "Buchungstag",
			}
		}
		betragString := strings.Replace(row[11], ",", ".", -1)
		var betrag float64
		betrag, err = strconv.ParseFloat(betragString, 64)
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      lineNr + 2,
				Field:     "Betrag",
			}
		}
		vRecord := volksbankRecord{
			buchungstag:             date,
			verwendungszweck:        row[10],
			nameZahlungsbeteiligter: row[6],
			betrag:                  betrag,
		}
		m.entries = append(m.entries, vRecord)
	}

	return nil
}

func (m *volksbankParser) GetFormat() SourceFormat {
	return Volksbank
}

func (m *volksbankParser) GetNumberOfEntries() int {
	return len(m.entries)
}

func (v *volksbankParser) ConvertToHomebank(filepath string) error {
	hRecords := make([]homebankRecord, 0, len(v.entries))
	for _, mRecord := range v.entries {
		hRecord := mRecord.convertRecord()
		hRecords = append(hRecords, hRecord)
	}

	err := writeHomeBankRecords(hRecords, filepath)
	if err != nil {
		return err
	}

	return nil
}

func isValidVolksbankHeader(record []string) bool {
	expected := []string{
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
		"Gekennzeichneter Umsatz",
		"Glaeubiger ID",
		"Mandatsreferenz",
	}
	return reflect.DeepEqual(record, expected)
}

// convertRecord converts a single record from volksbank to homebank format
func (v *volksbankRecord) convertRecord() (record homebankRecord) {
	var result homebankRecord
	result.payment = 0
	result.memo = v.verwendungszweck
	result.date = v.buchungstag.Format("2006-01-02")
	result.amount = v.betrag
	result.payee = v.nameZahlungsbeteiligter

	return result
}
