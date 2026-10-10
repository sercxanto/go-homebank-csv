package batchconvert

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sercxanto/go-homebank-csv/internal/pkg/settings"
	"github.com/sercxanto/go-homebank-csv/pkg/parser"
)

type fileEntry struct {
	Filename string
	ModTime  time.Time
}

type fileList []fileEntry

type findFilesInputData struct {
	FileGlobPattern string
	MinTime         time.Time
	ExpectedFiles   []string
}

type findFilesInputDataList []findFilesInputData

func (f fileList) createFiles(directory string) error {
	for _, entry := range f {
		filePath := filepath.Join(directory, entry.Filename)
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		file.Close()
		if !entry.ModTime.IsZero() {
			err = os.Chtimes(filePath, entry.ModTime, entry.ModTime)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getFilesInDirectory(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func areFilesEqual(file1, file2 string) (bool, error) {
	content1, err := os.ReadFile(file1)
	if err != nil {
		return false, err
	}

	content2, err := os.ReadFile(file2)
	if err != nil {
		return false, err
	}

	// Trim possible all line endings to avoid differences on Windows
	// and with git autocrlf settings
	return strings.ReplaceAll(string(content1), "\r", "") == strings.ReplaceAll(string(content2), "\r", ""), nil
}

func extractFileNames(paths []string) []string {
	var fileNames []string
	for _, path := range paths {
		fileName := filepath.Base(path)
		fileNames = append(fileNames, fileName)
	}
	return fileNames
}

func areDirectoriesEqual(dir1, dir2 string) (equal bool, reason string, err error) {
	files1, err := getFilesInDirectory(dir1)
	if err != nil {
		return false, "", fmt.Errorf("Failed to get files in directory '%s': '%w'", dir1, err)
	}

	files2, err := getFilesInDirectory(dir2)
	if err != nil {
		return false, "", fmt.Errorf("Failed to get files in directory '%s': '%w'", dir2, err)
	}

	files1BaseName := extractFileNames(files1)
	files2BaseName := extractFileNames(files2)

	if !reflect.DeepEqual(files1BaseName, files2BaseName) {
		reason := fmt.Sprintf("Filelist ist not equal '%s' (%s) and '%s' (%s)", dir1, files1BaseName, dir2, files2BaseName)
		return false, reason, nil
	}

	// sort files1 and files2
	sort.Strings(files1)
	sort.Strings(files2)

	for i := range files1 {
		equal, err := areFilesEqual(files1[i], files2[i])
		if err != nil {
			return false, "", err
		}
		if !equal {
			reason := fmt.Sprintf("Files '%s' and '%s' are not equal", files1[i], files2[i])
			return false, reason, nil
		}
	}

	return true, "", nil
}

func copyFile(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func TestFindFiles(t *testing.T) {

	outList, err := findFiles("", "", time.Time{})
	if err != nil {
		t.Fatalf("findFiles return error '%s'", err)
	}
	if len(outList) != 0 {
		t.Fatalf("findFiles should return nil list")
	}

	outList, err = findFiles("non-existent-path", "*", time.Time{})
	if err != nil {
		t.Fatalf("findFiles return error '%s'", err)
	}
	if len(outList) != 0 {
		t.Fatalf("findFiles should return nil list")
	}

	_, err = findFiles("non-existent-path", "[", time.Time{})
	if err == nil {
		t.Fatalf("findFiles should return error")
	}

	tmpDir := t.TempDir()
	now := time.Now()
	testFiles := &fileList{
		{"file1.ext1", getTimeFromMaxAgeDays(2, now)},
		{"file2.ext2", getTimeFromMaxAgeDays(3, now)},
		{"file3.csv", getTimeFromMaxAgeDays(0, now)},
		{"file4.csv", getTimeFromMaxAgeDays(1, now)},
		{"file5.csv", getTimeFromMaxAgeDays(2, now)}}
	if err := testFiles.createFiles(tmpDir); err != nil {
		t.Fatalf("Failed to create files in '%s'", tmpDir)
	}
	// Directories matching the patterns are not part of the result
	for _, dir := range []string{"subdir", "subdir.csv"} {
		if err := os.Mkdir(filepath.Join(tmpDir, dir), 0o700); err != nil {
			t.Fatalf("Failed to create directory '%s'", dir)
		}
	}

	input := &findFilesInputDataList{
		{"", getTimeFromMaxAgeDays(0, now), []string{
			filepath.Join(tmpDir, "file1.ext1"),
			filepath.Join(tmpDir, "file2.ext2"),
			filepath.Join(tmpDir, "file3.csv"),
			filepath.Join(tmpDir, "file4.csv"),
			filepath.Join(tmpDir, "file5.csv")}},
		{"*.ext1", getTimeFromMaxAgeDays(0, now), []string{
			filepath.Join(tmpDir, "file1.ext1")}},
		{"*.ext2", getTimeFromMaxAgeDays(0, now), []string{
			filepath.Join(tmpDir, "file2.ext2")}},
		{"*.csv", getTimeFromMaxAgeDays(0, now), []string{
			filepath.Join(tmpDir, "file3.csv"),
			filepath.Join(tmpDir, "file4.csv"),
			filepath.Join(tmpDir, "file5.csv")}},
		{"*.csv", getTimeFromMaxAgeDays(1, now), []string{
			filepath.Join(tmpDir, "file3.csv"),
			filepath.Join(tmpDir, "file4.csv")}},
		{"*.csv", getTimeFromMaxAgeDays(2, now), []string{
			filepath.Join(tmpDir, "file3.csv"),
			filepath.Join(tmpDir, "file4.csv"),
			filepath.Join(tmpDir, "file5.csv")}},
		{"*.ext*", getTimeFromMaxAgeDays(1, now), []string{}},
	}

	for nr, entry := range *input {
		outList, err := findFiles(tmpDir, entry.FileGlobPattern, entry.MinTime)
		if err != nil {
			t.Fatalf("findFiles return error '%s'", err)
		}
		if !reflect.DeepEqual(outList, entry.ExpectedFiles) {
			t.Errorf("Testcase %d:Expected %v, got %v", nr, entry.ExpectedFiles, outList)
		}
	}
}

func TestBatchConvertNoSets(t *testing.T) {
	settings := settings.BatchConvertSettings{}
	status, err := BatchConvert(settings, time.Now(), nil, nil)
	if err != nil {
		t.Fatalf("BatchConvert should return error")
	}
	if status != nil {
		t.Fatalf("BatchConvert should return nil status")
	}
}

func TestBatchConvertInvalidSet(t *testing.T) {
	settings := settings.BatchConvertSettings{
		Sets: []settings.BatchConvertSet{
			{
				Name:      "",
				Format:    nil,
				InputDir:  "",
				OutputDir: "",
			},
		},
	}
	status, err := BatchConvert(settings, time.Now(), nil, nil)
	if err == nil {
		t.Fatalf("BatchConvert should return error")
	}
	if status != nil {
		t.Fatalf("BatchConvert should return nil status")
	}
}

func TestBatchConvertNonExistentOutputDir(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %s", err)
	}
	settings := settings.BatchConvertSettings{
		Sets: []settings.BatchConvertSet{
			{
				Name:      "my name",
				Format:    nil,
				InputDir:  workingDir,
				OutputDir: "/some/non-existing/dir",
			},
		},
	}
	if _, err = BatchConvert(settings, time.Now(), nil, nil); err == nil {
		t.Fatalf("BatchConvert should return error")
	}
	// The error names the set and keeps the cause
	if !strings.Contains(err.Error(), "set 'my name': outputdir") {
		t.Errorf("Expected the error to name the set, got '%s'", err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Expected the error to wrap fs.ErrNotExist, got '%s'", err)
	}
}

func TestBatchConvertOutputDirNotDir(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %s", err)
	}
	tmpDir := t.TempDir()
	// create an empty file "testfile" in tmpDir
	testfilePath := filepath.Join(tmpDir, "testfile")
	file, err := os.Create(testfilePath)
	if err != nil {
		t.Fatalf("Failed to create file: %s", err)
	}
	file.Close()
	settings := settings.BatchConvertSettings{
		Sets: []settings.BatchConvertSet{
			{
				Name:      "my name",
				Format:    nil,
				InputDir:  workingDir,
				OutputDir: testfilePath,
			},
		},
	}
	if _, err = BatchConvert(settings, time.Now(), nil, nil); err == nil {
		t.Fatalf("BatchConvert should return error")
	}
	expected := "set 'my name': outputdir '" + testfilePath + "' is not a directory"
	if err.Error() != expected {
		t.Errorf("Expected error %q, got '%s'", expected, err)
	}
}

func TestBatchConvertConversionError(t *testing.T) {
	tmpDir := t.TempDir()

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")
	if err := os.Mkdir(inputDir, os.ModeDir|0o700); err != nil {
		t.Fatalf("Failed to create directory '%s'", inputDir)
	}
	if err := os.Mkdir(outputDir, os.ModeDir|0o700); err != nil {
		t.Fatalf("Failed to create directory '%s'", outputDir)
	}

	// Write empty file in inputDir
	emptyFilePath := filepath.Join(inputDir, "emptyfile")
	var emptyFile *os.File
	var err error
	emptyFile, err = os.Create(emptyFilePath)
	if err != nil {
		t.Fatalf("Failed to create empty file: %s", err)
	}
	emptyFile.Close()

	settings1 := settings.BatchConvertSettings{
		Sets: []settings.BatchConvertSet{
			{
				Name:      "set 1",
				Format:    nil,
				InputDir:  inputDir,
				OutputDir: outputDir,
			},
		},
	}

	settings2 := settings.BatchConvertSettings{
		Sets: []settings.BatchConvertSet{
			{
				Name:      "set 1",
				Format:    parser.NewSourceFormat(parser.Volksbank),
				InputDir:  inputDir,
				OutputDir: outputDir,
			},
		},
	}

	var cbStatus, status BatchStatus
	cbUserData := 42
	cbUpdateNr := 0

	cb := func(s BatchStatus, userData interface{}) {
		cbStatus = s
		cbUpdateNr++
		if userData == nil {
			t.Fatalf("cbUserData is nil")
		}
		if val, ok := userData.(int); !ok || val != cbUserData {
			t.Fatalf("cbUserData is not '%d', but '%d'", cbUserData, val)
		}
		if len(s) != 1 {
			t.Fatalf("len(s) is not 1")
		}
		if len(s[0].Files) != 1 {
			t.Fatalf("len(s[0].Files) is not 1, but %d (%v)", len(s[0].Files), s[0])
		}
		if cbUpdateNr == 1 {
			if s[0].Files[0].Status != NotStartedYet {
				t.Fatalf("s[0].Files[0].Status is not NotStartedYet, but '%v'", s[0].Files[0].Status)
			}
		}
		if cbUpdateNr == 2 {
			if s[0].Files[0].Status != ConversionInProgress {
				t.Fatalf("s[0].Files[0].Status is not ConversionInProgress, but '%v'", s[0].Files[0].Status)
			}
		}
		if cbUpdateNr == 3 {
			if s[0].Files[0].Status != ConversionError {
				t.Fatalf("s[0].Files[0].Status is not ConversionError, but '%v'", s[0].Files[0].Status)
			}
		}
	}

	// The format of the empty file cannot be guessed
	if status, err = BatchConvert(settings1, time.Now(), cb, cbUserData); err == nil {
		t.Fatalf("BatchConvert should return error")
	}
	if !strings.Contains(err.Error(), emptyFilePath) {
		t.Errorf("Expected the error to name the failed file, got '%s'", err)
	}
	if status[0].Files[0].Err == nil {
		t.Errorf("Expected the cause of the failure in the file status")
	}

	if !reflect.DeepEqual(status, cbStatus) {
		t.Fatalf("status and cbStatus are not equal")
	}

	// The empty file is no valid Volksbank file, the ParseError is kept
	cbUpdateNr = 0
	if status, err = BatchConvert(settings2, time.Now(), cb, cbUserData); err == nil {
		t.Fatalf("BatchConvert should return error")
	}
	var pError *parser.ParseError
	if !errors.As(err, &pError) || pError.Type != parser.HeaderError {
		t.Errorf("Expected a HeaderError in the returned error, got '%v'", err)
	}
	if !errors.As(status[0].Files[0].Err, &pError) || pError.Type != parser.HeaderError {
		t.Errorf("Expected a HeaderError in the file status, got '%v'", status[0].Files[0].Err)
	}

	if !reflect.DeepEqual(status, cbStatus) {
		t.Fatalf("status and cbStatus are not equal")
	}

}

// TestBatchConvertBasic tests a conversion of two BatchConvertSets and compares the OutputDir
// and returned status
func TestBatchConvertBasic(t *testing.T) {

	testfilesBase, err := filepath.Abs("testfiles")
	if err != nil {
		t.Fatalf("Failed to get absolute path to 'testfiles': %s", err)
	}
	tmpDir := t.TempDir()

	volksbankInputDir := filepath.Join(testfilesBase, "input", "volksbank")
	volksbankOutputDir := filepath.Join(tmpDir, "volksbank")
	if err := os.Mkdir(volksbankOutputDir, os.ModeDir|0o700); err != nil {
		t.Fatalf("Failed to create directory '%s'", volksbankOutputDir)
	}
	volksbankExpectedDir := filepath.Join(testfilesBase, "expected_output", "volksbank")
	sVolksbank := settings.BatchConvertSet{
		Name:      "volksbank",
		Format:    parser.NewSourceFormat(parser.Volksbank),
		InputDir:  volksbankInputDir,
		OutputDir: volksbankOutputDir,
	}

	mixedInputDir := filepath.Join(testfilesBase, "input", "mixed")
	mixedOutputDir := filepath.Join(tmpDir, "mixed")
	if err := os.Mkdir(mixedOutputDir, os.ModeDir|0o700); err != nil {
		t.Fatalf("Failed to create directory '%s'", mixedOutputDir)
	}
	mixedExpectedDir := filepath.Join(testfilesBase, "expected_output", "mixed")
	sMixed := settings.BatchConvertSet{
		Name:      "mixed",
		InputDir:  mixedInputDir,
		OutputDir: mixedOutputDir,
	}

	settings := settings.BatchConvertSettings{
		Sets: []settings.BatchConvertSet{sVolksbank, sMixed},
	}

	expectetedStatus := BatchStatus{
		{
			Name: "volksbank",
			Files: []FileStatus{
				{
					InputFile:  filepath.Join(volksbankInputDir, "Umsaetze_DE12345678901234567890_2023.10.04.csv"),
					OutputFile: filepath.Join(volksbankOutputDir, "Umsaetze_DE12345678901234567890_2023.10.04.csv"),
					Status:     ConversionSuccess,
					Format:     parser.NewSourceFormat(parser.Volksbank),
				},
			},
		},
		{
			Name: "mixed",
			Files: []FileStatus{
				{
					InputFile:  filepath.Join(mixedInputDir, "Umsaetze.xlsx"),
					OutputFile: filepath.Join(mixedOutputDir, "Umsaetze.csv"),
					Status:     ConversionSuccess,
					Format:     parser.NewSourceFormat(parser.Barclaycard),
				},
				{
					InputFile:  filepath.Join(mixedInputDir, "Umsaetze_DE12345678901234567890_2023.10.04.csv"),
					OutputFile: filepath.Join(mixedOutputDir, "Umsaetze_DE12345678901234567890_2023.10.04.csv"),
					Status:     ConversionSuccess,
					Format:     parser.NewSourceFormat(parser.Volksbank),
				},
			},
		},
	}

	var status BatchStatus
	var cbStatus BatchStatus
	cbUserData := 42

	cb := func(s BatchStatus, userData interface{}) {
		cbStatus = s
		if userData == nil {
			t.Fatalf("cbUserData is nil")
		}
		if val, ok := userData.(int); !ok || val != cbUserData {
			t.Fatalf("cbUserData is not '%d', but '%d'", cbUserData, val)
		}
	}

	status, err = BatchConvert(settings, time.Time{}, cb, cbUserData)

	if err != nil {
		t.Fatalf("BatchConvert return error '%s'", err)
	}

	if !reflect.DeepEqual(status, expectetedStatus) {
		t.Fatalf("BatchConvert return wrong status. Status: %v, Expected: %v", status, expectetedStatus)
	}

	if !reflect.DeepEqual(status, cbStatus) {
		t.Fatalf("BatchConvert return status and callback status do not match. Return status: %v, CB status: %v", status, cbStatus)
	}

	done, left := status[0].GetStats()
	if done != 1 || left != 0 {
		t.Fatalf("BatchConvert return wrong status")
	}

	done, left = status[1].GetStats()
	if done != 2 || left != 0 {
		t.Fatalf("BatchConvert return wrong status")
	}

	areEqual, reason, err := areDirectoriesEqual(volksbankExpectedDir, volksbankOutputDir)
	if err != nil {
		t.Fatalf("areDirectoriesEqual return error '%s'", err)
	}
	if !areEqual {
		t.Errorf("Output directory does not match expected directory. Reason: %s", reason)
	}

	areEqual, reason, err = areDirectoriesEqual(mixedExpectedDir, mixedOutputDir)
	if err != nil {
		t.Fatalf("areDirectoriesEqual return error '%s'", err)
	}
	if !areEqual {
		t.Errorf("Output directory does not match expected directory. Reason: %s", reason)
	}
}

// TestBatchConvertSkipped tests a BatchConvertSet with two files where one of it has
// already been converted
func TestBatchConvertSkipped(t *testing.T) {
	testfilesBase, err := filepath.Abs("testfiles")
	if err != nil {
		t.Fatalf("Failed to get absolute path to 'testfiles': %s", err)
	}
	tmpDir := t.TempDir()

	mixedInputDir := filepath.Join(testfilesBase, "input", "mixed")
	mixedOutputDir := filepath.Join(tmpDir, "mixed")
	if err := os.Mkdir(mixedOutputDir, os.ModeDir|0o700); err != nil {
		t.Fatalf("Failed to create directory '%s'", mixedOutputDir)
	}

	// Simulate that one of the files has been converted
	mixedExpectedDir := filepath.Join(testfilesBase, "expected_output", "mixed")
	err = copyFile(filepath.Join(mixedExpectedDir, "Umsaetze.csv"), filepath.Join(mixedOutputDir, "Umsaetze.csv"))
	if err != nil {
		t.Fatalf("Failed to copy file '%s' to '%s'", filepath.Join(mixedExpectedDir, "Umsaetze.csv"), filepath.Join(mixedOutputDir, "Umsaetze.csv"))
	}
	// Make sure that the output file is newer than the input file
	inInfo, err := os.Stat(filepath.Join(mixedInputDir, "Umsaetze.xlsx"))
	if err != nil {
		t.Fatalf("Failed to stat input file: %s", err)
	}
	outModTime := inInfo.ModTime().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(mixedOutputDir, "Umsaetze.csv"), outModTime, outModTime); err != nil {
		t.Fatalf("Failed to set modification time: %s", err)
	}

	sMixed := settings.BatchConvertSet{
		Name:      "mixed",
		InputDir:  mixedInputDir,
		OutputDir: mixedOutputDir,
	}

	settings := settings.BatchConvertSettings{
		Sets: []settings.BatchConvertSet{sMixed},
	}

	expectetedStatus := BatchStatus{

		{
			Name: "mixed",
			Files: []FileStatus{
				{
					InputFile:  filepath.Join(mixedInputDir, "Umsaetze.xlsx"),
					OutputFile: filepath.Join(mixedOutputDir, "Umsaetze.csv"),
					Status:     Skipped,
					Format:     nil,
				},
				{
					InputFile:  filepath.Join(mixedInputDir, "Umsaetze_DE12345678901234567890_2023.10.04.csv"),
					OutputFile: filepath.Join(mixedOutputDir, "Umsaetze_DE12345678901234567890_2023.10.04.csv"),
					Status:     ConversionSuccess,
					Format:     parser.NewSourceFormat(parser.Volksbank),
				},
			},
		},
	}

	var status BatchStatus
	var cbStatus BatchStatus
	cbUserData := 42

	cb := func(s BatchStatus, userData interface{}) {
		cbStatus = s
		if userData == nil {
			t.Fatalf("cbUserData is nil")
		}
		if val, ok := userData.(int); !ok || val != cbUserData {
			t.Fatalf("cbUserData is not '%d', but '%d'", cbUserData, val)
		}
		for _, set := range s {
			for _, f := range set.Files {
				if f.OutputFile == filepath.Join(mixedOutputDir, "Umsaetze.csv") {
					if f.Status != Skipped {
						t.Fatalf("Did not skip 'Umsaetze.csv'")
					}
				}
			}
		}
	}

	status, err = BatchConvert(settings, time.Time{}, cb, cbUserData)

	if err != nil {
		t.Fatalf("BatchConvert return error '%s'", err)
	}

	if !reflect.DeepEqual(status, expectetedStatus) {
		t.Fatalf("BatchConvert return wrong status. Status: %v, Expected: %v", status, expectetedStatus)
	}

	if !reflect.DeepEqual(status, cbStatus) {
		t.Fatalf("BatchConvert return status and callback status do not match. Return status: %v, CB status: %v", status, cbStatus)
	}

	done, left := status[0].GetStats()
	if done != 2 || left != 0 {
		t.Fatalf("BatchConvert return wrong status")
	}

	areEqual, reason, err := areDirectoriesEqual(mixedExpectedDir, mixedOutputDir)
	if err != nil {
		t.Fatalf("areDirectoriesEqual return error '%s'", err)
	}
	if !areEqual {
		t.Errorf("Output directory does not match expected directory. Reason: %s", reason)
	}
}

// TestBatchConvertOutputOlder tests that an existing output file is replaced if
// it is older than the input file and kept if it is not
func TestBatchConvertOutputOlder(t *testing.T) {
	testfilesBase, err := filepath.Abs("testfiles")
	if err != nil {
		t.Fatalf("Failed to get absolute path to 'testfiles': %s", err)
	}
	mixedExpectedDir := filepath.Join(testfilesBase, "expected_output", "mixed")
	inputModTime := time.Date(2023, 10, 4, 12, 0, 0, 0, time.UTC)
	staleContent := "stale output\n"

	testCases := []struct {
		name            string
		outputModTime   time.Time
		expectedStatus  ConversionStatus
		expectOverwrite bool
	}{
		{"output older", inputModTime.Add(-time.Hour), ConversionSuccess, true},
		{"output same age", inputModTime, Skipped, false},
		{"output newer", inputModTime.Add(time.Hour), Skipped, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputDir := filepath.Join(tmpDir, "input")
			outputDir := filepath.Join(tmpDir, "output")
			for _, dir := range []string{inputDir, outputDir} {
				if err := os.Mkdir(dir, os.ModeDir|0o700); err != nil {
					t.Fatalf("Failed to create directory '%s'", dir)
				}
			}

			inFile := filepath.Join(inputDir, "Umsaetze.xlsx")
			if err := copyFile(filepath.Join(testfilesBase, "input", "mixed", "Umsaetze.xlsx"), inFile); err != nil {
				t.Fatalf("Failed to copy input file: %s", err)
			}
			if err := os.Chtimes(inFile, inputModTime, inputModTime); err != nil {
				t.Fatalf("Failed to set modification time: %s", err)
			}

			outFile := filepath.Join(outputDir, "Umsaetze.csv")
			if err := os.WriteFile(outFile, []byte(staleContent), 0o600); err != nil {
				t.Fatalf("Failed to write output file: %s", err)
			}
			if err := os.Chtimes(outFile, tc.outputModTime, tc.outputModTime); err != nil {
				t.Fatalf("Failed to set modification time: %s", err)
			}

			s := settings.BatchConvertSettings{
				Sets: []settings.BatchConvertSet{{
					Name:      "mixed",
					InputDir:  inputDir,
					OutputDir: outputDir,
				}},
			}

			status, err := BatchConvert(s, time.Time{}, nil, nil)
			if err != nil {
				t.Fatalf("BatchConvert return error '%s'", err)
			}
			if len(status) != 1 || len(status[0].Files) != 1 {
				t.Fatalf("BatchConvert return wrong number of files: %v", status)
			}
			f := status[0].Files[0]
			if f.Status != tc.expectedStatus {
				t.Errorf("Status is %d, expected %d", f.Status, tc.expectedStatus)
			}
			if f.Overwrite != tc.expectOverwrite {
				t.Errorf("Overwrite is %t, expected %t", f.Overwrite, tc.expectOverwrite)
			}

			if tc.expectOverwrite {
				equal, err := areFilesEqual(filepath.Join(mixedExpectedDir, "Umsaetze.csv"), outFile)
				if err != nil {
					t.Fatalf("areFilesEqual return error '%s'", err)
				}
				if !equal {
					t.Errorf("Output file has not been replaced with the converted file")
				}
			} else {
				content, err := os.ReadFile(outFile)
				if err != nil {
					t.Fatalf("Failed to read output file: %s", err)
				}
				if string(content) != staleContent {
					t.Errorf("Output file has been modified")
				}
			}
		})
	}
}

// TestBatchConvertOutputCollision tests that input files which would be
// converted to the same output file are not converted, while the other files
// are
func TestBatchConvertOutputCollision(t *testing.T) {
	testfilesBase, err := filepath.Abs("testfiles")
	if err != nil {
		t.Fatalf("Failed to get absolute path to 'testfiles': %s", err)
	}
	barclaycardFile := filepath.Join(testfilesBase, "input", "mixed", "Umsaetze.xlsx")
	volksbankFile := filepath.Join(testfilesBase, "input", "volksbank", "Umsaetze_DE12345678901234567890_2023.10.04.csv")

	type inputFile struct {
		set    int    // index of the set
		name   string // file name in the input directory of the set
		source string // fixture to copy
	}

	testCases := []struct {
		name             string
		sharedOutputDir  bool        // all sets use the output directory of set 0
		inputFiles       []inputFile // the last file never collides
		collidingOutputs []string    // output file names of the colliding files
	}{
		{
			name: "different extensions in one set",
			inputFiles: []inputFile{
				{0, "Umsaetze.csv", volksbankFile},
				{0, "Umsaetze.xlsx", barclaycardFile},
				{0, "Other.csv", volksbankFile},
			},
			collidingOutputs: []string{"Umsaetze.csv"},
		},
		{
			name:            "same name in sets sharing the output directory",
			sharedOutputDir: true,
			inputFiles: []inputFile{
				{0, "Umsaetze.csv", volksbankFile},
				{1, "Umsaetze.xlsx", barclaycardFile},
				{1, "Other.csv", volksbankFile},
			},
			collidingOutputs: []string{"Umsaetze.csv"},
		},
		{
			// The input files are in different directories so that they
			// can be created on a case-insensitive file system
			name:            "names differing only in case",
			sharedOutputDir: true,
			inputFiles: []inputFile{
				{0, "Umsaetze.csv", volksbankFile},
				{1, "umsaetze.CSV", volksbankFile},
				{1, "Other.csv", volksbankFile},
			},
			collidingOutputs: []string{"Umsaetze.csv", "umsaetze.csv"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			var sets []settings.BatchConvertSet
			for setNr := range 2 {
				inputDir := filepath.Join(tmpDir, fmt.Sprintf("input%d", setNr))
				outputDir := filepath.Join(tmpDir, fmt.Sprintf("output%d", setNr))
				if tc.sharedOutputDir {
					outputDir = filepath.Join(tmpDir, "output0")
				}
				for _, dir := range []string{inputDir, outputDir} {
					if err := os.MkdirAll(dir, 0o700); err != nil {
						t.Fatalf("Failed to create directory '%s'", dir)
					}
				}
				sets = append(sets, settings.BatchConvertSet{
					Name:      fmt.Sprintf("set %d", setNr),
					InputDir:  inputDir,
					OutputDir: outputDir,
				})
			}

			for _, f := range tc.inputFiles {
				if err := copyFile(f.source, filepath.Join(sets[f.set].InputDir, f.name)); err != nil {
					t.Fatalf("Failed to copy input file: %s", err)
				}
			}

			var firstCbStatus BatchStatus
			cb := func(s BatchStatus, _ interface{}) {
				if firstCbStatus == nil {
					firstCbStatus = make(BatchStatus, len(s))
					for i := range s {
						firstCbStatus[i] = BatchSetStatus{Name: s[i].Name, Files: slices.Clone(s[i].Files)}
					}
				}
			}

			status, err := BatchConvert(settings.BatchConvertSettings{Sets: sets}, time.Time{}, cb, nil)
			if !errors.Is(err, ErrOutputCollision) {
				t.Fatalf("Expected ErrOutputCollision, got '%v'", err)
			}

			collidingInputs := make([]string, 0, len(tc.inputFiles)-1)
			for _, f := range tc.inputFiles[:len(tc.inputFiles)-1] {
				collidingInputs = append(collidingInputs, filepath.Join(sets[f.set].InputDir, f.name))
			}
			for _, input := range collidingInputs {
				if !strings.Contains(err.Error(), input) {
					t.Errorf("Expected the error to name '%s', got '%s'", input, err)
				}
			}

			// The collisions are reported before any file is converted
			checkStatus := func(s BatchStatus, otherStatus ConversionStatus) {
				t.Helper()
				for _, set := range s {
					for _, f := range set.Files {
						if !slices.Contains(collidingInputs, f.InputFile) {
							if f.Status != otherStatus {
								t.Errorf("Expected status %d of '%s', got %d, error '%v'", otherStatus, f.InputFile, f.Status, f.Err)
							}
							continue
						}
						if f.Status != ConversionError || !errors.Is(f.Err, ErrOutputCollision) {
							t.Errorf("Expected '%s' to fail with ErrOutputCollision, got status %d, error '%v'", f.InputFile, f.Status, f.Err)
						}
						if f.OutputFile == "" {
							t.Errorf("Expected the output file of '%s' to be set", f.InputFile)
						}
					}
				}
			}
			checkStatus(firstCbStatus, NotStartedYet)
			checkStatus(status, ConversionSuccess)

			for _, set := range sets {
				for _, name := range tc.collidingOutputs {
					if _, err := os.Stat(filepath.Join(set.OutputDir, name)); err == nil {
						t.Errorf("Expected no output file '%s' in '%s'", name, set.OutputDir)
					}
				}
			}
		})
	}
}
