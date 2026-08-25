// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Milestone 2 — ESLint Rule Enforcement & Negative Test Validation
// (lifecycle/test-plans/frontend-lint-gap-5-test.md)
//
// Runs the real web/eslint.config.js against fixture files in
// tests/fixtures/frontend-lint/ to verify that each prohibited defect
// pattern from FR-3 is flagged with the correct rule, and that the
// permitted counterparts pass cleanly.
package cli_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

// eslintMessage mirrors one entry in ESLint's `-f json` message array.
type eslintMessage struct {
	RuleID   string `json:"ruleId"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity int    `json:"severity"`
}

// eslintResult mirrors one per-file entry in ESLint's `-f json` output.
type eslintResult struct {
	FilePath   string          `json:"filePath"`
	Messages   []eslintMessage `json:"messages"`
	ErrorCount int             `json:"errorCount"`
}

// eslintBinPath locates the eslint binary installed under web/node_modules.
// Tests using it skip (rather than fail) when node_modules has not been
// installed, since that is an environment-setup precondition, not a defect.
func eslintBinPath(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "web", "node_modules", ".bin", "eslint")
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("eslint binary not found at %s (run `pnpm install` in web/): %v", bin, err)
	}
	return bin
}

// runESLintJSON runs eslint against a single fixture file (path relative to
// the repo root) using web/eslint.config.js and parses the `-f json` output.
func runESLintJSON(t *testing.T, relPath string) (eslintResult, int) {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	bin := eslintBinPath(t)

	cmd := exec.Command(bin, "-c", "web/eslint.config.js", "-f", "json", relPath)
	cmd.Dir = root
	out, runErr := cmd.Output()

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running eslint on %s: %v", relPath, runErr)
		}
	}

	return decodeESLintJSON(t, relPath, out), exitCode
}

// decodeESLintJSON parses ESLint's `-f json` output for a single-file lint
// invocation, shared by both web/eslint.config.js and tests/web/eslint.config.js
// test runners.
func decodeESLintJSON(t *testing.T, relPath string, out []byte) eslintResult {
	t.Helper()
	var results []eslintResult
	if err := json.Unmarshal(out, &results); err != nil {
		t.Fatalf("parsing eslint JSON output for %s: %v\noutput: %s", relPath, err, out)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result entry for %s, got %d", relPath, len(results))
	}
	return results[0]
}

// lintViolationCases maps each fixture under
// tests/fixtures/frontend-lint/violations/ to the FR-3 rule it must trigger.
var lintViolationCases = []struct {
	name   string
	file   string
	ruleID string
}{
	{"unused var without _ prefix", "tests/fixtures/frontend-lint/violations/no-unused-vars.violation.ts", "@typescript-eslint/no-unused-vars"},
	{"floating promise", "tests/fixtures/frontend-lint/violations/floating-promises.violation.ts", "@typescript-eslint/no-floating-promises"},
	{"misused promise in sync callback", "tests/fixtures/frontend-lint/violations/misused-promises.violation.ts", "@typescript-eslint/no-misused-promises"},
	{"unused Vue component", "tests/fixtures/frontend-lint/violations/UnusedComponent.violation.vue", "vue/no-unused-components"},
	{"direct prop mutation", "tests/fixtures/frontend-lint/violations/MutatingProps.violation.vue", "vue/no-mutating-props"},
	{"raw v-html", "tests/fixtures/frontend-lint/violations/VHtml.violation.vue", "vue/no-v-html"},
	{"loose equality", "tests/fixtures/frontend-lint/violations/eqeqeq.violation.ts", "eqeqeq"},
	{"let that should be const", "tests/fixtures/frontend-lint/violations/prefer-const.violation.ts", "prefer-const"},
}

// TestFrontendLintRules_ViolationsAreFlagged verifies that each prohibited
// defect pattern in FR-3 triggers a lint failure with the expected rule name,
// and that diagnostics carry file path, line, column, and rule name (NFR-4).
func TestFrontendLintRules_ViolationsAreFlagged(t *testing.T) {
	for _, tc := range lintViolationCases {
		t.Run(tc.name, func(t *testing.T) {
			result, exitCode := runESLintJSON(t, tc.file)

			if exitCode == 0 {
				t.Errorf("expected non-zero exit code for %s, got 0", tc.file)
			}
			if result.ErrorCount == 0 {
				t.Fatalf("expected at least 1 error for %s, got 0", tc.file)
			}

			var found *eslintMessage
			for i := range result.Messages {
				if result.Messages[i].RuleID == tc.ruleID {
					found = &result.Messages[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("expected rule %q to fire on %s, got rules: %v", tc.ruleID, tc.file, ruleIDs(result.Messages))
			}

			// NFR-4: diagnostics must carry file path, line, column, and rule name.
			if result.FilePath == "" {
				t.Error("diagnostic missing file path")
			}
			if found.Line <= 0 {
				t.Errorf("diagnostic for %s has non-positive line %d", tc.ruleID, found.Line)
			}
			if found.Column <= 0 {
				t.Errorf("diagnostic for %s has non-positive column %d", tc.ruleID, found.Column)
			}
		})
	}
}

// lintValidCases lists fixtures under tests/fixtures/frontend-lint/valid/
// that demonstrate the permitted counterpart of each FR-3 pattern and must
// pass without errors.
var lintValidCases = []struct {
	name string
	file string
}{
	{"_-prefixed unused arg is exempt", "tests/fixtures/frontend-lint/valid/no-unused-vars.valid.ts"},
	{"void-discarded promise", "tests/fixtures/frontend-lint/valid/floating-promises.valid.ts"},
	{"sync callback wrapping an async call", "tests/fixtures/frontend-lint/valid/misused-promises.valid.ts"},
	{"strict equality and == null idiom", "tests/fixtures/frontend-lint/valid/eqeqeq.valid.ts"},
	{"const for never-reassigned binding", "tests/fixtures/frontend-lint/valid/prefer-const.valid.ts"},
	{"used component, read-only props, no v-html", "tests/fixtures/frontend-lint/valid/valid-component.valid.vue"},
}

// TestFrontendLintRules_ValidPatternsPass verifies that the permitted
// counterpart of each prohibited pattern (underscore-prefixed args, void
// promises, strict equality, etc.) lints cleanly.
func TestFrontendLintRules_ValidPatternsPass(t *testing.T) {
	for _, tc := range lintValidCases {
		t.Run(tc.name, func(t *testing.T) {
			result, exitCode := runESLintJSON(t, tc.file)

			if exitCode != 0 {
				t.Errorf("expected exit code 0 for %s, got %d (messages: %v)", tc.file, exitCode, result.Messages)
			}
			if result.ErrorCount != 0 {
				t.Errorf("expected 0 errors for %s, got %d: %v", tc.file, result.ErrorCount, ruleIDs(result.Messages))
			}
		})
	}
}

func ruleIDs(msgs []eslintMessage) []string {
	ids := make([]string, len(msgs))
	for i, m := range msgs {
		ids[i] = m.RuleID
	}
	return ids
}
