// Package setting implements config file settings for go-homebank-csv.
package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/goccy/go-yaml"
	"github.com/sercxanto/go-homebank-csv/pkg/parser"
)

const defaultConfigFilePath = "go-homebank-csv/config.yml"

type BatchConvertSet struct {
	// Name of the batchconvert set, must be unique
	Name string `yaml:"name"`
	// Where to search for input files, must be non-empty
	InputDir string `yaml:"inputdir"`
	// Where to place output files, must be non-empty and not equal to InputDir
	OutputDir string `yaml:"outputdir"`
	// Source format, nil to use format autodetect
	Format *parser.SourceFormat `yaml:"format"`
	// Glob pattern to search for input files
	FileGlobPattern string `yaml:"fileglobpattern"`
	// Maximum age of input files in days
	FileMaxAgeDays int `yaml:"filemaxagedays"`
}

type BatchConvertSets []BatchConvertSet

type BatchConvertSettings struct {
	Sets BatchConvertSets `yaml:"sets"`
}

type Settings struct {
	BatchConvert BatchConvertSettings `yaml:"batchconvert"`
}

func (s *BatchConvertSet) LoadFromString(str string) error {
	// Reset s to default values as yaml unmarshal does only write to
	// fields present in yaml string
	*s = BatchConvertSet{}

	err := yaml.Unmarshal([]byte(str), s)
	if err != nil {
		return err
	}
	return nil
}

func (settings *Settings) LoadFromString(str string) error {
	// Load settings from str
	// Parse yaml contained in str into variable settings
	*settings = Settings{}
	err := yaml.Unmarshal([]byte(str), settings)
	if err != nil {
		return err
	}
	return nil
}

func (settings *Settings) LoadFromFile(filePath string) error {

	// Open file filePath for reading
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read file into a byte slice
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	*settings = Settings{}
	err = yaml.Unmarshal(content, settings)
	if err != nil {
		return err
	}
	return nil
}

// LoadFromDefaultFile loads settings from default config file.
func (settings *Settings) LoadFromDefaultFile() (string, error) {
	configFilePath, err := xdg.SearchConfigFile(defaultConfigFilePath)
	if err != nil {
		return "", err
	}
	return configFilePath, settings.LoadFromFile(configFilePath)
}

// CheckValidity reports whether a the whole settings are valid
func (s Settings) CheckValidity() error {
	if len(s.BatchConvert.Sets) > 0 {
		return s.BatchConvert.Sets.CheckValidity()
	}
	return nil
}

// IsFileGlobPatternValid reports whether a file glob pattern is valid.
//
//   - pattern: the file glob pattern to be validated.
//   - bool: returns true if the pattern is valid, false otherwise.
func IsFileGlobPatternValid(pattern string) bool {
	_, err := filepath.Match(pattern, "")
	return err == nil
}

// CheckValidity reports whether a BatchConvertSet is valid
//
// Possible errors:
//
//   - Name is empty
//   - InputDir is empty
//   - OutputDir is empty
//   - OutputDir == InputDir
//   - FileMaxAgeDays < 0
//   - FileGlobPattern is invalid
func (s BatchConvertSet) CheckValidity() error {
	if s.Name == "" {
		return errors.New("name is empty")
	}
	if s.InputDir == "" {
		return errors.New("InputDir is empty")
	}
	if s.OutputDir == "" {
		return errors.New("OutputDir is empty")
	}
	if s.InputDir == s.OutputDir {
		return errors.New("InputDir == OutputDir")
	}
	if s.FileMaxAgeDays < 0 {
		return errors.New("FileMaxAgeDays < 0")
	}
	if !IsFileGlobPatternValid(s.FileGlobPattern) {
		return errors.New("FileGlobPattern is invalid")
	}
	return nil
}

// CheckValidity reports whether a BatchConvertSets are valid
//
// Possible errors:
//
//   - invalid CheckValidity() of entry
//   - duplicate Name
//   - duplicate InputDir / FileGlobPattern combination
func (s BatchConvertSets) CheckValidity() error {

	names := make([]string, 0, len(s))
	inputDirAndGlobPattern := make([]string, 0, len(s))

	for _, entry := range s {
		if err := entry.CheckValidity(); err != nil {
			return err
		}
		for i := range names {
			if names[i] == entry.Name {
				return fmt.Errorf("duplicate Name '%s' detected", entry.Name)
			}
		}
		names = append(names, entry.Name)

		value := entry.InputDir + entry.FileGlobPattern
		for i := range inputDirAndGlobPattern {
			if inputDirAndGlobPattern[i] == value {
				return fmt.Errorf("duplicate InputDir / FileGlobPattern combination detected ('%s', '%s')",
					entry.InputDir, entry.FileGlobPattern)
			}
		}
		inputDirAndGlobPattern = append(inputDirAndGlobPattern, value)
	}

	return nil
}
