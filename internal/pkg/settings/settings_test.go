package settings

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"

	"github.com/sercxanto/go-homebank-csv/pkg/parser"
)

func TestIsFileGlobPatternValid(t *testing.T) {

	patterns := map[string]bool{
		"*":   true,
		"*.*": true,
		"":    true,
		"[":   false,
	}

	for pattern, expected := range patterns {
		if IsFileGlobPatternValid(pattern) != expected {
			t.Errorf("Expected '%t' for '%s'", expected, pattern)
		}
	}
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

func TestLoadFromDefaultFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	xdg.Reload()

	var s Settings
	_, err := s.LoadFromDefaultFile()
	if err == nil {
		t.Error("Expected error for non existing config file")
	}

	srcConfigFilePath := filepath.Join("testfiles", "config.yml")
	configFilePath := filepath.Join(tmpDir, filepath.FromSlash(defaultConfigFilePath))
	configFileDir := filepath.Dir(configFilePath)
	if err := os.MkdirAll(configFileDir, os.ModeDir|0o700); err != nil {
		t.Fatalf("Failed to create directory '%s'", configFileDir)
	}
	// Copy file srcConfigFilePath to configFilePath
	if err := copyFile(srcConfigFilePath, configFilePath); err != nil {
		t.Fatalf("Failed to copy file '%s' to '%s'", srcConfigFilePath, configFilePath)
	}
	path, err := s.LoadFromDefaultFile()
	if err != nil {
		t.Fatalf("LoadFromDefaultFile return error '%s'", err)
	}
	if path != configFilePath {
		t.Errorf("Expected '%s' got '%s'", configFilePath, path)
	}
}

func TestBatchConvertSetCheckValidity(t *testing.T) {
	var s BatchConvertSet
	if s.CheckValidity() == nil {
		t.Error("Expected empty name error")
	}

	s.Name = "My name"
	if s.CheckValidity() == nil {
		t.Error("Expected empty InputDir error")
	}

	s.InputDir = "/some/path"

	if s.CheckValidity() == nil {
		t.Error("Expected empty OutputDir error")
	}

	s.OutputDir = "/some/path"
	if s.CheckValidity() == nil {
		t.Error("Expected same InputDir/OutputDir error")
	}

	s.OutputDir = "/some/other/path"
	s.FileMaxAgeDays = -1
	if s.CheckValidity() == nil {
		t.Error("Expected FileMaxAgeDays error")
	}

	s.FileMaxAgeDays = 0
	s.FileGlobPattern = "["
	if err := s.CheckValidity(); err == nil {
		t.Error("Expected FileGlobPattern error")
	}

	s.FileGlobPattern = "*"
	if err := s.CheckValidity(); err != nil {
		t.Errorf("No error expected, got '%s' instead", err)
	}
}

func TestBatchConvertSetsCheckValidity(t *testing.T) {

	s := BatchConvertSets{
		BatchConvertSet{
			Name:      "name1",
			InputDir:  "/my/path1",
			OutputDir: "/my/path2",
		},
		BatchConvertSet{
			Name:      "name2",
			InputDir:  "/my/path3",
			OutputDir: "/my/path4",
		},
	}

	if s.CheckValidity() != nil {
		t.Error("Expected nil error")
	}

	s = BatchConvertSets{
		BatchConvertSet{
			Name:      "name1",
			InputDir:  "/my/path1",
			OutputDir: "/my/path2",
		},
		BatchConvertSet{
			Name:      "name1",
			InputDir:  "/my/path3",
			OutputDir: "/my/path4",
		},
	}

	if s.CheckValidity() == nil {
		t.Error("Expected duplicate name error")
	}

	s = BatchConvertSets{
		BatchConvertSet{
			Name:      "name1",
			InputDir:  "/my/path1",
			OutputDir: "/my/path2",
		},
		BatchConvertSet{
			Name:      "name2",
			InputDir:  "/my/path1",
			OutputDir: "/my/path4",
		},
	}

	if s.CheckValidity() == nil {
		t.Error("Expected duplicate InputDir/FileGlobPattern error")
	}

	s = BatchConvertSets{
		BatchConvertSet{
			Name:      "name1",
			InputDir:  "/my/path1",
			OutputDir: "/my/path2",
		},
		BatchConvertSet{
			Name:            "name2",
			InputDir:        "/my/path1",
			OutputDir:       "/my/path4",
			FileGlobPattern: "some glob pattern",
		},
	}

	if s.CheckValidity() != nil {
		t.Error("Did not expect error")
	}

}

func TestSettingsCheckValidity(t *testing.T) {
	var s Settings
	if s.CheckValidity() != nil {
		t.Error("Expected nil error")
	}
}

