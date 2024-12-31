package parser

/*

Parsing rules:

- The first lines of DKBs CSV can be skipped until the header line with the field names is found
- Homebanks "date" field" is equivalent to DKBs "Buchungsdatum"
- DKBs "Umsatztyp" depicts incoming ("Eingang") or outgoing ("Ausgang") transactions
- There are two DKB fields with the name "Zahlungspflichtige*r". Only in case of outgoing transactions this field is equivalent to Homebanks "payee". In other cases parsing of this field is skipped.
- For incoming transactions ("Umsatztyp"=Eingang) payee is left empty
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
	buchungsdatum        time.Time
	wertstellung         time.Time
	status               string
	zahlungspflichtiger1 string
	zahlungspflichtiger2 string
	verwendungszweck     string
	umsatztyp            string
	iban                 string
	betrag_eur           float64
	glaeubigerId         string
	mandatsreferenz      string
	kundenreferenz       string
}

type dkbParser struct {
	entries []dkbRecord
}

func (p *dkbParser) ParseFile(filepath string) error {
	const headerInRecordNr int = 3 // csvReader skips completely empty lines, so the header is in the third line
	const lineNrOffset int = 6     // line number offset for error messages
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
	if len(records) < headerInRecordNr+1 {
		return &ParserError{ErrorType: HeaderError}
	}

	if !isValidDkbHeader(records[headerInRecordNr]) {
		return &ParserError{
			ErrorType: HeaderError,
			Line:      headerInRecordNr + 2,
		}
	}

	for lineNr, row := range records[headerInRecordNr+1:] {
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
				Line:      lineNrOffset + lineNr,
				Field:     "Buchungsdatum",
			}
		}
		parsedWertstellung, err := time.Parse("02.01.06", row[1])
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      lineNrOffset + lineNr,
				Field:     "Wertstellung",
			}
		}
		amountString := strings.Replace(row[8], ".", "", -1)
		amountString = strings.Replace(amountString, ",", ".", -1)
		var amount float64
		amount, err = strconv.ParseFloat(amountString, 64)
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      lineNrOffset + lineNr,
				Field:     "Betrag (€)",
			}
		}
		dRecord := dkbRecord{
			buchungsdatum:        parsedBuchungsdatum,
			wertstellung:         parsedWertstellung,
			status:               row[2],
			zahlungspflichtiger1: row[3],
			zahlungspflichtiger2: row[4],
			verwendungszweck:     row[5],
			umsatztyp:            row[6],
			iban:                 row[7],
			betrag_eur:           amount,
			glaeubigerId:         row[9],
			mandatsreferenz:      row[10],
			kundenreferenz:       row[11],
		}
		if dRecord.umsatztyp == "Eingang" && dRecord.betrag_eur == 0 && dRecord.zahlungspflichtiger1 == "DKB AG" && dRecord.zahlungspflichtiger2 == "DKB AG" {
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
		h.payee = d.zahlungspflichtiger2
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
		"Zahlungspflichtige*r",
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
