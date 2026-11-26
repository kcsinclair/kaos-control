// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// runLogHeaderRe requires driver= and provider= to appear, in that order,
// on the "# agent=" line — the shape every driver must produce (FR-4). It
// tolerates an empty provider value (the CLI-driver case) but not an
// omitted token.
var runLogHeaderRe = regexp.MustCompile(`^# agent=\S* role=\S* driver=\S* provider=\S*( model=\S*)?$`)

// assertHeaderLine finds the "# agent=" header line in logContent and checks
// it matches runLogHeaderRe, failing the test with the full content on
// mismatch.
func assertHeaderLine(t *testing.T, logContent string) string {
	t.Helper()
	for _, line := range splitLines(logContent) {
		if len(line) > 2 && line[:2] == "# " && containsAll(line, "agent=", "role=") {
			if !runLogHeaderRe.MatchString(line) {
				t.Errorf("header line %q does not match %s\nfull log:\n%s", line, runLogHeaderRe.String(), logContent)
			}
			return line
		}
	}
	t.Fatalf("no header line found in log:\n%s", logContent)
	return ""
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !bytes.Contains([]byte(s), []byte(sub)) {
			return false
		}
	}
	return true
}

// TestWriteRunLogHeader_Format verifies the shared writeRunLogHeader helper
// (agent.go) emits driver= and provider= on the header line for both a
// non-empty and an empty provider, and that the args line is present only
// when args is non-nil (Milestone 5).
func TestWriteRunLogHeader_Format(t *testing.T) {
	t.Run("with provider and args", func(t *testing.T) {
		var buf bytes.Buffer
		run := Run{RunID: "r1", AgentName: "a1", Role: "analyst", Driver: "openai-compatible", ProviderName: "prov-a", Model: "m1"}
		writeRunLogHeader(&buf, run, []string{"--flag"})
		content := buf.String()
		assertHeaderLine(t, content)
		if !containsAll(content, "driver=openai-compatible", "provider=prov-a") {
			t.Errorf("expected driver/provider tokens in header, got:\n%s", content)
		}
		if !containsAll(content, "# args=") {
			t.Errorf("expected args line when args is non-nil, got:\n%s", content)
		}
	})

	t.Run("empty provider emits literal empty token, no args line", func(t *testing.T) {
		var buf bytes.Buffer
		run := Run{RunID: "r2", AgentName: "a2", Role: "analyst", Driver: "gemini-cli", ProviderName: "", Model: "m2"}
		writeRunLogHeader(&buf, run, nil)
		content := buf.String()
		line := assertHeaderLine(t, content)
		if !containsAll(line, "driver=gemini-cli", "provider=") {
			t.Errorf("expected driver=gemini-cli and literal provider= token, got: %q", line)
		}
		if containsAll(content, "# args=") {
			t.Errorf("did not expect args line when args is nil, got:\n%s", content)
		}
	})
}

// TestShellStubDriver_LogHeader_IncludesDriverProvider verifies shell-stub
// now writes a header (previously it wrote none) containing both driver=
// and provider= before the first output line (Milestone 5).
func TestShellStubDriver_LogHeader_IncludesDriverProvider(t *testing.T) {
	d := &ShellStubDriver{}
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "shell-stub.log")

	run := Run{
		RunID:        "run-shell-stub-1",
		AgentName:    "stub-agent",
		Role:         "developer",
		Driver:       "shell-stub",
		ProviderName: "",
		LogPath:      logPath,
		ProjectRoot:  tmpDir,
	}

	proc, err := d.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	for range proc.Progress() {
	}
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	content := string(logBytes)
	line := assertHeaderLine(t, content)
	if !containsAll(line, "driver=shell-stub", "provider=") {
		t.Errorf("shell-stub header missing driver=/provider= tokens: %q", line)
	}
}

// TestCodexCLIDriver_LogHeader_IncludesDriverProvider verifies the codex-cli
// header (via the shared writeRunLogHeader helper) contains driver= and an
// empty provider= token (Milestone 5).
func TestCodexCLIDriver_LogHeader_IncludesDriverProvider(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "codex-header.log")
	shimPath := filepath.Join(tmpDir, "fake-codex")
	shim := "#!/bin/sh\n" + "printf '%s\\n' '{\"type\":\"output\",\"text\":\"ok\"}'\n"
	if err := os.WriteFile(shimPath, []byte(shim), 0o755); err != nil {
		t.Fatalf("writing shim: %v", err)
	}

	driver := &CodexCLIDriver{BinaryPath: shimPath}
	run := Run{
		RunID:       "run-codex-header",
		AgentName:   "codex-agent",
		Role:        "developer",
		Driver:      "codex-cli",
		Model:       "gpt-5-codex",
		PromptText:  "do the thing",
		LogPath:     logPath,
		ProjectRoot: tmpDir,
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	for range proc.Progress() {
	}
	_ = proc.Wait()

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	content := string(logBytes)
	line := assertHeaderLine(t, content)
	if !containsAll(line, "driver=codex-cli", "provider=") {
		t.Errorf("codex-cli header missing driver=/provider= tokens: %q", line)
	}
}

// TestGeminiCliDriver_LogHeader_IncludesDriverProvider verifies the
// gemini-cli header (via the shared writeRunLogHeader helper) contains
// driver= and an empty provider= token (Milestone 5). Reuses the
// GO_WANT_HELPER_PROCESS re-exec trick from gemini_cli_test.go.
func TestGeminiCliDriver_LogHeader_IncludesDriverProvider(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "gemini-cli-header.log")

	driver := &GeminiCliDriver{BinaryPath: os.Args[0]}
	run := Run{
		RunID:       "run-gemini-cli-header",
		AgentName:   "gemini-cli-agent",
		Role:        "developer",
		Driver:      "gemini-cli",
		Model:       "gemini-cli-model",
		PromptText:  "do the thing",
		LogPath:     logPath,
		ProjectRoot: tmpDir,
	}

	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	for range proc.Progress() {
	}
	_ = proc.Wait()

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	content := string(logBytes)
	line := assertHeaderLine(t, content)
	if !containsAll(line, "driver=gemini-cli", "provider=") {
		t.Errorf("gemini-cli header missing driver=/provider= tokens: %q", line)
	}
}
