// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// T-2: buildArgs parity + result-event classification
// ---------------------------------------------------------------------------

// TestClaudeEnvDriver_BuildArgsParity verifies that ClaudeEnvDriver's effective
// CLI arg vector equals ClaudeCodeDriver.buildArgs for the same Run (T-2).
// ClaudeEnvDriver delegates directly to (&ClaudeCodeDriver{}).buildArgs, so
// the arg vector is structurally identical.
func TestClaudeEnvDriver_BuildArgsParity(t *testing.T) {
	d := &ClaudeCodeDriver{}

	t.Run("all required flags present", func(t *testing.T) {
		args := d.buildArgs(Run{PromptText: "implement it"})
		required := []string{
			"--permission-mode", "bypassPermissions",
			"--dangerously-skip-permissions",
			"-p", "implement it",
			"--output-format", "stream-json",
			"--verbose",
		}
		for _, flag := range required {
			if !slices.Contains(args, flag) {
				t.Errorf("expected flag %q in args %v", flag, args)
			}
		}
	})

	t.Run("model flag present when Model set", func(t *testing.T) {
		args := d.buildArgs(Run{PromptText: "x", Model: "claude-opus-4-6"})
		mIdx := slices.Index(args, "--model")
		if mIdx < 0 {
			t.Fatal("--model flag not found")
		}
		if mIdx+1 >= len(args) || args[mIdx+1] != "claude-opus-4-6" {
			t.Errorf("expected --model claude-opus-4-6, got %v", args[mIdx:])
		}
	})

	t.Run("model flag absent when Model empty", func(t *testing.T) {
		args := d.buildArgs(Run{PromptText: "x"})
		for i, a := range args {
			if a == "--model" {
				t.Errorf("unexpected --model at index %d when Model is empty", i)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// T-3: Environment injection + precedence
// ---------------------------------------------------------------------------

// envStubShim returns the path to a shell script that prints the process's
// working directory and all ANTHROPIC_* environment variables to stdout.
func envStubShim(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "claude")
	script := "#!/bin/sh\n" +
		"printf 'PWD_STUB=%s\\n' \"$(pwd)\"\n" +
		"env | grep '^ANTHROPIC_' || true\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatalf("writing env shim: %v", err)
	}
	return p
}

// prependPATH prepends dir to PATH so our fake claude is resolved first.
func prependPATH(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// drainProgress drains the Process progress channel and returns all raw lines.
func drainProgress(t *testing.T, proc Process) []string {
	t.Helper()
	var lines []string
	for ev := range proc.Progress() {
		lines = append(lines, ev.Raw)
	}
	return lines
}

// lastEnvValue returns the last occurrence of KEY=<value> from lines (last wins
// because append(os.Environ(), "KEY=new") keeps the last occurrence).
func lastEnvValue(lines []string, key string) string {
	prefix := key + "="
	var last string
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			last = strings.TrimPrefix(l, prefix)
		}
	}
	return last
}

// TestClaudeEnvDriver_EnvInjection verifies ANTHROPIC_BASE_URL and
// ANTHROPIC_AUTH_TOKEN are injected into the subprocess (T-3).
func TestClaudeEnvDriver_EnvInjection(t *testing.T) {
	tmpDir := t.TempDir()
	envStubShim(t, tmpDir)
	prependPATH(t, tmpDir)

	t.Run("BASE_URL and AUTH_TOKEN are injected", func(t *testing.T) {
		run := Run{
			RunID:       "env-basic",
			AgentName:   "env-agent",
			Role:        "analyst",
			PromptText:  "x",
			ProjectRoot: tmpDir,
			BaseURL:     "http://configured-endpoint:11434",
			AuthToken:   "configured-secret-abc",
		}
		proc, err := (&ClaudeEnvDriver{}).Start(context.Background(), run)
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
		lines := drainProgress(t, proc)
		_ = proc.Wait()

		if got := lastEnvValue(lines, "ANTHROPIC_BASE_URL"); got != run.BaseURL {
			t.Errorf("ANTHROPIC_BASE_URL: got %q, want %q", got, run.BaseURL)
		}
		if got := lastEnvValue(lines, "ANTHROPIC_AUTH_TOKEN"); got != run.AuthToken {
			t.Errorf("ANTHROPIC_AUTH_TOKEN: got %q, want %q", got, run.AuthToken)
		}
	})

	t.Run("configured values override inherited parent values", func(t *testing.T) {
		t.Setenv("ANTHROPIC_BASE_URL", "inherited-base-url")
		t.Setenv("ANTHROPIC_AUTH_TOKEN", "inherited-auth-token")

		run := Run{
			RunID:       "env-precedence",
			AgentName:   "env-agent",
			Role:        "analyst",
			PromptText:  "x",
			ProjectRoot: tmpDir,
			BaseURL:     "http://override-endpoint:11434",
			AuthToken:   "override-secret",
		}
		proc, err := (&ClaudeEnvDriver{}).Start(context.Background(), run)
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
		lines := drainProgress(t, proc)
		_ = proc.Wait()

		if got := lastEnvValue(lines, "ANTHROPIC_BASE_URL"); got != run.BaseURL {
			t.Errorf("ANTHROPIC_BASE_URL precedence: got %q, want configured %q", got, run.BaseURL)
		}
		if got := lastEnvValue(lines, "ANTHROPIC_AUTH_TOKEN"); got != run.AuthToken {
			t.Errorf("ANTHROPIC_AUTH_TOKEN precedence: got %q, want configured %q", got, run.AuthToken)
		}
	})

	t.Run("cmd.Dir is set to ProjectRoot", func(t *testing.T) {
		run := Run{
			RunID:       "env-dir",
			AgentName:   "env-agent",
			Role:        "analyst",
			PromptText:  "x",
			ProjectRoot: tmpDir,
			BaseURL:     "http://test:11434",
			AuthToken:   "tok",
		}
		proc, err := (&ClaudeEnvDriver{}).Start(context.Background(), run)
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
		lines := drainProgress(t, proc)
		_ = proc.Wait()

		var stubPWD string
		for _, l := range lines {
			if strings.HasPrefix(l, "PWD_STUB=") {
				stubPWD = strings.TrimPrefix(l, "PWD_STUB=")
				break
			}
		}
		if stubPWD == "" {
			t.Fatal("PWD_STUB not found in stub output")
		}
		// Resolve symlinks for macOS where t.TempDir() may be under /var → /private/var.
		wantResolved, _ := filepath.EvalSymlinks(tmpDir)
		gotResolved, _ := filepath.EvalSymlinks(stubPWD)
		if gotResolved == "" {
			gotResolved = stubPWD
		}
		if wantResolved == "" {
			wantResolved = tmpDir
		}
		if gotResolved != wantResolved {
			t.Errorf("cmd.Dir: got %q (resolved: %q), want %q (resolved: %q)", stubPWD, gotResolved, tmpDir, wantResolved)
		}
	})
}

// ---------------------------------------------------------------------------
// T-4: Streaming, TTFT, log file, kill/Wait parity
// ---------------------------------------------------------------------------

// streamStubShim writes a fake claude that emits a valid stream-json sequence:
// an assistant content event (triggers TTFT) followed by a terminal result event.
func streamStubShim(t *testing.T, dir string) {
	t.Helper()
	script := "#!/bin/sh\n" +
		"printf '%s\\n' " +
		`'{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}'` +
		"\n" +
		"printf '%s\\n' " +
		`'{"type":"result","subtype":"success","is_error":false}'` +
		"\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("writing stream shim: %v", err)
	}
}

func TestClaudeEnvDriver_StreamingAndTTFT(t *testing.T) {
	tmpDir := t.TempDir()
	streamStubShim(t, tmpDir)
	prependPATH(t, tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	var ttftMS int64 = -1

	run := Run{
		RunID:       "run-stream-1",
		AgentName:   "stream-agent",
		Role:        "analyst",
		Model:       "claude-opus-4-6",
		PromptText:  "stream test",
		ProjectRoot: tmpDir,
		BaseURL:     "http://localhost:11434",
		AuthToken:   "stream-token",
		LogPath:     logPath,
		OnTTFT:      func(ms int64) { ttftMS = ms },
	}

	proc, err := (&ClaudeEnvDriver{}).Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var events []ProgressEvent
	for ev := range proc.Progress() {
		events = append(events, ev)
	}

	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	// Progress yields events in order.
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d: %v", len(events), events)
	}
	if events[0].Event == nil || events[0].Event["type"] != "assistant" {
		t.Errorf("event[0]: expected assistant type, got %v", events[0].Event)
	}
	if events[1].Event == nil || events[1].Event["type"] != "result" {
		t.Errorf("event[1]: expected result type, got %v", events[1].Event)
	}

	// OnTTFT called once with non-negative value on first assistant content token.
	if ttftMS < 0 {
		t.Error("OnTTFT was not called")
	}

	// Log file has header, content, and footer.
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	logContent := string(logBytes)
	if !strings.Contains(logContent, "# kaos-control agent run "+run.RunID) {
		t.Error("log missing run ID header")
	}
	if !strings.Contains(logContent, "# agent=stream-agent") {
		t.Error("log missing agent header")
	}
	if !strings.Contains(logContent, "model=claude-opus-4-6") {
		t.Error("log missing model in header")
	}
	if !strings.Contains(logContent, `"type":"assistant"`) {
		t.Error("log missing assistant event line")
	}
	if !strings.Contains(logContent, `"type":"result"`) {
		t.Error("log missing result event line")
	}
	if !strings.Contains(logContent, "# finished=") {
		t.Error("log missing # finished= footer")
	}
}

func TestClaudeEnvDriver_KillAndWait(t *testing.T) {
	tmpDir := t.TempDir()

	// Stub that sleeps indefinitely.
	script := "#!/bin/sh\nsleep 60\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("writing sleep shim: %v", err)
	}
	prependPATH(t, tmpDir)

	run := Run{
		RunID:       "run-kill-1",
		AgentName:   "kill-agent",
		Role:        "analyst",
		PromptText:  "x",
		ProjectRoot: tmpDir,
		BaseURL:     "http://localhost:11434",
		AuthToken:   "tok",
	}

	proc, err := (&ClaudeEnvDriver{}).Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Drain progress in background.
	go func() {
		for range proc.Progress() {
		}
	}()

	if err := proc.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}

	// Wait must return non-nil error after Kill.
	done := make(chan error, 1)
	go func() { done <- proc.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Error("Wait after Kill should return non-nil error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not return within 5s after Kill")
	}
}

