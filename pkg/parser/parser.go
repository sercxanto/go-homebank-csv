// Package parser provides parsers for different banking file formats.
//
// The several parsers implement one common interface: Parser.
package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// New returns a new parser for the given source format.
// It returns an error if the format is not supported.
func New(s SourceFormat) (Parser, error) {
	if p := newParser(s); p != nil {
		return p, nil
	}
	return nil, fmt.Errorf("unsupported source format %d", int(s))
}

// newParser returns a new parser for the given source format, nil if the
// format is not supported
func newParser(s SourceFormat) Parser {
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

// SourceFormats returns the list of supported source formats.
//
// The formats are returned in the order in which they are defined. Sorting is
// needed as the iteration order of a map is not specified: without it the
// result differs between calls, which shows up in the output of the
// "list-formats" command and in the order in which Detect tries the
// parsers.
func SourceFormats() []SourceFormat {
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

// An ErrorType describes the type of a ParseError
type ErrorType int

const (
	IOError          ErrorType = iota // Error during file I/O
	HeaderError                       // Error in expected header
	DataParsingError                  // Error during parsing section
)

func (e ErrorType) String() string {
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

// ParseError describes the error which could occur during parsing
type ParseError struct {
	// Type of the error
	Type ErrorType

	// Optional line number where the error occurs. Line numbers are
	// 1 based. The value "0" means no line number applies here, e.g.
	// when no header has been found. Empty lines are not counted.
	Line int

	// Optional field name where the error occurred
	Field string

	// Optional underlying error which caused this error, e.g. the error
	// returned by os.Open or strconv.ParseFloat. It is accessible with
	// errors.Is and errors.As through Unwrap.
	Err error
}

func (e *ParseError) Error() string {
	var msg string
	msg = e.Type.String()
	if e.Line > 0 {
		msg += fmt.Sprintf(" in line %d", e.Line)
	}
	if len(e.Field) > 0 {
		msg += fmt.Sprintf(" in field name '%s'", e.Field)
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

// Unwrap returns the underlying error, nil if there is none
func (e *ParseError) Unwrap() error {
	return e.Err
}

// Parser is the interface to be implemented by all parsers
type Parser interface {

	// Parse the given file into internal structure.
	ParseFile(filepath string) error

	// Len returns the number of parsed entries.
	Len() int

	// Convert the internal structure into HomebankRecord CSV file.
	ConvertToHomebank(filepath string) error

	// SourceFormat returns the format of the parser.
	SourceFormat() SourceFormat
}

// ErrUnknownFormat is the error of Detect if no parser accepts the file.
var ErrUnknownFormat = errors.New("cannot deduce format")

// Detect tries to autodetect the file format.
// It iterates through the available formats, calls the ParseFile function and
// returns the first parser which does not fail with an error.
//
// If no parser accepts the file, the returned error wraps ErrUnknownFormat and
// the error of each parser, one line per format, e.g.
//
//	cannot deduce format
//	  MoneyWallet: HeaderError in line 1
//	  Barclaycard: IOError: zip: not a valid zip file
//	  ...
func Detect(filepath string) (Parser, error) {
	errs := []error{ErrUnknownFormat}
	// SourceFormats returns supported formats only, for which newParser
	// never returns nil
	for _, f := range SourceFormats() {
		p := newParser(f)
		err := p.ParseFile(filepath)
		if err == nil {
			return p, nil
		}
		errs = append(errs, fmt.Errorf("  %s: %w", f, err))
	}
	return nil, errors.Join(errs...)
}

// Payment types of a homebankRecord, as defined by the HomeBank CSV format.
// Only the types used by the parsers are listed.
const (
	paymentNone       int8 = 0
	paymentCreditCard int8 = 1
)

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

// tempFilePattern is the pattern for the name of the temporary file, which
// is appended to the output file name while it is written. os.CreateTemp
// replaces "*" with a random string.
const tempFilePattern = ".*.tmp"

// writeFileAtomically creates the file outPath with the content written by
// write.
//
// The content is written to a temporary file next to outPath, which is
// renamed to outPath only after it has been written and closed successfully.
// So a failure while writing, e.g. a full disk, never leaves an incomplete
// file behind. batchconvert relies on this, as it skips input files whose
// output file already exists.
//
// The temporary file gets a unique name. So it does not overwrite an
// unrelated file, and a temporary file left behind by an aborted run, e.g.
// after a power failure, does not block later runs. As it is created by
// os.CreateTemp, the file is only accessible by the current user.
func writeFileAtomically(outPath string, write func(w io.Writer) error) error {
	tempFile, err := os.CreateTemp(filepath.Dir(outPath), filepath.Base(outPath)+tempFilePattern)
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	writer := bufio.NewWriter(tempFile)
	err = write(writer)
	if err == nil {
		err = writer.Flush()
	}
	// Close reports errors of writes which have been delayed by the OS, so
	// it has to be checked before the file is considered complete
	if closeErr := tempFile.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tempPath, outPath)
	}
	if err != nil {
		_ = os.Remove(tempPath)
	}
	return err
}

// writeConvertedRecords converts the records of a parser with convert and
// writes them as homebank CSV file
func writeConvertedRecords[T any](entries []T, convert func(*T) homebankRecord, filepath string) error {
	records := make([]homebankRecord, 0, len(entries))
	for i := range entries {
		records = append(records, convert(&entries[i]))
	}
	return writeHomeBankRecords(records, filepath)
}

// writeHomeBankRecords writes a slice of HomebankRecord to a CSV file
// See "Transaction import CSV format" under http://homebank.free.fr/help/misc-csvformat.html
func writeHomeBankRecords(records []homebankRecord, filepath string) error {
	return writeFileAtomically(filepath, func(w io.Writer) error {
		return writeHomeBankCSV(w, records)
	})
}

// writeHomeBankCSV writes the records in the homebank CSV format. No header
// line is written, as HomeBank skips it on import, but warns about it.
func writeHomeBankCSV(w io.Writer, records []homebankRecord) error {
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
		_, err := fmt.Fprintln(w, line)
		if err != nil {
			return err
		}
	}
	return nil
}
