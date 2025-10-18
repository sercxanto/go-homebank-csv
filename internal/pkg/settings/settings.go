// Package setting implements config file settings for go-homebank-csv.
package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	return s.NormalizePaths()
}

func (settings *Settings) LoadFromString(str string) error {
	// Load settings from str
	// Parse yaml contained in str into variable settings
	*settings = Settings{}
	err := yaml.Unmarshal([]byte(str), settings)
	if err != nil {
		return err
	}
	return settings.NormalizePaths()
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
	return settings.NormalizePaths()
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

// NormalizePaths expands user facing shortcuts within all configured paths.
func (s *Settings) NormalizePaths() error {
	return s.BatchConvert.Sets.NormalizePaths()
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

// NormalizePaths expands supported directory shortcuts for all sets.
func (s BatchConvertSets) NormalizePaths() error {
	for i := range s {
		if err := s[i].NormalizePaths(); err != nil {
			return fmt.Errorf("batchconvert set %q: %w", s[i].Name, err)
		}
	}
	return nil
}

// NormalizePaths expands supported directory shortcuts (e.g. "~", "xdg:documents") for a set.
func (s *BatchConvertSet) NormalizePaths() error {
	expandedInput, err := expandPath(s.InputDir)
	if err != nil {
		return fmt.Errorf("inputdir: %w", err)
	}
	expandedOutput, err := expandPath(s.OutputDir)
	if err != nil {
		return fmt.Errorf("outputdir: %w", err)
	}
	s.InputDir = expandedInput
	s.OutputDir = expandedOutput
	return nil
}

func expandPath(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	path := raw
	if strings.HasPrefix(path, "~") {
		var err error
		path, err = expandHome(path)
		if err != nil {
			return "", err
		}
	}

	if strings.HasPrefix(strings.ToLower(path), "xdg:") {
		var err error
		path, err = expandXDG(path)
		if err != nil {
			return "", err
		}
	}

	return filepath.Clean(path), nil
}

func expandHome(path string) (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", fmt.Errorf("unsupported home shortcut '%s'", path)
	}
	trimmed := strings.TrimLeft(path[1:], "/\\")
	if trimmed == "" {
		return home, nil
	}
	return filepath.Join(home, trimmed), nil
}

func expandXDG(path string) (string, error) {
	lower := strings.ToLower(path)
	tokenWithRest := path[len("xdg:"):]
	lowerTokenWithRest := lower[len("xdg:"):]

	sepIndex := strings.IndexAny(lowerTokenWithRest, "/\\")
	var token string
	var remainder string
	if sepIndex == -1 {
		token = lowerTokenWithRest
	} else {
		token = lowerTokenWithRest[:sepIndex]
		remainder = tokenWithRest[sepIndex:]
	}

	base, err := xdgDirForToken(token)
	if err != nil {
		return "", err
	}
	if remainder == "" {
		return base, nil
	}
	remainder = strings.TrimLeft(remainder, "/\\")
	return filepath.Join(base, remainder), nil
}

func xdgDirForToken(token string) (string, error) {
	switch token {
	case "documents":
		if dir := xdg.UserDirs.Documents; dir != "" {
			return dir, nil
		}
		return "", fmt.Errorf("xdg documents directory not found")
	case "downloads":
		if dir := xdg.UserDirs.Download; dir != "" {
			return dir, nil
		}
		return "", fmt.Errorf("xdg downloads directory not found")
	default:
		return "", fmt.Errorf("unknown xdg shortcut '%s'", token)
	}
}

func userHomeDir() (string, error) {
	if home := xdg.Home; home != "" {
		return home, nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("cannot resolve home directory: %w", err)
	}
	return home, nil
}