func TestBatchConvertLoadFromString(t *testing.T) {

	var s BatchConvertSet

	err := s.LoadFromString("invalid yaml")
	if err == nil {
		t.Error("Expected error")
	}

	text1 := `
name: my name
inputdir: /my/path
outputdir: /my/path2
filemaxagedays: 10`

	err = s.LoadFromString(text1)
	if err != nil {
		t.Errorf("Expected nil error, got '%s' instead", err)
	}

	if s.Name != "my name" {
		t.Errorf("Expected 'my name', got '%s' instead", s.Name)
	}
	if s.InputDir != "/my/path" {
		t.Errorf("Expected '/my/path', got '%s' instead", s.InputDir)
	}
	if s.OutputDir != "/my/path2" {
		t.Errorf("Expected '/my/path2', got '%s' instead", s.OutputDir)
	}
	if s.Format != nil {
		t.Errorf("Expected nil, got '%s' instead", s.Format)
	}
	if s.FileGlobPattern != "" {
		t.Errorf("Expected '', got '%s' instead", s.FileGlobPattern)
	}
	if s.FileMaxAgeDays != 10 {
		t.Errorf("Expected '10', got '%d' instead", s.FileMaxAgeDays)
	}

	text2 := `
name: my name2
inputdir: /my/path
outputdir: /my/path2
format: Barclaycard
fileglobpattern: some glob pattern`

	err = s.LoadFromString(text2)
	if err != nil {
		t.Errorf("Expected nil error, got '%s' instead", err)
	}
	if s.Name != "my name2" {
		t.Errorf("Expected 'my name', got '%s' instead", s.Name)
	}
	if s.InputDir != "/my/path" {
		t.Errorf("Expected '/my/path', got '%s' instead", s.InputDir)
	}
	if s.OutputDir != "/my/path2" {
		t.Errorf("Expected '/my/path2', got '%s' instead", s.OutputDir)
	}
	if *(s.Format) != parser.Barclaycard {
		t.Errorf("Expected 'Barclaycard', got '%s' instead", (*s.Format))
	}
	if s.FileGlobPattern != "some glob pattern" {
		t.Errorf("Expected 'some glob pattern', got '%s' instead", s.FileGlobPattern)
	}
	if s.FileMaxAgeDays != 0 {
		t.Errorf("Expected '0', got '%d' instead", s.FileMaxAgeDays)
	}

	err = s.LoadFromString(text1)
	if err != nil {
		t.Errorf("Expected nil error, got '%s' instead", err)
	}
	if s.Format != nil {
		t.Errorf("Expected 'nil', got '%s' instead", *(s.Format))
	}
	if s.FileGlobPattern != "" {
		t.Errorf("Expected '', got '%s' instead", s.FileGlobPattern)
	}
}

func TestSettingsLoadFromString(t *testing.T) {
	var s Settings

	err := s.LoadFromString("invalid yaml")
	if err == nil {
		t.Error("Expected error")
	}

	text := `
batchconvert:
  sets:
  - name: name1
    inputdir: /my/path11
    outputdir: /my/path12
  - name: name2
    inputdir: /my/path21
    outputdir: /my/path22
    format: Barclaycard
    fileglobpattern: "*.*"`

	err = s.LoadFromString(text)
	if err != nil {
		t.Fatalf("Expected nil error, got '%s' instead", err)
	}
	if len(s.BatchConvert.Sets) != 2 {
		t.Fatalf("Expected 2 batchconvert sets, got '%d' instead", len(s.BatchConvert.Sets))
	}
	if s.BatchConvert.Sets[0].Name != "name1" {
		t.Errorf("Expected 'name1', got '%s' instead", s.BatchConvert.Sets[0].Name)
	}
	if s.BatchConvert.Sets[0].InputDir != "/my/path11" {
		t.Errorf("Expected '/my/path11', got '%s' instead", s.BatchConvert.Sets[0].InputDir)
	}
	if s.BatchConvert.Sets[0].OutputDir != "/my/path12" {
		t.Errorf("Expected '/my/path12', got '%s' instead", s.BatchConvert.Sets[0].OutputDir)
	}
	if s.BatchConvert.Sets[0].Format != nil {
		t.Errorf("Expected 'nil', got '%s' instead", *(s.BatchConvert.Sets[0].Format))
	}
	if s.BatchConvert.Sets[0].FileGlobPattern != "" {
		t.Errorf("Expected '', got '%s' instead", s.BatchConvert.Sets[0].FileGlobPattern)
	}
	if s.BatchConvert.Sets[0].FileMaxAgeDays != 0 {
		t.Errorf("Expected '0', got '%d' instead", s.BatchConvert.Sets[0].FileMaxAgeDays)
	}
	if s.BatchConvert.Sets[1].Name != "name2" {
		t.Errorf("Expected 'name2', got '%s' instead", s.BatchConvert.Sets[1].Name)
	}
	if s.BatchConvert.Sets[1].InputDir != "/my/path21" {
		t.Errorf("Expected '/my/path21', got '%s' instead", s.BatchConvert.Sets[1].InputDir)
	}
	if s.BatchConvert.Sets[1].OutputDir != "/my/path22" {
		t.Errorf("Expected '/my/path22', got '%s' instead", s.BatchConvert.Sets[1].OutputDir)
	}
	if s.BatchConvert.Sets[1].Format == nil {
		t.Errorf("Expected 'non-nil', got '%s' instead", *(s.BatchConvert.Sets[1].Format))
	}
	if *(s.BatchConvert.Sets[1].Format) != parser.Barclaycard {
		t.Errorf("Expected 'Barclaycard', got '%s' instead", *(s.BatchConvert.Sets[1].Format))
	}
	if s.BatchConvert.Sets[1].FileGlobPattern != "*.*" {
		t.Errorf("Expected '*.*', got '%s' instead", s.BatchConvert.Sets[1].FileGlobPattern)
	}
	if s.BatchConvert.Sets[1].FileMaxAgeDays != 0 {
		t.Errorf("Expected '0', got '%d' instead", s.BatchConvert.Sets[1].FileMaxAgeDays)
	}
}

