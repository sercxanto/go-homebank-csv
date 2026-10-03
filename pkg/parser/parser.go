// Package parser provides parsers for different banking file formats.
//
// The several parsers implement one common interface: Parser.
package parser

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// SourceFormat is the source file format
type SourceFormat int

// Supported source format types
const (
	MoneyWallet SourceFormat = iota
	Barclaycard
	Volksbank
	Comdirect
	DKB
)

// sourceFormats is the internal mapping between SourceFormat and its textual representation
// it is used in the functions below to avoid duplicate code
var sourceFormats = map[SourceFormat]string{
	MoneyWallet: "MoneyWallet",
	Barclaycard: "Barclaycard",
	Volksbank:   "Volksbank",
	Comdirect:   "Comdirect",
	DKB:         "DKB",
}

// GetParser returns a parser for the given source format
func GetParser(s SourceFormat) Parser {
	switch s {
	case MoneyWallet:
		return &moneywalletParser{}
	case Barclaycard:
		return &barclaycardParser{}
	case Volksbank:
		return &volksbankParser{}
	case Comdirect:
		return &comdirectParser{}
	case DKB:
		return &dkbParser{}
	}
	return nil
}

// GetSourceFormats returns the list of supported source formats.
//
// The formats are returned in the order in which they are defined. Sorting is
// needed as the iteration order of a map is not specified: without it the
// result differs between calls, which shows up in the output of the
// "list-formats" command and in the order in which GetGuessedParser tries the
// parsers.
func GetSourceFormats() []SourceFormat {
	formats := make([]SourceFormat, 0, len(sourceFormats))
	for key := range sourceFormats {
		formats = append(formats, key)
	}
	slices.Sort(formats)
	return formats
}

// Returns the textual representation of the source format
// Returns "unknown format" if the format is not supported
func (s SourceFormat) String() string {
	for key, value := range sourceFormats {
		if key == s {
			return value
		}
	}
	return "unknown format"
}

func (s *SourceFormat) UnmarshalText(text []byte) error {
	textString := string(text)
	for key, value := range sourceFormats {
		if value == textString {
			*s = key
			return nil
		}
	}
	return fmt.Errorf("unsupported format '%s'", textString)
}

// NewSourceFormat returns a pointer to a new SourceFormat
func NewSourceFormat(value SourceFormat) *SourceFormat {
	return &value
}

// A ParserErrorType describes the type of error
type ParserErrorType int

const (
	IOError          ParserErrorType = iota // Error during file I/O
	HeaderError                             // Error in expected header
	DataParsingError                        // Error during parsing section
)

func (e ParserErrorType) String() string {
	switch e {
	case IOError:
		return "IOError"
	case HeaderError:
		return "HeaderError"
	case DataParsingError:
		return "DataParsingError"
	default:
		return "unknown error"
	}
}

// ParserError describes the error which could occur during parsing
type ParserError struct {
	ErrorType ParserErrorType

	// Optional line number where the error occurs. Line numbers are
	// 1 based. The value "0" means no line number applies here, e.g.
	// when no header has been found. Empty lines are not counted.
	Line int

	// Optional field name where the error occured
	Field string
}

func (e *ParserError) Error() string {
	var msg string
	msg = e.ErrorType.String()
	if e.Line > 0 {
		msg += fmt.Sprintf(" in line %d", e.Line)
	}
	if len(e.Field) > 0 {
		msg += fmt.Sprintf(" in field name '%s'", e.Field)
	}
	return msg
}

// Parser is the interface to be implemented by all parsers
type Parser interface {

	// Parse the given file into internal structure.
	ParseFile(filepath string) error

	// Returns the number of parsed entries.
	GetNumberOfEntries() int

	// Convert the internal structure into HomebankRecord CSV file.
	ConvertToHomebank(filepath string) error

	// Returns the format of the parser.
	GetFormat() SourceFormat
}

// GetGuessedParser tries to autodetect the file format.
// It iterates through the available, calls the ParseFile function and returns the
// first parser which does not fail with an error.
// It returns nil if no parser could be found.
func GetGuessedParser(filepath string) Parser {
	for _, f := range GetSourceFormats() {
		p := GetParser(f)
		if err := p.ParseFile(filepath); err == nil {
			return p
		}
	}
	return nil
}

// homebankRecord reflects the data in the CSV file,
// see http://homebank.free.fr/help/misc-csvformat.html
type homebankRecord struct {
	date     string
	payment  int8
	info     string
	payee    string
	memo     string
	amount   float64
	category string
	tags     string
}

// homebankFieldSeparator separates the fields of a homebank CSV file
const homebankFieldSeparator = ";"

// sanitizeHomebankField makes the content of a text field safe to write.
//
// The fields are separated by homebankFieldSeparator. A separator inside a
// field would shift all following fields of the record, a line break would end
// the record early. Both are replaced instead of quoted: the format does not
// document a way to quote or escape a field, so a quoted separator cannot be
// relied on to be understood by the importing side.
//
// Bank data does contain such characters, e.g. a "Verwendungszweck" with a
// semicolon, so dropping the affected records is not an option.
func sanitizeHomebankField(value string) string {
	value = strings.ReplaceAll(value, homebankFieldSeparator, ",")
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return value
}

// writeHomeBankRecords writes a slice of HomebankRecord to a CSV file
// See "Transaction import CSV format" under http://homebank.free.fr/help/misc-csvformat.html
func writeHomeBankRecords(records []homebankRecord, filepath string) error {
	outfile, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer outfile.Close()

	header := "date;payment;info;payee;memo;amount;category;tags"
	_, err = fmt.Fprintln(outfile, header)
	if err != nil {
		return err
	}

	for _, rec := range records {
		// date is a formatted timestamp, payment and amount are numbers, so
		// only the text fields can contain a separator or a line break
		line := fmt.Sprintf("%s;%d;%s;%s;%s;%f;%s;%s",
			rec.date, rec.payment,
			sanitizeHomebankField(rec.info),
			sanitizeHomebankField(rec.payee),
			sanitizeHomebankField(rec.memo),
			rec.amount,
			sanitizeHomebankField(rec.category),
			sanitizeHomebankField(rec.tags))
		_, err := fmt.Fprintln(outfile, line)
		if err != nil {
			return err
		}
	}
	return nil
}
