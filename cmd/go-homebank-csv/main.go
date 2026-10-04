package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/sercxanto/go-homebank-csv/internal/pkg/batchconvert"
	"github.com/sercxanto/go-homebank-csv/internal/pkg/settings"
	"github.com/sercxanto/go-homebank-csv/pkg/parser"
)

type ConvertCmd struct {
	Format  *parser.SourceFormat `name:"format" help:"Format of input file, if not given it will be guessed. For a list of supported formats see the command 'list-formats'"`
	Infile  string               `arg:"" name:"infile" type:"existingfile" help:"Input file" type:"path"`
	Outfile string               `arg:"" name:"outfile" type:"path" help:"CSV file ready to import into homebank" type:"path"`
}

type ListFormatsCmd struct {
}

type BatchConvertCmd struct {
}

var CLI struct {
	Version      kong.VersionFlag `help:"Show version and exit"`
	Convert      ConvertCmd       `cmd:"" default:"withargs" help:"Convert CSV"`
	BatchConvert BatchConvertCmd  `cmd:"" help:"Batch convert CSV"`
	ListFormats  ListFormatsCmd   `cmd:"" help:"Lists supported formats"`
}

func (c *ConvertCmd) Run() error {
	var formatString string
	if c.Format == nil {
		formatString = "autodetect format"
	} else {
		formatString = fmt.Sprintf("format '%s'", *c.Format)
	}
	fmt.Printf("Converting file '%s' (%s) to file '%s'\n", c.Infile, formatString, c.Outfile)

	var p parser.Parser

	if c.Format == nil {
		p = parser.GetGuessedParser(c.Infile)
		if p == nil {
			return fmt.Errorf("cannot deduce format for file '%s'", c.Infile)
		}
		// GetGuessedParser returns the parser which has already parsed the
		// file successfully, so the file is not parsed a second time
		fmt.Printf("Detected format '%s'\n", p.GetFormat())
	} else {
		p = parser.GetParser(*c.Format)
		if err := p.ParseFile(c.Infile); err != nil {
			return err
		}
	}
	fmt.Printf("Found %d entries\n", p.GetNumberOfEntries())
	return p.ConvertToHomebank(c.Outfile)
}

func (c *BatchConvertCmd) Run() error {
	var s settings.Settings
	configFile, err := s.LoadFromDefaultFile()
	if err != nil {
		return err
	}
	fmt.Println("Loaded configuration from", configFile)
	if err := s.CheckValidity(); err != nil {
		return err
	}
	if len(s.BatchConvert.Sets) == 0 {
		return errors.New("no batchconvert sets defined in config file")
	}
	fmt.Println("Found", len(s.BatchConvert.Sets), "sets:")
	for _, set := range s.BatchConvert.Sets {
		fmt.Println(" ", set.Name, ":", set.InputDir)
	}

	// Remember last conversion state for each file to not show duplicate output
	fileStatus := make(map[string]batchconvert.ConversionStatus, 20)

	cb := func(status batchconvert.BatchStatus, _ interface{}) {
		for _, b := range status {
			for _, f := range b.Files {
				changed := false
				if _, ok := fileStatus[f.InputFile]; !ok {
					changed = true
				} else if fileStatus[f.InputFile] != f.Status {
					changed = true
				}
				fileStatus[f.InputFile] = f.Status
				if changed {
					switch f.Status {
					case batchconvert.ConversionInProgress:
						fmt.Println("  In Progress:", f.InputFile)
					case batchconvert.ConversionSuccess:
						fmt.Println("  Success:", f.InputFile)
					case batchconvert.ConversionError:
						fmt.Println("  Failed:", f.InputFile+":", f.Err)
					case batchconvert.Skipped:
						fmt.Println("  Skipped:", f.InputFile)
					}
				}
			}
		}
	}

	fmt.Println("BatchConvert starting ...")
	_, err = batchconvert.BatchConvert(s.BatchConvert, time.Now(), cb, nil)
	if err != nil {
		return fmt.Errorf("BatchConvert finished with errors:\n%w", err)
	}
	fmt.Println("BatchConvert finished")
	return nil
}

func (l *ListFormatsCmd) Run() error {
	for _, f := range parser.GetSourceFormats() {
		fmt.Println(f)
	}
	return nil
}

// version returns the module version embedded by the Go toolchain at build
// time, without the leading "v" of the git tag
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	return strings.TrimPrefix(info.Main.Version, "v")
}

func main() {
	ctx := kong.Parse(&CLI, kong.Vars{"version": "go-homebank-csv " + version()})
	err := ctx.Run()
	if err != nil {
		// Errors go to stderr, so that they are not mixed into redirected
		// regular output
		fmt.Fprintln(os.Stderr, err)
		// Exit with a non zero code so that callers, e.g. shell scripts or
		// cron jobs, can detect the failure
		os.Exit(1)
	}
}
