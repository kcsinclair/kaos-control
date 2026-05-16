// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"strings"
	"testing"
)

const playwrightJSONPass = `{
  "suites": [
    {
      "title": "login.spec.ts",
      "file": "login.spec.ts",
      "suites": [],
      "specs": [
        {
          "title": "logs in",
          "ok": true,
          "tests": [{"results": [{"status": "passed", "duration": 1234}]}],
          "location": {"file": "login.spec.ts", "line": 5, "column": 0}
        }
      ]
    }
  ],
  "stats": {"total": 1, "expected": 1, "unexpected": 0, "skipped": 0}
}`

const playwrightJSONFail = `{
  "suites": [
    {
      "title": "auth.spec.ts",
      "file": "auth.spec.ts",
      "suites": [
        {
          "title": "login flow",
          "suites": [],
          "specs": [
            {
              "title": "should redirect",
              "ok": false,
              "tests": [
                {
                  "results": [
                    {
                      "status": "failed",
                      "duration": 3000,
                      "error": {
                        "message": "Expected page to navigate",
                        "location": {"file": "auth.spec.ts", "line": 42, "column": 5}
                      }
                    }
                  ]
                }
              ],
              "location": {"file": "auth.spec.ts", "line": 38, "column": 0}
            }
          ]
        }
      ],
      "specs": [
        {
          "title": "top-level spec",
          "ok": false,
          "tests": [
            {
              "results": [
                {
                  "status": "failed",
                  "duration": 500,
                  "error": {
                    "message": "Element not found",
                    "location": {"file": "auth.spec.ts", "line": 10, "column": 2}
                  }
                }
              ]
            }
          ],
          "location": {"file": "auth.spec.ts", "line": 8, "column": 0}
        }
      ]
    }
  ],
  "stats": {"total": 2, "expected": 0, "unexpected": 2, "skipped": 0}
}`

const playwrightJSONBadInput = `{invalid json`

func TestParsePlaywrightJSON_AllPass(t *testing.T) {
	r, err := ParsePlaywrightJSON(strings.NewReader(playwrightJSONPass))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Suite != "playwright" {
		t.Errorf("Suite = %q, want playwright", r.Suite)
	}
	if r.Total != 1 {
		t.Errorf("Total = %d, want 1", r.Total)
	}
	if r.Passed != 1 {
		t.Errorf("Passed = %d, want 1", r.Passed)
	}
	if r.Failed != 0 {
		t.Errorf("Failed = %d, want 0", r.Failed)
	}
	if len(r.Failures) != 0 {
		t.Errorf("len(Failures) = %d, want 0", len(r.Failures))
	}
}

func TestParsePlaywrightJSON_Failures(t *testing.T) {
	r, err := ParsePlaywrightJSON(strings.NewReader(playwrightJSONFail))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Total != 2 {
		t.Errorf("Total = %d, want 2", r.Total)
	}
	if r.Failed != 2 {
		t.Errorf("Failed = %d, want 2", r.Failed)
	}
	if len(r.Failures) != 2 {
		t.Fatalf("len(Failures) = %d, want 2", len(r.Failures))
	}

	// Top-level spec (from specs array of the top-level suite)
	top := r.Failures[0]
	if top.TestName != "auth.spec.ts > top-level spec" {
		t.Errorf("Failures[0].TestName = %q", top.TestName)
	}
	if top.Line != 10 {
		t.Errorf("Failures[0].Line = %d, want 10", top.Line)
	}
	if top.ErrorMsg != "Element not found" {
		t.Errorf("Failures[0].ErrorMsg = %q", top.ErrorMsg)
	}

	// Nested spec inside a sub-suite
	nested := r.Failures[1]
	if nested.TestName != "auth.spec.ts > login flow > should redirect" {
		t.Errorf("Failures[1].TestName = %q", nested.TestName)
	}
	if nested.Line != 42 {
		t.Errorf("Failures[1].Line = %d, want 42", nested.Line)
	}
}

func TestParsePlaywrightJSON_BadInput(t *testing.T) {
	r, err := ParsePlaywrightJSON(strings.NewReader(playwrightJSONBadInput))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.RawError == "" {
		t.Error("RawError should be set for non-JSON input")
	}
}

func TestParsePlaywrightJSON_Skipped(t *testing.T) {
	input := `{
  "suites": [],
  "stats": {"total": 5, "expected": 3, "unexpected": 1, "skipped": 1}
}`
	r, err := ParsePlaywrightJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", r.Skipped)
	}
}
