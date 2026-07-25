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
		"Karteninhaber",
		"Kartennetzwerk",
		"Kontaktlose Bezahlung",
		"Details",
	}
	return reflect.DeepEqual(record, expected)
}

// barclaycardCell returns the value of the given column of a row.
//
// It returns an empty string if the row does not contain the column. excelize
// omits trailing empty cells, so a row can be shorter than the header, e.g.
// when the last column "Details" is not filled.
func barclaycardCell(row []string, column int) string {
	if column >= len(row) {
		return ""
	}
	return row[column]
}

// isEmptyBarclaycardRow reports whether a row contains no data at all.
func isEmptyBarclaycardRow(row []string) bool {
	for _, column := range row {
		if len(strings.TrimSpace(column)) > 0 {
			return false
		}
	}
	return true
}

func parseBarclaycardAmount(valueString string) (float64, error) {
	valueString = strings.TrimSpace(valueString)
	valueString = strings.TrimSuffix(valueString, "€")
	valueString = strings.TrimSpace(valueString)
	valueString = strings.ReplaceAll(valueString, ".", "")
	valueString = strings.ReplaceAll(valueString, ",", ".")
	return strconv.ParseFloat(valueString, 64)
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

			// Empty rows, e.g. separating the data section from a footer,
			// carry no transaction and are skipped
			if isEmptyBarclaycardRow(row) {
				continue
			}

			tDate, err := time.Parse("02.01.2006", barclaycardCell(row, 1))
			if err != nil {
				return &ParserError{
					ErrorType: DataParsingError,
					Line:      lineNr + 1,
					Field:     "Buchungsdatum(1)/Transaktionsdatum",
				}
			}

			// Entries with an empty "Buchungsdatum" are "vorgemerkt", not "Berechnet"
			// and need to be skipped
			if len(barclaycardCell(row, 2)) == 0 {
				continue
			}

			bDate, err := time.Parse("02.01.2006", barclaycardCell(row, 2))
			if err != nil {
				return &ParserError{
					ErrorType: DataParsingError,
					Line:      lineNr + 1,
					Field:     "Buchungsdatum",
				}
			}

			value, err := parseBarclaycardAmount(barclaycardCell(row, 3))
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
				description:     barclaycardCell(row, 14),
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
