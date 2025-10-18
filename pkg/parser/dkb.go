package parser

/*

Parsing rules:

- The first lines of DKBs CSV can be skipped until the header line with the field names is found
- Homebanks "date" field" is equivalent to DKBs "Buchungsdatum"
- DKBs "Umsatztyp" depicts incoming ("Eingang") or outgoing ("Ausgang") transactions
- There is a special record for "Abrechnung". It is skipped and not transferred to Homebank. It can be identified by the following values:
  "Umsatztyp"=Eingang, "Betrag"=0, both Fields "Zahlungspflichtige*r Name" are set to "DKB AG"
*/

import (
	"encoding/csv"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type dkbRecord struct {
	buchungsdatum       time.Time
	wertstellung        time.Time
	status              string
	zahlungspflichtiger string
	zahlungsempfaenger  string
	verwendungszweck    string
	umsatztyp           string
	iban                string
	betrag_eur          float64
	glaeubigerId        string
	mandatsreferenz     string
	kundenreferenz      string
}

type dkbParser struct {
	entries []dkbRecord
}

func (p *dkbParser) ParseFile(filepath string) error {
	p.entries = make([]dkbRecord, 0)
	infile, err := os.Open(filepath)
	if err != nil {
		return &ParserError{ErrorType: IOError}
	}
	defer infile.Close()

	csvReader := csv.NewReader(infile)
	csvReader.Comma = ';'
	csvReader.FieldsPerRecord = -1 // Enable variable length records
	// Workaround for UTF-8 Byte Order Mark (BOM) not supported by csv reader
	// see https://github.com/golang/go/issues/33887
	csvReader.LazyQuotes = true
	records, err := csvReader.ReadAll()
	if err != nil {
		return &ParserError{ErrorType: IOError}
	}

	var headerIndex = -1
	for i, record := range records {
		if isValidDkbHeader(record) {
			headerIndex = i
			break
		}
	}

	if headerIndex == -1 {
		return &ParserError{ErrorType: HeaderError}
	}

	for lineNr, row := range records[headerIndex+1:] {
		nonEmptyLineNr := headerIndex + lineNr + 2
		if len(row) != 12 {
			continue
		}
		if row[2] != "Gebucht" {
			continue
		}
		parsedBuchungsdatum, err := time.Parse("02.01.06", row[0])
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      nonEmptyLineNr,
				Field:     "Buchungsdatum",
			}
		}
		parsedWertstellung, err := time.Parse("02.01.06", row[1])
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      nonEmptyLineNr,
				Field:     "Wertstellung",
			}
		}
		amountString := strings.ReplaceAll(row[8], ".", "")
		amountString = strings.ReplaceAll(amountString, ",", ".")
		var amount float64
		amount, err = strconv.ParseFloat(amountString, 64)
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      nonEmptyLineNr,
				Field:     "Betrag (€)",
			}
		}
		dRecord := dkbRecord{
			buchungsdatum:       parsedBuchungsdatum,
			wertstellung:        parsedWertstellung,
			status:              row[2],
			zahlungspflichtiger: row[3],
			zahlungsempfaenger:  row[4],
			verwendungszweck:    row[5],
			umsatztyp:           row[6],
			iban:                row[7],
			betrag_eur:          amount,
			glaeubigerId:        row[9],
			mandatsreferenz:     row[10],
			kundenreferenz:      row[11],
		}
		if dRecord.umsatztyp == "Eingang" && dRecord.betrag_eur == 0 && dRecord.zahlungspflichtiger == "DKB AG" && dRecord.zahlungsempfaenger == "DKB AG" {
			continue
		}
		p.entries = append(p.entries, dRecord)
	}
	return nil
}

func (d *dkbParser) GetFormat() SourceFormat {
	return DKB
}

func (d *dkbParser) GetNumberOfEntries() int {
	return len(d.entries)
}

func (v *dkbParser) ConvertToHomebank(filepath string) error {
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

func (d *dkbRecord) convertRecord() (h homebankRecord) {
	h.payment = 0
	h.date = d.buchungsdatum.Format("2006-01-02")
	if d.betrag_eur < 0 {
		h.payee = d.zahlungsempfaenger
	}
	h.memo = d.verwendungszweck
	h.amount = d.betrag_eur
	return
}

func isValidDkbHeader(record []string) bool {
	expected := []string{
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
	return reflect.DeepEqual(record, expected)
}
