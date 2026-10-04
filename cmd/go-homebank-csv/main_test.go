package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	// Marks the subprocess which runs main() instead of the tests
	testMainEnv = "GO_HOMEBANK_CSV_TEST_MAIN"
	// Separates the arguments of that subprocess from the ones of the test binary
	testMainSeparator = "--"
)

// TestMain runs main() instead of the tests if testMainEnv is set.
//
// Checking the exit code of main() requires a subprocess, as os.Exit cannot be
// intercepted inside the test process.
func TestMain(m *testing.M) {
	if os.Getenv(testMainEnv) == "1" {
		os.Args = append([]string{"go-homebank-csv"}, testMainArgs(os.Args)...)
		main()
		// Only reached if main() did not exit on its own
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// testMainArgs returns the arguments following testMainSeparator
func testMainArgs(args []string) []string {
	for i, arg := range args {
		if arg == testMainSeparator {
			return args[i+1:]
		}
	}
	return nil
}

// runMain runs main() with the given arguments in a subprocess and returns its
// exit code together with its combined output.
func runMain(t *testing.T, args ...string) (int, string) {
	t.Helper()
	return runMainWithEnv(t, nil, args...)
}

// runMainWithEnv is like runMain, but adds env to the environment of the
// subprocess
func runMainWithEnv(t *testing.T, env []string, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{testMainSeparator}, args...)...)
	cmd.Env = append(append(os.Environ(), testMainEnv+"=1"), env...)
	output, err := cmd.CombinedOutput()
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode(), string(output)
	}
	if err != nil {
		t.Fatalf("Cannot run subprocess: %v", err)
	}
	return 0, string(output)
}

// runMainSeparateOutput is like runMain, but returns stdout and stderr of the
// subprocess separately
func runMainSeparateOutput(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{testMainSeparator}, args...)...)
	cmd.Env = append(os.Environ(), testMainEnv+"=1")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode(), stdout.String(), stderr.String()
	}
	if err != nil {
		t.Fatalf("Cannot run subprocess: %v", err)
	}
	return 0, stdout.String(), stderr.String()
}

// writeTempFile writes content into a new file inside the test's temp dir
func writeTempFile(t *testing.T, name string, content string) string {
	t.Helper()
	fpath := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(fpath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return fpath
}

// A file of an unknown format cannot be converted, so the exit code has to
// signal the failure
func TestMainConvertUnknownFormat(t *testing.T) {
	infile := writeTempFile(t, "unknown_format.csv", "not,a,known,bank,export\n")
	outfile := filepath.Join(filepath.Dir(infile), "output.csv")

	exitCode, output := runMain(t, "convert", infile, outfile)

	if exitCode == 0 {
		t.Errorf("Expected non zero exit code, got %d. Output: %s", exitCode, output)
	}
}

// A file which does not match the explicitly given format cannot be converted
// either
func TestMainConvertParserError(t *testing.T) {
	infile := writeTempFile(t, "no_dkb_file.csv", "not,a,known,bank,export\n")
	outfile := filepath.Join(filepath.Dir(infile), "output.csv")

	exitCode, output := runMain(t, "convert", "--format=DKB", infile, outfile)

	if exitCode == 0 {
		t.Errorf("Expected non zero exit code, got %d. Output: %s", exitCode, output)
	}
}

// A successful conversion has to keep the exit code at zero
func TestMainConvertSuccess(t *testing.T) {
	content := "wallet,currency,category,datetime,money,description\n" +
		"Wallet,EUR,Category,2024-01-02 10:00:00,-12.34,Description\n"
	infile := writeTempFile(t, "moneywallet.csv", content)
	outfile := filepath.Join(filepath.Dir(infile), "output.csv")

	exitCode, output := runMain(t, "convert", "--format=MoneyWallet", infile, outfile)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Output: %s", exitCode, output)
	}
	if _, err := os.Stat(outfile); err != nil {
		t.Errorf("Output file has not been written: %v", err)
	}
}

// Listing the formats succeeds without any further arguments
func TestMainListFormats(t *testing.T) {
	exitCode, output := runMain(t, "list-formats")

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Output: %s", exitCode, output)
	}
	if !strings.Contains(output, "MoneyWallet") {
		t.Errorf("Expected the format list to contain 'MoneyWallet', got: %s", output)
	}
}

