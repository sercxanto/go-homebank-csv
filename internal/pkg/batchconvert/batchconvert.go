// Package batchconvert implements converting sets of files in batches.
package batchconvert

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
// Directories are not part of the list, even if they match.
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
		// Only files are converted, the input directory is not searched
		// recursively
		if fileInfo.IsDir() {
			continue
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
	slices.Sort(matchingFiles)

	return matchingFiles, nil
}

// ConversionStatus is the status of the conversion of a single file
type ConversionStatus int

const (
	NotStartedYet        ConversionStatus = iota // Conversion has not started yet
	Skipped                                      // File is skipped because its output file exists and is not older than the input file
	ConversionInProgress                         // Conversion is in progress
	ConversionError                              // Conversion failed
	ConversionSuccess                            // Conversion was successful
)

// Conversion status of a single file
type FileStatus struct {
	InputFile  string               // Absolute path of the input file
	OutputFile string               // Absolute path of the output file. Only set after conversion started.
	Status     ConversionStatus     // Status of the conversion
	Overwrite  bool                 // An older output file exists and is replaced by the conversion
	Format     *parser.SourceFormat // Detected source format
	Err        error                // Cause of the failure if Status is ConversionError
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
type StatusCallback func(s BatchStatus, userData any)

// ErrOutputCollision is the error of input files which would be converted to
// the same output file as other input files. It is wrapped together with the
// names of the other input files.
var ErrOutputCollision = errors.New("output file name collides with other input files")

// outputFileName returns the path of the output file for the given input file
func outputFileName(infile string, outputDir string) string {
	outfileBasename := strings.TrimSuffix(filepath.Base(infile), filepath.Ext(infile)) + ".csv"
	return filepath.Join(outputDir, outfileBasename)
}

// outputFileKey returns the key to compare output file paths with.
//
// Case is ignored as the output directory may be on a case-insensitive file
// system, e.g. on Windows, macOS or a synchronized drive. There two input
// files which differ only in case would be converted to the same output file.
func outputFileKey(path string) string {
	if absPath, err := filepath.Abs(path); err == nil {
		path = absPath
	}
	return strings.ToLower(path)
}

// markOutputCollisions sets the status of all files which share an output
// file with other files to ConversionError.
//
// Which of these files would end up in the output file depends on the order
// of the conversion and the modification times, the others would be skipped
// or overwritten without notice. So none of them is converted.
//
// outfiles holds the output file of each file in status. The returned errors
// are in the order of status.
func markOutputCollisions(status BatchStatus, outfiles [][]string) []error {
	inputFilesByKey := make(map[string][]string)
	for setNr := range status {
		for fileNr, fileStatus := range status[setNr].Files {
			key := outputFileKey(outfiles[setNr][fileNr])
			inputFilesByKey[key] = append(inputFilesByKey[key], fileStatus.InputFile)
		}
	}

	var fileErrors []error
	for setNr := range status {
		for fileNr := range status[setNr].Files {
			fileStatus := &status[setNr].Files[fileNr]
			inputFiles := inputFilesByKey[outputFileKey(outfiles[setNr][fileNr])]
			if len(inputFiles) < 2 {
				continue
			}
			otherInputFiles := slices.DeleteFunc(slices.Clone(inputFiles), func(f string) bool {
				return f == fileStatus.InputFile
			})
			fileStatus.OutputFile = outfiles[setNr][fileNr]
			fileStatus.Status = ConversionError
			fileStatus.Err = fmt.Errorf("%w: %s", ErrOutputCollision, strings.Join(otherInputFiles, ", "))
			fileErrors = append(fileErrors, fmt.Errorf("%s: %w", fileStatus.InputFile, fileStatus.Err))
		}
	}
	return fileErrors
}

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
// if the file with the same name does not exist yet in the output directory or if
// it is older than the input file, e.g. because the input file has been
// downloaded again. In the latter case the output file is replaced.
//
// The input files of all sets are searched before any file is converted.
// Input files which would be converted to the same output file, e.g.
// "Umsaetze.csv" and "Umsaetze.xlsx" or files of different sets sharing an
// output directory, are not converted. Their cause is ErrOutputCollision.
//
// A file which fails to convert does not stop the conversion of the remaining
// files. Its cause is reported in FileStatus.Err and, together with the causes
// of all other failed files, in the returned error.
func BatchConvert(s settings.BatchConvertSettings, now time.Time, c StatusCallback, userData any) (status BatchStatus, err error) {

	if len(s.Sets) == 0 {
		return nil, nil
	}

	if err := s.Sets.NormalizePaths(); err != nil {
		return nil, err
	}

	if err := s.Sets.CheckValidity(); err != nil {
		return nil, err
	}

	// Output file of each found input file, indexed like status
	outfiles := make([][]string, 0, len(s.Sets))

	for _, set := range s.Sets {
		var fileInfo os.FileInfo
		fileInfo, err = os.Stat(set.OutputDir)
		if err != nil {
			return status, fmt.Errorf("set '%s': outputdir: %w", set.Name, err)
		}
		if !fileInfo.IsDir() {
			return status, fmt.Errorf("set '%s': outputdir '%s' is not a directory", set.Name, set.OutputDir)
		}

		var fileList []string
		fileList, err = findFiles(set.InputDir, set.FileGlobPattern, getTimeFromMaxAgeDays(uint(set.FileMaxAgeDays), now))
		if err != nil {
			return status, fmt.Errorf("set '%s': %w", set.Name, err)
		}

		setStatus := BatchSetStatus{
			Files: []FileStatus{},
			Name:  set.Name,
		}
		setOutfiles := make([]string, 0, len(fileList))
		for _, infile := range fileList {
			setStatus.Files = append(setStatus.Files, FileStatus{
				InputFile: infile,
				Status:    NotStartedYet})
			setOutfiles = append(setOutfiles, outputFileName(infile, set.OutputDir))
		}
		status = append(status, setStatus)
		outfiles = append(outfiles, setOutfiles)
	}

	// Causes of the failed files, prefixed with the file name
	fileErrors := markOutputCollisions(status, outfiles)
	if c != nil {
		c(status, userData)
	}

	for setNr, set := range s.Sets {
		for fileNr, fileStatus := range status[setNr].Files {
			if fileStatus.Status != NotStartedYet {
				continue
			}
			infile := fileStatus.InputFile
			outfile := outfiles[setNr][fileNr]
			status[setNr].Files[fileNr].OutputFile = outfile

			// Skip if output file already exists and is up to date
			if outInfo, err := os.Stat(outfile); err == nil {
				inInfo, err := os.Stat(infile)
				if err != nil {
					status[setNr].Files[fileNr].Status = ConversionError
					status[setNr].Files[fileNr].Err = err
					fileErrors = append(fileErrors, fmt.Errorf("%s: %w", infile, err))
					if c != nil {
						c(status, userData)
					}
					continue
				}
				if !inInfo.ModTime().After(outInfo.ModTime()) {
					status[setNr].Files[fileNr].Status = Skipped
					if c != nil {
						c(status, userData)
					}
					continue
				}
				status[setNr].Files[fileNr].Overwrite = true
			}

			var fileParser parser.Parser
			status[setNr].Files[fileNr].Status = ConversionInProgress
			if c != nil {
				c(status, userData)
			}

			if set.Format == nil {
				var err error
				fileParser, err = parser.GuessParser(infile)
				if err != nil {
					status[setNr].Files[fileNr].Status = ConversionError
					status[setNr].Files[fileNr].Err = err
					fileErrors = append(fileErrors, fmt.Errorf("%s: %w", infile, err))
					if c != nil {
						c(status, userData)
					}
					continue
				}
			} else {
				fileParser = parser.GetParser(*set.Format)
				if err := fileParser.ParseFile(infile); err != nil {
					status[setNr].Files[fileNr].Status = ConversionError
					status[setNr].Files[fileNr].Err = err
					fileErrors = append(fileErrors, fmt.Errorf("%s: %w", infile, err))
					if c != nil {
						c(status, userData)
					}
					continue
				}
			}
			status[setNr].Files[fileNr].Format = parser.NewSourceFormat(fileParser.GetFormat())
			if err := fileParser.ConvertToHomebank(outfile); err != nil {
				status[setNr].Files[fileNr].Status = ConversionError
				status[setNr].Files[fileNr].Err = err
				fileErrors = append(fileErrors, fmt.Errorf("%s: %w", infile, err))
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
	return status, errors.Join(fileErrors...)
}
