package parser

import (
	"encoding/csv"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Single record of moneywallet data, all data is stored as quoted string in the CSV file
type moneywalletRecord struct {
	wallet      string
	currency    string
	category    string
	datetime    time.Time
	money       float64
	description string
}

type moneywalletParser struct {
	entries []moneywalletRecord
}

func (m *moneywalletParser) ParseFile(filepath string) error {
	m.entries = make([]moneywalletRecord, 0)
	infile, err := os.Open(filepath)
	if err != nil {
		return &ParseError{Type: IOError, Err: err}
	}
	defer infile.Close()
	csvReader := csv.NewReader(infile)
	records, err := readCSVRecords(csvReader)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return &ParseError{Type: HeaderError}
	}
	if !isValidMoneyWalletHeader(records[0].fields) {
		return &ParseError{
			Type: HeaderError,
			Line: records[0].line(0),
		}
	}
	// Only header found, no entries
	if len(records) == 1 {
		return nil
	}

	for _, record := range records[1:] {
		row := record.fields
		date, err := time.Parse("2006-01-02 15:04:05", row[3])
		if err != nil {
			return &ParseError{
				Type:  DataParsingError,
				Line:  record.line(3),
				Field: "datetime",
				Err:   err,
			}
		}

		moneyString := strings.ReplaceAll(row[4], ",", ".")
		var money float64
		money, err = strconv.ParseFloat(strings.TrimSpace(moneyString), 64)
		if err != nil {
			return &ParseError{
				Type:  DataParsingError,
				Line:  record.line(4),
				Field: "money",
				Err:   err,
			}
		}

		mwRecord := moneywalletRecord{
			wallet:      row[0],
			currency:    row[1],
			category:    row[2],
			datetime:    date,
			money:       money,
			description: row[5],
		}
		m.entries = append(m.entries, mwRecord)
	}

	return nil
}

func (m *moneywalletParser) SourceFormat() SourceFormat {
	return MoneyWallet
}

func (m *moneywalletParser) Len() int {
	return len(m.entries)
}

func (m *moneywalletParser) ConvertToHomebank(filepath string) error {
	return writeConvertedRecords(m.entries, (*moneywalletRecord).convertRecord, filepath)
}

func isValidMoneyWalletHeader(record []string) bool {
	expected := []string{
		"wallet",
		"currency",
		"category",
		"datetime",
		"money",
		"description",
	}
	return slices.Equal(record, expected)
}

// convertRecord converts a single record from moneywallet to homebank format
func (m *moneywalletRecord) convertRecord() (record homebankRecord) {
	var result homebankRecord

	result.category = m.category
	result.payment = paymentNone
	result.info = m.description
	result.date = m.datetime.Format("2006-01-02")
	result.amount = m.money

	return result
}