func TestClaudeEnvDriver_CleanExitReturnsNilWait(t *testing.T) {
	tmpDir := t.TempDir()
	streamStubShim(t, tmpDir)
	prependPATH(t, tmpDir)

	run := Run{
		RunID:       "run-clean-1",
		AgentName:   "clean-agent",
		Role:        "analyst",
		PromptText:  "x",
		ProjectRoot: tmpDir,
		BaseURL:     "http://localhost:11434",
		AuthToken:   "tok",
	}

	proc, err := (&ClaudeEnvDriver{}).Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	for range proc.Progress() {
	}

	if err := proc.Wait(); err != nil {
		t.Errorf("Wait on clean exit: got %v, want nil", err)
	}
}

// ---------------------------------------------------------------------------
// T-5: Secret hygiene — agent-level (log, stderr, progress)
// ---------------------------------------------------------------------------

func TestClaudeEnvDriver_SecretHygiene(t *testing.T) {
	const token = "sup3r-s3cr3t-auth-t0k3n"

	tmpDir := t.TempDir()
	// Stub that emits a canned result event — no env echoing.
	script := "#!/bin/sh\n" +
		"printf '%s\\n' " +
		`'{"type":"result","subtype":"success","is_error":false}'` +
		"\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("writing secret shim: %v", err)
	}
	prependPATH(t, tmpDir)

	logPath := filepath.Join(tmpDir, "secret-run.log")

	run := Run{
		RunID:       "run-secret-1",
		AgentName:   "secret-agent",
		Role:        "analyst",
		PromptText:  "test prompt",
		ProjectRoot: tmpDir,
		BaseURL:     "http://localhost:11434",
		AuthToken:   token,
		LogPath:     logPath,
	}

	proc, err := (&ClaudeEnvDriver{}).Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var events []ProgressEvent
	for ev := range proc.Progress() {
		events = append(events, ev)
	}
	_ = proc.Wait()

	// Token must not appear in any ProgressEvent.Raw.
	for i, ev := range events {
		if strings.Contains(ev.Raw, token) {
			t.Errorf("token found in event[%d].Raw: %q", i, ev.Raw)
		}
	}

	// Token must not appear in StderrTail.
	if tail := proc.StderrTail(); strings.Contains(tail, token) {
		t.Errorf("token found in StderrTail: %q", tail)
	}

	// Token must not appear in the log file.
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	logContent := string(logBytes)
	if strings.Contains(logContent, token) {
		t.Errorf("token literal found in log file; log:\n%s", logContent)
	}
	if strings.Contains(logContent, "ANTHROPIC_AUTH_TOKEN="+token) {
		t.Errorf("ANTHROPIC_AUTH_TOKEN=<token> found in log file")
	}
}
