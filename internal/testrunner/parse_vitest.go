// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
)

// vitestReport is the top-level structure of the Vitest JSON reporter output.
type vitestReport struct {
	NumTotalTests   int                `json:"numTotalTests"`
	NumPassedTests  int                `json:"numPassedTests"`
	NumFailedTests  int                `json:"numFailedTests"`
	NumPendingTests int                `json:"numPendingTests"`
	TestResults     []vitestFileResult `json:"testResults"`
}

type vitestFileResult struct {
	TestFilePath     string               `json:"testFilePath"`
	AssertionResults []vitestAssertResult `json:"assertionResults"`
}

type vitestAssertResult struct {
	AncestorTitles  []string        `json:"ancestorTitles"`
	Title           string          `json:"title"`
	Status          string          `json:"status"` // "passed", "failed", "pending"
	FailureMessages []string        `json:"failureMessages"`
	Duration        float64         `json:"duration"` // milliseconds
	Location        *vitestLocation `json:"location,omitempty"`
}

type vitestLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// ParseVitestJSON parses the Vitest JSON reporter format.
// If the input is not valid JSON, the returned SuiteResult has RawError set.
func ParseVitestJSON(r io.Reader) (*SuiteResult, error) {
	result := &SuiteResult{Suite: "vitest"}

	var report vitestReport
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		result.RawError = err.Error()
		return result, nil
	}

	result.Total = report.NumTotalTests
	result.Passed = report.NumPassedTests
	result.Failed = report.NumFailedTests
	result.Skipped = report.NumPendingTests

	for _, fileResult := range report.TestResults {
		for _, ar := range fileResult.AssertionResults {
			if ar.Status != "failed" {
				continue
			}

			// Build fully-qualified test name from ancestor titles + title.
			parts := append(append([]string(nil), ar.AncestorTitles...), ar.Title)
			testName := strings.Join(parts, " > ")

			var lineN int
			if ar.Location != nil {
				lineN = ar.Location.Line
			}

			var errMsg string
			if len(ar.FailureMessages) > 0 {
				errMsg = ar.FailureMessages[0]
			}

			elapsedSec := ar.Duration / 1000.0
			result.Elapsed += elapsedSec

			result.Failures = append(result.Failures, TestFailure{
				Suite:    "vitest",
				Package:  fileResult.TestFilePath,
				TestName: testName,
				File:     filepath.Base(fileResult.TestFilePath),
				Line:     lineN,
				ErrorMsg: errMsg,
				Output:   strings.Join(ar.FailureMessages, "\n"),
				Elapsed:  elapsedSec,
			})
		}
	}

	return result, nil
}