func TestSettingsLoadFromFile(t *testing.T) {
	var s Settings

	fpath := filepath.Join("testfiles", "non_existing_config.yml")
	err := s.LoadFromFile(fpath)
	if err == nil {
		t.Errorf("Expected error for file not existing")
	}

	fpath = filepath.Join("testfiles", "invalid_yaml.yml")
	err = s.LoadFromFile(fpath)
	if err == nil {
		t.Errorf("Expected error for invalid yaml")
	}

	fpath = filepath.Join("testfiles", "config.yml")

	err = s.LoadFromFile(fpath)
	if err != nil {
		t.Fatalf("LoadFromFile return error '%s'", err)
	}
	if len(s.BatchConvert.Sets) != 2 {
		t.Fatalf("Expected 2 batchconvert sets, got '%d' instead", len(s.BatchConvert.Sets))
	}
	if s.BatchConvert.Sets[0].Name != "name1" {
		t.Errorf("Expected 'name1', got '%s' instead", s.BatchConvert.Sets[0].Name)
	}
	if s.BatchConvert.Sets[0].InputDir != "/my/path11" {
		t.Errorf("Expected '/my/path11', got '%s' instead", s.BatchConvert.Sets[0].InputDir)
	}
	if s.BatchConvert.Sets[0].OutputDir != "/my/path12" {
		t.Errorf("Expected '/my/path12', got '%s' instead", s.BatchConvert.Sets[0].OutputDir)
	}
	if s.BatchConvert.Sets[0].Format == nil {
		t.Fatalf("Expected 'non-nil', got 'non-nil' instead")
	}
	if *(s.BatchConvert.Sets[0].Format) != parser.Barclaycard {
		t.Errorf("Expected 'Barclaycard', got '%s' instead", *(s.BatchConvert.Sets[0].Format))
	}
	if s.BatchConvert.Sets[0].FileGlobPattern != "*.csv" {
		t.Errorf("Expected '*.csv', got '%s' instead", s.BatchConvert.Sets[0].FileGlobPattern)
	}
	if s.BatchConvert.Sets[0].FileMaxAgeDays != 5 {
		t.Errorf("Expected '5', got '%d' instead", s.BatchConvert.Sets[0].FileMaxAgeDays)
	}
	if s.BatchConvert.Sets[1].Name != "name2" {
		t.Errorf("Expected 'name2', got '%s' instead", s.BatchConvert.Sets[1].Name)
	}
	if s.BatchConvert.Sets[1].InputDir != "/my/path21" {
		t.Errorf("Expected '/my/path21', got '%s' instead", s.BatchConvert.Sets[1].InputDir)
	}
	if s.BatchConvert.Sets[1].OutputDir != "/my/path22" {
		t.Errorf("Expected '/my/path22', got '%s' instead", s.BatchConvert.Sets[1].OutputDir)
	}
	if s.BatchConvert.Sets[1].Format != nil {
		t.Fatal("Expected 'nil', got 'non-nil' instead")
	}
	if s.BatchConvert.Sets[1].FileGlobPattern != "" {
		t.Errorf("Expected '', got '%s' instead", s.BatchConvert.Sets[0].FileGlobPattern)
	}
	if s.BatchConvert.Sets[1].FileMaxAgeDays != 0 {
		t.Errorf("Expected '0', got '%d' instead", s.BatchConvert.Sets[0].FileMaxAgeDays)
	}

	err = s.CheckValidity()
	if err != nil {
		t.Errorf("Expected no error, got '%s' instead", err)
	}

}
