// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Executor runs each test suite sequentially and returns normalised results.
type Executor struct {
	// cmdFunc is used by tests to inject a mock command builder.
	// If nil, exec.CommandContext is used.
	cmdFunc func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// command returns the command to run, using cmdFunc if set.
func (e *Executor) command(ctx context.Context, name string, args ...string) *exec.Cmd {
	if e.cmdFunc != nil {
		return e.cmdFunc(ctx, name, args...)
	}
	return exec.CommandContext(ctx, name, args...)
}

// RunAll runs all configured test suites sequentially. Earlier failures do not
// prevent later suites from running. Playwright is skipped if tests/e2e/ does
// not exist under projectDir.
func (e *Executor) RunAll(ctx context.Context, projectDir string) ([]SuiteResult, error) {
	var results []SuiteResult

	goResult := e.runGoTests(ctx, projectDir)
	results = append(results, goResult)

	vitestDir := filepath.Join(projectDir, "tests", "web")
	if _, err := os.Stat(vitestDir); err == nil {
		vitestResult := e.runVitestTests(ctx, vitestDir)
		results = append(results, vitestResult)
	}

	e2eDir := filepath.Join(projectDir, "tests", "e2e")
	if _, err := os.Stat(e2eDir); err == nil {
		pwResult := e.runPlaywrightTests(ctx, e2eDir)
		results = append(results, pwResult)
	}

	return results, nil
}

// runGoTests runs `go test -json -count=1 ./...` from projectDir.
func (e *Executor) runGoTests(ctx context.Context, projectDir string) SuiteResult {
	start := time.Now()

	cmd := e.command(ctx, "go", "test", "-json", "-count=1", "./...")
	cmd.Dir = projectDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return SuiteResult{Suite: "go", RawError: err.Error(), Elapsed: time.Since(start).Seconds()}
	}

	if err := cmd.Start(); err != nil {
		return SuiteResult{Suite: "go", RawError: err.Error(), Elapsed: time.Since(start).Seconds()}
	}

	result, parseErr := ParseGoJSON(stdout)
	_ = cmd.Wait() // capture exit code; non-zero is expected on test failures

	elapsed := time.Since(start).Seconds()
	if result == nil {
		result = &SuiteResult{Suite: "go"}
	}
	if parseErr != nil {
		result.RawError = parseErr.Error()
	}
	result.Elapsed = elapsed

	if stderrStr := stderr.String(); stderrStr != "" {
		slog.Debug("go test stderr", "output", stderrStr)
	}

	return *result
}

// runVitestTests runs `pnpm exec vitest run --reporter=json` from vitestDir.
func (e *Executor) runVitestTests(ctx context.Context, vitestDir string) SuiteResult {
	start := time.Now()

	cmd := e.command(ctx, "pnpm", "exec", "vitest", "run", "--reporter=json")
	cmd.Dir = vitestDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return SuiteResult{Suite: "vitest", RawError: err.Error(), Elapsed: time.Since(start).Seconds()}
	}

	if err := cmd.Start(); err != nil {
		return SuiteResult{Suite: "vitest", RawError: err.Error(), Elapsed: time.Since(start).Seconds()}
	}

	result, parseErr := ParseVitestJSON(stdout)
	_ = cmd.Wait()

	elapsed := time.Since(start).Seconds()
	if result == nil {
		result = &SuiteResult{Suite: "vitest"}
	}
	if parseErr != nil {
		result.RawError = parseErr.Error()
	}
	result.Elapsed = elapsed

	if stderrStr := stderr.String(); stderrStr != "" {
		slog.Debug("vitest stderr", "output", stderrStr)
	}

	return *result
}

// runPlaywrightTests runs `pnpm exec playwright test --reporter=json` from e2eDir.
func (e *Executor) runPlaywrightTests(ctx context.Context, e2eDir string) SuiteResult {
	start := time.Now()

	cmd := e.command(ctx, "pnpm", "exec", "playwright", "test", "--reporter=json")
	cmd.Dir = e2eDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return SuiteResult{Suite: "playwright", RawError: err.Error(), Elapsed: time.Since(start).Seconds()}
	}

	if err := cmd.Start(); err != nil {
		return SuiteResult{Suite: "playwright", RawError: err.Error(), Elapsed: time.Since(start).Seconds()}
	}

	result, parseErr := ParsePlaywrightJSON(stdout)
	_ = cmd.Wait()

	elapsed := time.Since(start).Seconds()
	if result == nil {
		result = &SuiteResult{Suite: "playwright"}
	}
	if parseErr != nil {
		result.RawError = parseErr.Error()
	}
	result.Elapsed = elapsed

	if stderrStr := stderr.String(); stderrStr != "" {
		slog.Debug("playwright stderr", "output", stderrStr)
	}

	return *result
}
