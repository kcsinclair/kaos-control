// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"encoding/json"
	"io"
)

// playwrightReport is the top-level Playwright JSON reporter structure.
type playwrightReport struct {
	Suites []*playwrightSuite `json:"suites"`
	Stats  *playwrightStats   `json:"stats"`
}

type playwrightStats struct {
	Total      int `json:"total"`
	Expected   int `json:"expected"`
	Unexpected int `json:"unexpected"`
	Skipped    int `json:"skipped"`
}

type playwrightSuite struct {
	Title  string             `json:"title"`
	File   string             `json:"file"`
	Suites []*playwrightSuite `json:"suites"`
	Specs  []*playwrightSpec  `json:"specs"`
}

type playwrightSpec struct {
	Title    string              `json:"title"`
	OK       bool                `json:"ok"`
	Tests    []*playwrightTest   `json:"tests"`
	Location *playwrightLocation `json:"location"`
}

type playwrightTest struct {
	Results []*playwrightResult `json:"results"`
	Status  string              `json:"status"` // "expected"|"unexpected"|"flaky"|"skipped"
}

type playwrightResult struct {
	Status   string           `json:"status"` // "passed"|"failed"|"timedOut"|"skipped"
	Error    *playwrightError `json:"error"`
	Duration float64          `json:"duration"` // milliseconds
}

type playwrightError struct {
	Message  string              `json:"message"`
	Location *playwrightLocation `json:"location"`
}

type playwrightLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// ParsePlaywrightJSON parses the Playwright JSON reporter output.
// If the input is not valid JSON, the returned SuiteResult has RawError set.
func ParsePlaywrightJSON(r io.Reader) (*SuiteResult, error) {
	result := &SuiteResult{Suite: "playwright"}

	var report playwrightReport
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		result.RawError = err.Error()
		return result, nil
	}

	if report.Stats != nil {
		result.Total = report.Stats.Total
		result.Passed = report.Stats.Expected
		result.Failed = report.Stats.Unexpected
		result.Skipped = report.Stats.Skipped
	}

	for _, suite := range report.Suites {
		extractPlaywrightFailures(suite, "", result)
	}

	return result, nil
}

// extractPlaywrightFailures walks nested Playwright suites recursively,
// appending failures to result.
func extractPlaywrightFailures(suite *playwrightSuite, titlePrefix string, result *SuiteResult) {
	prefix := titlePrefix
	if suite.Title != "" {
		if prefix != "" {
			prefix += " > " + suite.Title
		} else {
			prefix = suite.Title
		}
	}

	for _, spec := range suite.Specs {
		specTitle := prefix
		if spec.Title != "" {
			if specTitle != "" {
				specTitle += " > " + spec.Title
			} else {
				specTitle = spec.Title
			}
		}

		for _, test := range spec.Tests {
			for _, res := range test.Results {
				if res.Status != "failed" && res.Status != "timedOut" {
					continue
				}

				f := TestFailure{
					Suite:    "playwright",
					TestName: specTitle,
					Elapsed:  res.Duration / 1000.0,
				}

				// Prefer location from spec; override with error location if present.
				if spec.Location != nil {
					f.File = spec.Location.File
					f.Line = spec.Location.Line
				}
				if res.Error != nil {
					f.ErrorMsg = res.Error.Message
					f.Output = res.Error.Message
					if res.Error.Location != nil {
						f.File = res.Error.Location.File
						f.Line = res.Error.Location.Line
					}
				}

				result.Elapsed += f.Elapsed
				result.Failures = append(result.Failures, f)
			}
		}
	}

	// Recurse into nested suites.
	for _, sub := range suite.Suites {
		extractPlaywrightFailures(sub, prefix, result)
	}
}
