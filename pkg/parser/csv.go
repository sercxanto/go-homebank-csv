package parser

import (
	"encoding/csv"
	"errors"
	"io"
)

// csvRecord is a record of a CSV file together with the line numbers of its
// fields in the file
type csvRecord struct {
	fields []string
	lines  []int
}

// line returns the line number of the given field in the file. A quoted field
// can span several lines, so this is the line where the field starts.
func (r csvRecord) line(field int) int {
	return r.lines[field]
}

// readCSVRecords reads all records of r together with the line numbers of
// their fields.
//
// The line numbers count all lines of the file, like a text editor does. The
// position in the slice of records does not, as csv.Reader skips empty lines
// and a quoted field can span several lines.
//
// A malformed CSV file is reported as IOError with the line of the error.
func readCSVRecords(r *csv.Reader) ([]csvRecord, error) {
	var records []csvRecord
	for {
		fields, err := r.Read()
		if errors.Is(err, io.EOF) {
			return records, nil
		}
		if err != nil {
			parseErr := &ParseError{Type: IOError, Err: err}
			var csvErr *csv.ParseError
			if errors.As(err, &csvErr) {
				parseErr.Line = csvErr.Line
			}
			return nil, parseErr
		}
		lines := make([]int, len(fields))
		for i := range fields {
			lines[i], _ = r.FieldPos(i)
		}
		records = append(records, csvRecord{fields: fields, lines: lines})
	}
}
