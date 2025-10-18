package parser

import (
	"encoding/csv"
	"os"
	"reflect"
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
		return &ParserError{ErrorType: IOError}
	}
	defer infile.Close()
	csvReader := csv.NewReader(infile)
	records, err := csvReader.ReadAll()
	if err != nil {
		return &ParserError{ErrorType: IOError}
	}
	if len(records) == 0 {
		return &ParserError{ErrorType: HeaderError}
	}
	if !isValidMoneyWalletHeader(records[0]) {
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
		date, err := time.Parse("2006-01-02 15:04:05", row[3])
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      lineNr + 1,
				Field:     "datetime",
			}
		}

		moneyString := strings.ReplaceAll(row[4], ",", ".")
		var money float64
		money, err = strconv.ParseFloat(strings.TrimSpace(moneyString), 64)
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      lineNr + 1,
				Field:     "money",
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

func (m *moneywalletParser) GetFormat() SourceFormat {
	return MoneyWallet
}

func (m *moneywalletParser) GetNumberOfEntries() int {
	return len(m.entries)
}

func (m *moneywalletParser) ConvertToHomebank(filepath string) error {
	hRecords := make([]homebankRecord, 0, len(m.entries))
	for _, mRecord := range m.entries {
		hRecord := mRecord.convertRecord()
		hRecords = append(hRecords, hRecord)
	}

	err := writeHomeBankRecords(hRecords, filepath)
	if err != nil {
		return err
	}

	return nil
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
	return reflect.DeepEqual(record, expected)
}

// convertRecord converts a single record from barclaycard to homebank format
func (m *moneywalletRecord) convertRecord() (record homebankRecord) {
	var result homebankRecord

	result.category = m.category
	result.payment = 0
	result.info = m.description
	result.date = m.datetime.Format("2006-01-02")
	result.amount = m.money

	return result
}
