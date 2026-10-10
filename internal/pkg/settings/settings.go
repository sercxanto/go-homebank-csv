// Package settings implements config file settings for go-homebank-csv.
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

// unmarshal parses the YAML in data into v.
//
// Unknown keys are rejected, so that a misspelled key, e.g. "inputDir"
// instead of "inputdir", is reported instead of being silently ignored.
// Duplicate keys are rejected by the yaml package by default.
func unmarshal(data []byte, v any) error {
	return yaml.UnmarshalWithOptions(data, v, yaml.Strict())
}

func (s *BatchConvertSet) LoadFromString(str string) error {
	// Reset s to default values as yaml unmarshal does only write to
	// fields present in yaml string
	*s = BatchConvertSet{}

	err := unmarshal([]byte(str), s)
	if err != nil {
		return err
	}
	return s.NormalizePaths()
}

func (s *Settings) LoadFromString(str string) error {
	// Load settings from str
	// Parse yaml contained in str into variable s
	*s = Settings{}
	err := unmarshal([]byte(str), s)
	if err != nil {
		return err
	}
	return s.NormalizePaths()
}

func (s *Settings) LoadFromFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	*s = Settings{}
	err = unmarshal(content, s)
	if err != nil {
		// The error names line and column only, so add the file
		return fmt.Errorf("config file '%s': %w", filePath, err)
	}
	return s.NormalizePaths()
}

// LoadFromDefaultFile loads settings from default config file.
func (s *Settings) LoadFromDefaultFile() (string, error) {
	configFilePath, err := xdg.SearchConfigFile(defaultConfigFilePath)
	if err != nil {
		return "", err
	}
	return configFilePath, s.LoadFromFile(configFilePath)
}

// CheckValidity reports whether the whole settings are valid
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
//
// The errors name the keys of the config file, not the fields of the struct.
func (s BatchConvertSet) CheckValidity() error {
	if s.Name == "" {
		return errors.New("name is empty")
	}
	if s.InputDir == "" {
		return errors.New("inputdir is empty")
	}
	if s.OutputDir == "" {
		return errors.New("outputdir is empty")
	}
	if s.InputDir == s.OutputDir {
		return fmt.Errorf("inputdir and outputdir are the same directory '%s'", s.InputDir)
	}
	if s.FileMaxAgeDays < 0 {
		return fmt.Errorf("filemaxagedays must not be negative, got %d", s.FileMaxAgeDays)
	}
	if !IsFileGlobPatternValid(s.FileGlobPattern) {
		return fmt.Errorf("fileglobpattern '%s' is invalid", s.FileGlobPattern)
	}
	return nil
}

// CheckValidity reports whether a BatchConvertSets are valid
//
// Possible errors:
//
//   - invalid CheckValidity() of entry, prefixed with the name of the entry
//     or, if the name is empty, its 1 based position
//   - duplicate Name
//   - duplicate InputDir / FileGlobPattern combination
//   - OutputDir of one entry is the InputDir of another one
func (s BatchConvertSets) CheckValidity() error {

	// inputFiles identifies the input files of a set
	type inputFiles struct {
		inputDir        string
		fileGlobPattern string
	}

	names := make(map[string]bool, len(s))
	inputs := make(map[inputFiles]bool, len(s))

	for i, entry := range s {
		if err := entry.CheckValidity(); err != nil {
			if entry.Name == "" {
				return fmt.Errorf("set %d: %w", i+1, err)
			}
			return fmt.Errorf("set '%s': %w", entry.Name, err)
		}
		if names[entry.Name] {
			return fmt.Errorf("duplicate name '%s' detected", entry.Name)
		}
		names[entry.Name] = true

		// An empty pattern matches all files, like "*"
		pattern := entry.FileGlobPattern
		if pattern == "" {
			pattern = "*"
		}
		input := inputFiles{entry.InputDir, pattern}
		if inputs[input] {
			return fmt.Errorf("duplicate inputdir / fileglobpattern combination detected ('%s', '%s')",
				entry.InputDir, entry.FileGlobPattern)
		}
		inputs[input] = true
	}

	// The converted files of one set would be input files of the other one.
	// They are no supported source format, so their conversion would fail on
	// every run.
	inputDirs := make(map[string]string, len(s))
	for _, entry := range s {
		if _, ok := inputDirs[entry.InputDir]; !ok {
			inputDirs[entry.InputDir] = entry.Name
		}
	}
	for _, entry := range s {
		if name, ok := inputDirs[entry.OutputDir]; ok {
			return fmt.Errorf("outputdir of '%s' is the inputdir of '%s' ('%s')",
				entry.Name, name, entry.OutputDir)
		}
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
	case "desktop":
		if dir := xdg.UserDirs.Desktop; dir != "" {
			return dir, nil
		}
		return "", fmt.Errorf("xdg desktop directory not found")
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
