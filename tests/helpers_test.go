// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Shared test helpers for the cli_test package.
package cli_test

import (
	"bytes"
	"os/exec"
	"testing"
)

// newBinCmd constructs an exec.Cmd for the compiled kaos-control binary.
// The binary path is resolved by TestMain (in cli_init_test.go).
func newBinCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	return exec.Command(binPath, args...)
}

// runBin executes the kaos-control binary with the given args and returns
// stdout, stderr, and exit code. A non-zero exit code is not a test failure;
// callers assert it explicitly.
func runBin(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := newBinCmd(t, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			code = ex.ExitCode()
		} else {
			t.Fatalf("exec error (not ExitError): %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), code
}