// The version is shown without any further arguments
func TestMainVersion(t *testing.T) {
	exitCode, output := runMain(t, "--version")

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Output: %s", exitCode, output)
	}
	if !strings.HasPrefix(output, "go-homebank-csv ") {
		t.Errorf("Expected the output to start with 'go-homebank-csv ', got: %s", output)
	}
}

// writeBatchConvertConfig writes a config file with a single batchconvert set
// and returns the environment pointing the subprocess to it
func writeBatchConvertConfig(t *testing.T, inputDir string, outputDir string) []string {
	t.Helper()
	configHome := t.TempDir()
	configDir := filepath.Join(configHome, "go-homebank-csv")
	if err := os.Mkdir(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	config := "batchconvert:\n" +
		"  sets:\n" +
		"    - name: test\n" +
		"      inputdir: " + strconv.Quote(inputDir) + "\n" +
		"      outputdir: " + strconv.Quote(outputDir) + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	return []string{"XDG_CONFIG_HOME=" + configHome}
}

// A file which cannot be converted has to be reported with its cause and
// has to be signaled by the exit code
func TestMainBatchConvertFailedFile(t *testing.T) {
	infile := writeTempFile(t, "unknown_format.csv", "not,a,known,bank,export\n")
	env := writeBatchConvertConfig(t, filepath.Dir(infile), t.TempDir())

	exitCode, output := runMainWithEnv(t, env, "batch-convert")

	if exitCode == 0 {
		t.Errorf("Expected non zero exit code, got %d. Output: %s", exitCode, output)
	}
	if !strings.Contains(output, "Failed: "+infile+": cannot deduce format") {
		t.Errorf("Expected the failed file with its cause in the output, got: %s", output)
	}
}

// A successful batch conversion has to keep the exit code at zero
func TestMainBatchConvertSuccess(t *testing.T) {
	content := "wallet,currency,category,datetime,money,description\n" +
		"Wallet,EUR,Category,2024-01-02 10:00:00,-12.34,Description\n"
	infile := writeTempFile(t, "moneywallet.csv", content)
	outputDir := t.TempDir()
	env := writeBatchConvertConfig(t, filepath.Dir(infile), outputDir)

	exitCode, output := runMainWithEnv(t, env, "batch-convert")

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Output: %s", exitCode, output)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "moneywallet.csv")); err != nil {
		t.Errorf("Output file has not been written: %v", err)
	}
}

// Errors are written to stderr, the regular output stays on stdout
func TestMainErrorOnStderr(t *testing.T) {
	infile := writeTempFile(t, "unknown_format.csv", "not,a,known,bank,export\n")
	outfile := filepath.Join(filepath.Dir(infile), "output.csv")

	exitCode, stdout, stderr := runMainSeparateOutput(t, "convert", infile, outfile)

	if exitCode == 0 {
		t.Errorf("Expected non zero exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr, "cannot deduce format") {
		t.Errorf("Expected the error on stderr, got: %q", stderr)
	}
	if strings.Contains(stdout, "cannot deduce format") {
		t.Errorf("Expected no error on stdout, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Converting file") {
		t.Errorf("Expected the regular output on stdout, got: %q", stdout)
	}
}

// A file of a detected format is converted without an explicit format
func TestMainConvertDetectedFormat(t *testing.T) {
	content := "wallet,currency,category,datetime,money,description\n" +
		"Wallet,EUR,Category,2024-01-02 10:00:00,-12.34,Description\n"
	infile := writeTempFile(t, "moneywallet.csv", content)
	outfile := filepath.Join(filepath.Dir(infile), "output.csv")

	exitCode, output := runMain(t, "convert", infile, outfile)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Output: %s", exitCode, output)
	}
	if !strings.Contains(output, "Detected format 'MoneyWallet'") || !strings.Contains(output, "Found 1 entries") {
		t.Errorf("Expected the detected format and the number of entries, got: %s", output)
	}
	if _, err := os.Stat(outfile); err != nil {
		t.Errorf("Output file has not been written: %v", err)
	}
}
