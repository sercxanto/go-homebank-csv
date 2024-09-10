package parser

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Single record of relevant barclaycard data, all data is stored as string in the the excel file
type barclaycardRecord struct {
	transactionDate time.Time
	bookingDate     time.Time
	value           float64
	description     string
	payee           string
}

type barclaycardParser struct {
	entries []barclaycardRecord
}

func (b *barclaycardParser) GetFormat() SourceFormat {
	return Barclaycard
}

func (b *barclaycardParser) GetNumberOfEntries() int {
	return len(b.entries)
}

func isValidBarclaycardHeader(record []string) bool {
	expected := []string{
		"Referenznummer",
		"Buchungsdatum", // eigentlich: Transaktionsdatum
		"Buchungsdatum",
		"Betrag",
		"Beschreibung",
		"Typ",
		"Status",
		"Kartennummer",
		"Originalbetrag",
		"Mögliche Zahlpläne",
		"Land",
		"Name des Karteninhabers",
		"Kartennetzwerk",
		"Kontaktlose Bezahlung",
		"Händlerdetails",
	}
	return reflect.DeepEqual(record, expected)
}

func (b *barclaycardParser) ParseFile(filepath string) error {
	b.entries = make([]barclaycardRecord, 0)
	f, err := excelize.OpenFile(filepath)
	if err != nil {
		return &ParserError{ErrorType: IOError}
	}
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return &ParserError{
			ErrorType: HeaderError,
		}
	}

	inDataSection := false
	dataSectionFound := false

	for lineNr, row := range rows {
		if inDataSection {

			tDate, err := time.Parse("02.01.2006", row[1])
			if err != nil {
				return &ParserError{
					ErrorType: DataParsingError,
					Line:      lineNr + 1,
					Field:     "Buchungsdatum(1)/Transaktionsdatum",
				}
			}

			// Entries with an empty "Buchungsdatum" are "vorgemerkt", not "Berechnet"
			// and need to be skipped
			if len(row[2]) == 0 {
				continue
			}

			bDate, err := time.Parse("02.01.2006", row[2])
			if err != nil {
				return &ParserError{
					ErrorType: DataParsingError,
					Line:      lineNr + 1,
					Field:     "Buchungsdatum",
				}
			}

			var value float64
			// Format in excel export is "3,14 €"
			valueString := strings.Replace(row[3], ",", ".", -1)
			valueString = strings.TrimRight(valueString, "€")
			value, err = strconv.ParseFloat(strings.TrimSpace(valueString), 64)
			if err != nil {
				return &ParserError{
					ErrorType: DataParsingError,
					Line:      lineNr + 1,
					Field:     "Betrag",
				}
			}

			bRecord := barclaycardRecord{
				transactionDate: tDate,
				bookingDate:     bDate,
				value:           value,
				description:     row[4],
				payee:           row[14],
			}
			b.entries = append(b.entries, bRecord)
		} else {
			if isValidBarclaycardHeader(row) {
				inDataSection = true
				dataSectionFound = true
			}
		}
	}
	if !dataSectionFound {
		return &ParserError{
			ErrorType: HeaderError,
		}
	}
	return nil
}

func (b *barclaycardRecord) convertRecord() homebankRecord {
	return homebankRecord{
		date:     b.transactionDate.Format("2006-01-02"),
		payment:  1, // Credit card
		info:     b.description,
		payee:    b.payee,
		memo:     "",
		amount:   b.value,
		category: "",
		tags:     "",
	}
}

func (b *barclaycardParser) ConvertToHomebank(filepath string) error {
	hRecords := make([]homebankRecord, 0, len(b.entries))
	for _, bRecord := range b.entries {
		hRecord := bRecord.convertRecord()
		hRecords = append(hRecords, hRecord)
	}
	err := writeHomeBankRecords(hRecords, filepath)
	if err != nil {
		return err
	}
	return nil
}
