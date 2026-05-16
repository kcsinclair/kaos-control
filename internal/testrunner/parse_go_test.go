// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"strings"
	"testing"
)

const goJSONPass = `{"Action":"run","Package":"example.com/foo","Test":"TestAlpha"}
{"Action":"output","Package":"example.com/foo","Test":"TestAlpha","Output":"=== RUN   TestAlpha\n"}
{"Action":"pass","Package":"example.com/foo","Test":"TestAlpha","Elapsed":0.001}
{"Action":"pass","Package":"example.com/foo","Elapsed":0.005}`

const goJSONFail = `{"Action":"run","Package":"example.com/foo","Test":"TestBeta"}
{"Action":"output","Package":"example.com/foo","Test":"TestBeta","Output":"=== RUN   TestBeta\n"}
{"Action":"output","Package":"example.com/foo","Test":"TestBeta","Output":"    beta_test.go:42: expected 1, got 2\n"}
{"Action":"fail","Package":"example.com/foo","Test":"TestBeta","Elapsed":0.002}
{"Action":"fail","Package":"example.com/foo","Elapsed":0.010}`

const goJSONMixed = `{"Action":"run","Package":"example.com/foo","Test":"TestAlpha"}
{"Action":"pass","Package":"example.com/foo","Test":"TestAlpha","Elapsed":0.001}
{"Action":"run","Package":"example.com/foo","Test":"TestBeta"}
{"Action":"output","Package":"example.com/foo","Test":"TestBeta","Output":"    beta_test.go:42: expected 1, got 2\n"}
{"Action":"fail","Package":"example.com/foo","Test":"TestBeta","Elapsed":0.002}
{"Action":"run","Package":"example.com/foo","Test":"TestGamma"}
{"Action":"skip","Package":"example.com/foo","Test":"TestGamma","Elapsed":0.000}
{"Action":"pass","Package":"example.com/foo","Elapsed":0.010}`

const goJSONCompileError = `# example.com/foo
./foo.go:5:2: undefined: Bar
FAIL    example.com/foo [build failed]`

func TestParseGoJSON_AllPass(t *testing.T) {
	r, err := ParseGoJSON(strings.NewReader(goJSONPass))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Suite != "go" {
		t.Errorf("Suite = %q, want %q", r.Suite, "go")
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
		t.Errorf("Failures = %d, want 0", len(r.Failures))
	}
	if r.RawError != "" {
		t.Errorf("RawError = %q, want empty", r.RawError)
	}
}

func TestParseGoJSON_Failure(t *testing.T) {
	r, err := ParseGoJSON(strings.NewReader(goJSONFail))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Failed != 1 {
		t.Errorf("Failed = %d, want 1", r.Failed)
	}
	if len(r.Failures) != 1 {
		t.Fatalf("len(Failures) = %d, want 1", len(r.Failures))
	}
	f := r.Failures[0]
	if f.TestName != "TestBeta" {
		t.Errorf("TestName = %q, want %q", f.TestName, "TestBeta")
	}
	if f.Package != "example.com/foo" {
		t.Errorf("Package = %q, want %q", f.Package, "example.com/foo")
	}
	if f.File != "beta_test.go" {
		t.Errorf("File = %q, want %q", f.File, "beta_test.go")
	}
	if f.Line != 42 {
		t.Errorf("Line = %d, want 42", f.Line)
	}
	if f.ErrorMsg != "expected 1, got 2" {
		t.Errorf("ErrorMsg = %q, want %q", f.ErrorMsg, "expected 1, got 2")
	}
}

func TestParseGoJSON_Mixed(t *testing.T) {
	r, err := ParseGoJSON(strings.NewReader(goJSONMixed))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Total != 3 {
		t.Errorf("Total = %d, want 3", r.Total)
	}
	if r.Passed != 1 {
		t.Errorf("Passed = %d, want 1", r.Passed)
	}
	if r.Failed != 1 {
		t.Errorf("Failed = %d, want 1", r.Failed)
	}
	if r.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", r.Skipped)
	}
}

func TestParseGoJSON_CompileError(t *testing.T) {
	r, err := ParseGoJSON(strings.NewReader(goJSONCompileError))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.RawError == "" {
		t.Error("RawError should be set for non-JSON input")
	}
	if len(r.Failures) != 0 {
		t.Errorf("Failures = %d, want 0 when RawError is set", len(r.Failures))
	}
}

func TestParseGoJSON_EmptyInput(t *testing.T) {
	r, err := ParseGoJSON(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Total != 0 {
		t.Errorf("Total = %d, want 0 for empty input", r.Total)
	}
}

func TestParseGoOutputFileLine(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantFile string
		wantLine int
		wantMsg  string
	}{
		{
			name:     "standard go test failure",
			output:   "=== RUN   TestFoo\n    foo_test.go:99: assertion failed\n",
			wantFile: "foo_test.go",
			wantLine: 99,
			wantMsg:  "assertion failed",
		},
		{
			name:     "no match",
			output:   "no source location here",
			wantFile: "",
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "multiple lines picks first",
			output:   "    a_test.go:1: first error\n    b_test.go:2: second error\n",
			wantFile: "a_test.go",
			wantLine: 1,
			wantMsg:  "first error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file, line, msg := parseGoOutputFileLine(tc.output)
			if file != tc.wantFile {
				t.Errorf("file = %q, want %q", file, tc.wantFile)
			}
			if line != tc.wantLine {
				t.Errorf("line = %d, want %d", line, tc.wantLine)
			}
			if msg != tc.wantMsg {
				t.Errorf("msg = %q, want %q", msg, tc.wantMsg)
			}
		})
	}
}
