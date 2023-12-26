package batchconvert

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
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
	if dstFile, err := os.Create(dst); err != nil {
		return err
	} else {
		defer dstFile.Close()
		_, err = io.Copy(dstFile, srcFile)
		return err
	}
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

	if status, err = BatchConvert(settings1, time.Now(), cb, cbUserData); err != nil {
		t.Fatalf("BatchConvert should not return error")
	}

	if !reflect.DeepEqual(status, cbStatus) {
		t.Fatalf("status and cbStatus are not equal")
	}

	cbUpdateNr = 0
	if status, err = BatchConvert(settings2, time.Now(), cb, cbUserData); err != nil {
		t.Fatalf("BatchConvert should return error")
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
