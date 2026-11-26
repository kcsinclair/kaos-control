// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Milestone 3 — Test Suite Override & Mock Ergonomics Validation
// (lifecycle/test-plans/frontend-lint-gap-5-test.md)
//
// Runs the real tests/web/eslint.config.js (the ergonomics-relaxed config
// used for tests/web/*.test.ts files) against fixtures in
// tests/fixtures/frontend-lint/test-overrides/ to verify that `any` is
// permitted for mocks/spies, while genuine bugs (floating promises, unused
// non-underscore-prefixed imports) are still caught.
package cli_test

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// testWebEslintBinPath locates the eslint binary installed under
// tests/web/node_modules. Skips (rather than fails) when not installed,
// since that is an environment-setup precondition, not a defect.
func testWebEslintBinPath(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "tests", "web", "node_modules", ".bin", "eslint")
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("eslint binary not found at %s (run `pnpm install` in tests/web/): %v", bin, err)
	}
	return bin
}

// runTestsWebESLintJSON runs eslint against a single fixture file (path
// relative to the repo root) using tests/web/eslint.config.js.
func runTestsWebESLintJSON(t *testing.T, relPath string) (eslintResult, int) {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	bin := testWebEslintBinPath(t)

	cmd := exec.Command(bin, "-c", "tests/web/eslint.config.js", "-f", "json", relPath)
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

	results := decodeESLintJSON(t, relPath, out)
	return results, exitCode
}

// TestFrontendLintTestOverrides_AnyIsPermittedForMocks verifies that a test
// fixture using `any` for mock objects/spy return values lints cleanly under
// tests/web/eslint.config.js, even though production code (web/eslint.config.js)
// forbids @typescript-eslint/no-explicit-any.
func TestFrontendLintTestOverrides_AnyIsPermittedForMocks(t *testing.T) {
	const file = "tests/fixtures/frontend-lint/test-overrides/any-mock.valid.ts"
	result, exitCode := runTestsWebESLintJSON(t, file)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d: %v", exitCode, ruleIDs(result.Messages))
	}
	if result.ErrorCount != 0 {
		t.Errorf("expected 0 errors for a file using `any` in mocks, got %d: %v", result.ErrorCount, ruleIDs(result.Messages))
	}
}

// TestFrontendLintTestOverrides_RealBugsAreCaught verifies that genuine bugs
// in test files — a floating promise and a non-underscore-prefixed unused
// import — are still flagged despite the test-file ergonomics overrides.
func TestFrontendLintTestOverrides_RealBugsAreCaught(t *testing.T) {
	cases := []struct {
		name   string
		file   string
		ruleID string
	}{
		{"floating promise", "tests/fixtures/frontend-lint/test-overrides/floating-promise.violation.ts", "@typescript-eslint/no-floating-promises"},
		{"unused import without _ prefix", "tests/fixtures/frontend-lint/test-overrides/unused-import.violation.ts", "@typescript-eslint/no-unused-vars"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, exitCode := runTestsWebESLintJSON(t, tc.file)

			if exitCode == 0 {
				t.Errorf("expected non-zero exit code for %s, got 0", tc.file)
			}
			if result.ErrorCount == 0 {
				t.Fatalf("expected at least 1 error for %s, got 0", tc.file)
			}

			var found bool
			for _, m := range result.Messages {
				if m.RuleID == tc.ruleID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected rule %q to fire on %s, got rules: %v", tc.ruleID, tc.file, ruleIDs(result.Messages))
			}
		})
	}
}
