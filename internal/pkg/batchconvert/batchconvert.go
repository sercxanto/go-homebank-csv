// Package batchconvert implements convertig sets of files in batches.
package batchconvert

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sercxanto/go-homebank-csv/internal/pkg/settings"
	"github.com/sercxanto/go-homebank-csv/pkg/parser"
)

// getTimeFromMaxAgeDays returns the time.Time for the given fileMaxAgeDays
// if fileMaxAgeDays is 0, the zero time is returned (January 1, year 1, 00:00:00 UTC.)
func getTimeFromMaxAgeDays(fileMaxAgeDays uint, now time.Time) time.Time {
	if fileMaxAgeDays == 0 {
		return time.Time{}
	}
	return now.AddDate(0, 0, -int(fileMaxAgeDays))
}

// findFiles returns a list of files matching the given glob pattern and max age
//
// A file is considered matching if its modification time is younger than the given max age.
// A minTime of zero time (January 1, year 1, 00:00:00 UTC.) is considered matching all files.
// An empty fileGlobPattern is considered matching all files.
func findFiles(inputDir string, fileGlobPattern string, minTime time.Time) ([]string, error) {
	if len(inputDir) == 0 {
		return nil, nil
	}

	// Get list of files in inputDir
	if fileGlobPattern == "" {
		fileGlobPattern = "*"
	}
	files, err := filepath.Glob(filepath.Join(inputDir, fileGlobPattern))
	if err != nil {
		return nil, err
	}
	matchingFiles := make([]string, 0, len(files))
	for i := 0; i < len(files); i++ {
		fileInfo, err := os.Stat(files[i])
		if err != nil {
			return nil, err
		}
		if minTime.IsZero() {
			matchingFiles = append(matchingFiles, files[i])
		} else {
			modTime := fileInfo.ModTime()
			if modTime.After(minTime) || modTime.Equal(minTime) {
				matchingFiles = append(matchingFiles, files[i])
			}
		}
	}

	// sort matchingFiles alphabetically to keep the order consistent
	sort.Strings(matchingFiles)

	return matchingFiles, nil
}

const (
	NotStartedYet        = iota // Conversion has not started yet
	Skipped                     // File is skipped because it already exists in the output directory
	ConversionInProgress        // Conversion is in progress
	ConversionError             // Conversion failed
	ConversionSuccess           // Conversion was successful
)

type ConversionStatus int

// Conversion status of a single file
type FileStatus struct {
	InputFile  string               // Absolute path of the input file
	OutputFile string               // Absolute path of the output file. Only set after conversion started.
	Status     ConversionStatus     // Status of the conversion
	Format     *parser.SourceFormat // Detected source format
}

// Conversion status of a batch
type BatchSetStatus struct {
	Files []FileStatus // Status of found files in batch
	Name  string       // Name of the batch
}

// GetStats calculates the number of files that are done and the number of files that are left in the batch set status.
func (b BatchSetStatus) GetStats() (done uint, left uint) {
	for _, fileStatus := range b.Files {
		if fileStatus.Status == NotStartedYet {
			left++
		} else {
			done++
		}
	}
	return
}

// Conversion status of all sets
type BatchStatus []BatchSetStatus

// StatusCallback is a function that is called during the conversion process
// to report the progress of the conversion.
//
// It takes the following parameters:
//
//   - s: a BatchStatus struct containing the status of the conversion.
//   - userData: any user data that was passed to the BatchConvert function.
type StatusCallback func(s BatchStatus, userData interface{})

// BatchConvert is a function that performs batch conversion of files.
//
// It takes the following parameters:
//
//   - s: a settings.BatchConvertSet struct containing the settings for the batch conversion.
//   - now: a time.Time representing the current time.
//   - c: a StatusCallback function that is called during the conversion process.
//   - userData: any user data that was passed to the BatchConvert function.
//
// The converted files are placed in the output directory. The conversion happens only
// if the file with the same name does not exist yet in the output directory.
func BatchConvert(s settings.BatchConvertSettings, now time.Time, c StatusCallback, userData interface{}) (status BatchStatus, err error) {

	if len(s.Sets) == 0 {
		return nil, nil
	}

	if err := s.Sets.NormalizePaths(); err != nil {
		return nil, err
	}

	if err := s.Sets.CheckValidity(); err != nil {
		return nil, err
	}

	for setNr, set := range s.Sets {
		var fileInfo os.FileInfo
		fileInfo, err = os.Stat(set.OutputDir)
		if err != nil {
			return status, err
		}
		if !fileInfo.IsDir() {
			return status, errors.New("outputDir is not a directory")
		}

		status = append(status, BatchSetStatus{
			Files: []FileStatus{},
			Name:  set.Name,
		})

		var fileList []string
		fileList, err = findFiles(set.InputDir, set.FileGlobPattern, getTimeFromMaxAgeDays(uint(set.FileMaxAgeDays), now))
		if err != nil {
			return status, err
		}

		for _, infile := range fileList {
			status[setNr].Files = append(status[setNr].Files, FileStatus{
				InputFile: infile,
				Status:    NotStartedYet})
		}
		if c != nil {
			c(status, userData)
		}

		for fileNr, infile := range fileList {
			// get infile without extension
			outfileBasename := strings.TrimSuffix(infile, filepath.Ext(infile)) + ".csv"
			outfile := filepath.Join(set.OutputDir, filepath.Base(outfileBasename))
			status[setNr].Files[fileNr].OutputFile = outfile

			// Skip if output file already exists
			if _, err := os.Stat(outfile); err == nil {
				status[setNr].Files[fileNr].Status = Skipped
				if c != nil {
					c(status, userData)
				}
				continue
			}

			var fileParser parser.Parser
			status[setNr].Files[fileNr].Status = ConversionInProgress
			if c != nil {
				c(status, userData)
			}

			if set.Format == nil {
				fileParser = parser.GetGuessedParser(infile)
				if fileParser == nil {
					status[setNr].Files[fileNr].Status = ConversionError
					if c != nil {
						c(status, userData)
					}
					continue
				}
			} else {
				fileParser = parser.GetParser(*set.Format)
				if err := fileParser.ParseFile(infile); err != nil {
					status[setNr].Files[fileNr].Status = ConversionError
					if c != nil {
						c(status, userData)
					}
					continue
				}
			}
			status[setNr].Files[fileNr].Format = parser.NewSourceFormat(fileParser.GetFormat())
			if err := fileParser.ConvertToHomebank(outfile); err != nil {
				status[setNr].Files[fileNr].Status = ConversionError
				if c != nil {
					c(status, userData)
				}
				continue
			}
			status[setNr].Files[fileNr].Status = ConversionSuccess
			if c != nil {
				c(status, userData)
			}

		}
	}
	return

}
