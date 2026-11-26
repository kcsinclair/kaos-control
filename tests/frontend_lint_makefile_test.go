// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Milestone 1 — Makefile Target & Fail-Fast Integration Tests
// (lifecycle/test-plans/frontend-lint-gap-5-test.md)
//
// Verifies that `make lint-frontend`, `make lint-go`, and `make lint` invoke
// the expected sub-commands (via `make -n` dry runs against the real
// Makefile, so no toolchain needs to be installed) and that `make lint`
// propagates a non-zero exit code when either half fails, halting before the
// remaining half runs (fail-fast, exercised against a synthetic Makefile
// mirroring the real target structure).
package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// dryRunMake runs `make -n <target>` against the real repo Makefile and
// returns combined stdout+stderr. `-n` prints the commands a target would
// run without executing them, so this needs no linters/tools installed.
func dryRunMake(t *testing.T, target string) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("make", "-n", target)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n %s failed: %v\noutput:\n%s", target, err, out)
	}
	return string(out)
}

// TestMakefileLintFrontend_InvokesExpectedCommands verifies that
// `make lint-frontend` runs `pnpm run lint` followed by `vue-tsc --noEmit`.
func TestMakefileLintFrontend_InvokesExpectedCommands(t *testing.T) {
	out := dryRunMake(t, "lint-frontend")

	lintIdx := strings.Index(out, "pnpm run lint")
	tscIdx := strings.Index(out, "vue-tsc --noEmit")
	if lintIdx == -1 {
		t.Errorf("lint-frontend does not invoke `pnpm run lint`:\n%s", out)
	}
	if tscIdx == -1 {
		t.Errorf("lint-frontend does not invoke `vue-tsc --noEmit`:\n%s", out)
	}
	if lintIdx != -1 && tscIdx != -1 && lintIdx > tscIdx {
		t.Errorf("expected `pnpm run lint` to run before `vue-tsc --noEmit`:\n%s", out)
	}
}

// TestMakefileLintGo_InvokesExpectedCommands verifies that `make lint-go`
// runs go vet, staticcheck, govulncheck, gosec, and gitleaks.
func TestMakefileLintGo_InvokesExpectedCommands(t *testing.T) {
	out := dryRunMake(t, "lint-go")

	for _, want := range []string{"go vet ./...", "staticcheck", "govulncheck", "gosec", "gitleaks"} {
		if !strings.Contains(out, want) {
			t.Errorf("lint-go does not invoke %q:\n%s", want, out)
		}
	}
}

// TestMakefileLint_InvokesBoth verifies that `make lint` runs the lint-go
// commands before the lint-frontend commands (backend first, frontend
// second — cheaper/faster static analysis fails fastest).
func TestMakefileLint_InvokesBoth(t *testing.T) {
	out := dryRunMake(t, "lint")

	goIdx := strings.Index(out, "go vet ./...")
	frontendIdx := strings.Index(out, "pnpm run lint")
	if goIdx == -1 {
		t.Errorf("lint does not invoke lint-go's `go vet ./...`:\n%s", out)
	}
	if frontendIdx == -1 {
		t.Errorf("lint does not invoke lint-frontend's `pnpm run lint`:\n%s", out)
	}
	if goIdx != -1 && frontendIdx != -1 && goIdx > frontendIdx {
		t.Errorf("expected lint-go's commands to run before lint-frontend's:\n%s", out)
	}
}

// writeSyntheticLintMakefile creates a temp directory containing a Makefile
// with the same `lint: lint-go lint-frontend` dependency shape as the real
// root Makefile, but with fast, controllable step outcomes. This isolates
// the fail-fast *mechanism* (make's default sequential-prerequisite halt on
// first failure) from the real lint-go/lint-frontend recipes, which require
// a fully provisioned toolchain (staticcheck, gosec, pnpm, etc.) to execute.
func writeSyntheticLintMakefile(t *testing.T, goRecipe, frontendRecipe string) string {
	t.Helper()
	dir := t.TempDir()
	makefile := "" +
		".PHONY: lint lint-go lint-frontend\n\n" +
		"lint: lint-go lint-frontend\n\n" +
		"lint-go:\n\t" + goRecipe + "\n\n" +
		"lint-frontend:\n\t" + frontendRecipe + "\n"
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(makefile), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestMakefileLint_FailFast_BackendFailureStopsFrontend verifies that a
// lint-go failure causes `make lint` to exit non-zero and halts before
// lint-frontend's recipe ever runs.
func TestMakefileLint_FailFast_BackendFailureStopsFrontend(t *testing.T) {
	dir := writeSyntheticLintMakefile(t, "@exit 1", "@touch frontend-ran.marker")

	cmd := exec.Command("make", "lint")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected `make lint` to fail, but it succeeded:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "frontend-ran.marker")); statErr == nil {
		t.Error("lint-frontend ran despite lint-go failing; fail-fast did not halt execution")
	}
}

// TestMakefileLint_FailFast_FrontendFailurePropagates verifies that a
// lint-frontend failure (with lint-go passing) still causes `make lint` to
// exit non-zero.
func TestMakefileLint_FailFast_FrontendFailurePropagates(t *testing.T) {
	dir := writeSyntheticLintMakefile(t, "@true", "@exit 1")

	cmd := exec.Command("make", "lint")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected `make lint` to fail when lint-frontend fails, but it succeeded:\n%s", out)
	}
}
