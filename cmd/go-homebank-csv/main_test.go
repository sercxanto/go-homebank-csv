package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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
	cmd := exec.Command(os.Args[0], append([]string{testMainSeparator}, args...)...)
	cmd.Env = append(os.Environ(), testMainEnv+"=1")
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
