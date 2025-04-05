package parser

import (
	"encoding/csv"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Single record of comdirect data, all data is stored as quoted string in the CSV file
type comdirectRecord struct {
	buchungstag      time.Time
	vorgang          string
	fullBuchungstext string // Contains all fields
	auftraggeber     string // parsed from fullBuchungstext
	buchungstext     string // parsed from fullBuchungstext
	empfaenger       string // parsed from fullBuchungstext
	ktoIBAN          string // parsed from fullBuchungstext
	blzBic           string // parsed from fullBuchungstext
	umsatz_eur       float64
}

type comdirectParser struct {
	entries []comdirectRecord
}

func (m *comdirectParser) ParseFile(filepath string) error {
	m.entries = make([]comdirectRecord, 0)
	infile, err := os.Open(filepath)
	if err != nil {
		return &ParserError{ErrorType: IOError}
	}
	defer infile.Close()

	reader := transform.NewReader(infile, charmap.ISO8859_1.NewDecoder())
	csvReader := csv.NewReader(reader)
	csvReader.Comma = ';'
	csvReader.FieldsPerRecord = -1 // Enable variable length records
	records, err := csvReader.ReadAll()
	if err != nil {
		return &ParserError{ErrorType: IOError}
	}

	var headerIndex int = -1
	for i, record := range records {
		if isValidComdirectHeader(record) {
			headerIndex = i
			break
		}
	}

	if headerIndex == -1 {
		return &ParserError{ErrorType: HeaderError}
	}

	for lineNr, row := range records[headerIndex+1:] {
		if len(row) != 6 {
			continue
		}
		if row[0] == "offen" {
			continue
		}
		date, err := time.Parse("02.01.2006", row[0])
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      lineNr + headerIndex + 2,
				Field:     "Buchungstag",
			}
		}
		umsatzString := strings.Replace(row[4], ".", "", -1)
		umsatzString = strings.Replace(umsatzString, ",", ".", -1)
		var umsatz float64
		umsatz, err = strconv.ParseFloat(umsatzString, 64)
		if err != nil {
			return &ParserError{
				ErrorType: DataParsingError,
				Line:      lineNr + headerIndex + 2,
				Field:     "Umsatz in EUR",
			}
		}

		cRecord := comdirectRecord{
			buchungstag:      date,
			vorgang:          row[2],
			fullBuchungstext: row[3],
			umsatz_eur:       umsatz,
		}

		listOfFields := []string{"Auftraggeber", "Buchungstext", "Empfänger", "Kto/IBAN", "BLZ/BIC"}
		splitInfo := splitComdirectBuchungstext(listOfFields, row[3])

		if val, ok := splitInfo["Auftraggeber"]; ok {
			cRecord.auftraggeber = val
		}
		if val, ok := splitInfo["Buchungstext"]; ok {
			cRecord.buchungstext = val
		}
		if val, ok := splitInfo["Empfänger"]; ok {
			cRecord.empfaenger = val
		}
		if val, ok := splitInfo["Kto/IBAN"]; ok {
			cRecord.ktoIBAN = val
		}
		if val, ok := splitInfo["BLZ/BIC"]; ok {
			cRecord.blzBic = val
		}

		m.entries = append(m.entries, cRecord)
	}

	return nil
}

func (m *comdirectParser) GetFormat() SourceFormat {
	return Comdirect
}

func (m *comdirectParser) GetNumberOfEntries() int {
	return len(m.entries)
}

func (v *comdirectParser) ConvertToHomebank(filepath string) error {
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

/*
Split buchungstext according to fields

fields: ["first", "second", "third"]
buchungstext: "first:abcfirstsecond:abcsecond third:abcthird"

Result:

	{
		"first": "abcfirst",
		"second": "abcsecond",
		"third": "abcthird"
	}
*/
func splitComdirectBuchungstext(fields []string, buchungstext string) map[string]string {
	result := make(map[string]string)

	/*
		Idea: get a sorted map of start positions:

		* key: start of where field has been found
		* value: fieldname

		e.g.

		{
			0: "first",
			14: "second",
			31: "third"
		}
	*/

	startPositions := make(map[int]string, len(fields))
	for _, s := range fields {
		pos := strings.Index(buchungstext, s+":")
		if pos == -1 {
			continue
		}
		startPositions[pos] = s
	}

	if len(startPositions) == 0 {
		return result
	}

	/* Get a sorted list of startPosition sortedStartPositions,
	   e.g. [0,14,31]
	*/
	sortedStartPositions := make([]int, 0, len(startPositions))
	for k := range startPositions {
		sortedStartPositions = append(sortedStartPositions, k)
	}
	sort.Ints(sortedStartPositions)

	/*
	   Iterate over the sorted positions and extract the fieldname and value
	   the value is either until the next fieldname or the end of the buchungstext
	*/
	for i, startIndex := range sortedStartPositions {
		fieldName := startPositions[startIndex]
		endIndex := len(buchungstext)
		if i < len(sortedStartPositions)-1 {
			endIndex = sortedStartPositions[i+1]
		}
		value := buchungstext[startIndex+len(fieldName)+1 : endIndex]
		value = strings.TrimSpace(value)
		result[fieldName] = value
	}

	return result
}

func isValidComdirectHeader(record []string) bool {

	expected := []string{
		"Buchungstag",
		"Wertstellung (Valuta)",
		"Vorgang",
		"Buchungstext",
		"Umsatz in EUR",
		"", // yes, there is an empty field
	}
	return reflect.DeepEqual(record, expected)
}

/*
	convertRecord converts a single record from comdirect to homebank format

Example:

	{
		"buchungstag": "05.08.2019",
		"vorgang": "Übertrag/Überweisung",
		"fullBuchungstext": "Auftraggeber:auftragnameBuchungstext: Der Buchungstext 123 456
		Empfänger:empfängernameEmpfänger: nameKto/IBAN: DE123 BLZ/BIC: ABC123",
		"umsatz": "-139.40"
	}

	->

	{
		"date": ISO 8601 date string like "2006-01-02"
		"payee": "empfängername",
		"info": "first three space seperated words of buchungstext",
		"memo": "the full buchungstext",
		"amount": 12.34,
	}
*/
func (c *comdirectRecord) convertRecord() (h homebankRecord) {
	h.payment = 0
	h.date = c.buchungstag.Format("2006-01-02")
	h.amount = c.umsatz_eur
	h.memo = c.fullBuchungstext
	h.info = getFirstNWords(3, c.buchungstext)

	// Get payee information. This makes only sense if amount is negative
	if h.amount < 0 {
		if c.auftraggeber != "" {
			// For e.g. Lastschrift there is no "Empfänger", but a "Auftraggeber" in the CSV
			h.payee = c.auftraggeber
		} else {
			// For "Kartenverfügung" there is no "Empfänger" set, but usually
			// the payee encoded in the buchungstext
			if c.vorgang == "Kartenverfügung" {
				h.payee = getFirstNWords(4, c.buchungstext)
			} else {
				h.payee = c.empfaenger
			}
		}
	}

	return
}

func getFirstNWords(n uint, s string) string {
	if n == 0 {
		return ""
	}
	split := strings.Fields(s)
	if len(split) < int(n) {
		return s
	}
	return strings.Join(split[:n], " ")
}
