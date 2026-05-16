// SPDX-License-Identifier: AGPL-3.0-or-later

// Package testrunner implements the test execution orchestrator: parsers for
// Go/Vitest/Playwright JSON output, artifact mapping, deduplication, and
// automatic defect filing.
package testrunner

// TestFailure is one normalised failure from any supported test framework.
type TestFailure struct {
	Suite    string  // "go", "vitest", or "playwright"
	Package  string  // Go package path or Vitest/Playwright file path
	TestName string  // fully qualified test name
	File     string  // source file path (basename or relative path)
	Line     int     // line number; 0 if unknown
	ErrorMsg string  // primary assertion/error text
	Output   string  // full test output for context
	Elapsed  float64 // seconds
}

// SuiteResult is the parsed output of one test suite run.
type SuiteResult struct {
	Suite    string        // "go", "vitest", or "playwright"
	Total    int
	Passed   int
	Failed   int
	Skipped  int
	Elapsed  float64 // wall-clock seconds
	Failures []TestFailure
	// RawError is non-empty when the suite failed to produce valid JSON output
	// (e.g. a compilation error caused the process to write plain text to stdout).
	RawError string
}
