// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"strings"
	"testing"
)

const vitestJSONPass = `{
  "numTotalTests": 2,
  "numPassedTests": 2,
  "numFailedTests": 0,
  "numPendingTests": 0,
  "testResults": [
    {
      "testFilePath": "/project/tests/web/foo.spec.ts",
      "assertionResults": [
        {"ancestorTitles": ["FooSuite"], "title": "passes", "status": "passed", "failureMessages": [], "duration": 5},
        {"ancestorTitles": ["FooSuite"], "title": "also passes", "status": "passed", "failureMessages": [], "duration": 3}
      ]
    }
  ]
}`

const vitestJSONFail = `{
  "numTotalTests": 3,
  "numPassedTests": 1,
  "numFailedTests": 2,
  "numPendingTests": 0,
  "testResults": [
    {
      "testFilePath": "/project/tests/web/bar.spec.ts",
      "assertionResults": [
        {
          "ancestorTitles": ["BarSuite", "nested"],
          "title": "should work",
          "status": "failed",
          "failureMessages": ["Expected 1 to equal 2"],
          "duration": 12.5,
          "location": {"line": 42, "column": 5}
        },
        {
          "ancestorTitles": ["BarSuite"],
          "title": "should also work",
          "status": "failed",
          "failureMessages": ["Expected true to be false", "Second message"],
          "duration": 8.0,
          "location": {"line": 67, "column": 3}
        },
        {
          "ancestorTitles": [],
          "title": "passes fine",
          "status": "passed",
          "failureMessages": [],
          "duration": 2.0
        }
      ]
    }
  ]
}`

const vitestJSONBadInput = `not json at all`

func TestParseVitestJSON_AllPass(t *testing.T) {
	r, err := ParseVitestJSON(strings.NewReader(vitestJSONPass))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Suite != "vitest" {
		t.Errorf("Suite = %q, want vitest", r.Suite)
	}
	if r.Total != 2 {
		t.Errorf("Total = %d, want 2", r.Total)
	}
	if r.Passed != 2 {
		t.Errorf("Passed = %d, want 2", r.Passed)
	}
	if r.Failed != 0 {
		t.Errorf("Failed = %d, want 0", r.Failed)
	}
	if len(r.Failures) != 0 {
		t.Errorf("len(Failures) = %d, want 0", len(r.Failures))
	}
}

func TestParseVitestJSON_Failures(t *testing.T) {
	r, err := ParseVitestJSON(strings.NewReader(vitestJSONFail))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Total != 3 {
		t.Errorf("Total = %d, want 3", r.Total)
	}
	if r.Failed != 2 {
		t.Errorf("Failed = %d, want 2", r.Failed)
	}
	if len(r.Failures) != 2 {
		t.Fatalf("len(Failures) = %d, want 2", len(r.Failures))
	}

	f0 := r.Failures[0]
	if f0.TestName != "BarSuite > nested > should work" {
		t.Errorf("Failures[0].TestName = %q, want %q", f0.TestName, "BarSuite > nested > should work")
	}
	if f0.File != "bar.spec.ts" {
		t.Errorf("Failures[0].File = %q, want bar.spec.ts", f0.File)
	}
	if f0.Line != 42 {
		t.Errorf("Failures[0].Line = %d, want 42", f0.Line)
	}
	if f0.ErrorMsg != "Expected 1 to equal 2" {
		t.Errorf("Failures[0].ErrorMsg = %q", f0.ErrorMsg)
	}

	f1 := r.Failures[1]
	if f1.TestName != "BarSuite > should also work" {
		t.Errorf("Failures[1].TestName = %q, want %q", f1.TestName, "BarSuite > should also work")
	}
	if f1.Line != 67 {
		t.Errorf("Failures[1].Line = %d, want 67", f1.Line)
	}
}

func TestParseVitestJSON_BadInput(t *testing.T) {
	r, err := ParseVitestJSON(strings.NewReader(vitestJSONBadInput))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.RawError == "" {
		t.Error("RawError should be set for non-JSON input")
	}
}
