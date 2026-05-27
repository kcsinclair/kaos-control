// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestCodexBuildArgs(t *testing.T) {
	d := &CodexCLIDriver{}

	t.Run("minimal", func(t *testing.T) {
		args := d.buildArgs(Run{PromptText: "implement the thing"})
		want := []string{"exec", "--json", "--dangerously-bypass-approvals-and-sandbox", "implement the thing"}
		if !slices.Equal(args, want) {
			t.Fatalf("args=%v, want %v", args, want)
		}
	})

	t.Run("projectRootAndModel", func(t *testing.T) {
		args := d.buildArgs(Run{
			ProjectRoot: "/tmp/kaos-control",
			Model:       "gpt-5-codex",
			PromptText:  "x",
		})
		want := []string{"exec", "--json", "--dangerously-bypass-approvals-and-sandbox", "--cd", "/tmp/kaos-control", "--model", "gpt-5-codex", "x"}
		if !slices.Equal(args, want) {
			t.Fatalf("args=%v, want %v", args, want)
		}
		if got := args[len(args)-1]; got != "x" {
			t.Errorf("prompt should remain the final argument, got %q", got)
		}
	})

	t.Run("timeoutWhenSupported", func(t *testing.T) {
		args := d.buildArgsWithOptions(Run{
			PromptText:     "x",
			TimeoutMinutes: 30,
		}, true)
		want := []string{"exec", "--json", "--dangerously-bypass-approvals-and-sandbox", "--timeout", "1800", "x"}
		if !slices.Equal(args, want) {
			t.Fatalf("args=%v, want %v", args, want)
		}
	})

	t.Run("unlimitedTimeoutSentinelWhenSupported", func(t *testing.T) {
		args := d.buildArgsWithOptions(Run{PromptText: "x"}, true)
		want := []string{"exec", "--json", "--dangerously-bypass-approvals-and-sandbox", "--timeout", "86400", "x"}
		if !slices.Equal(args, want) {
			t.Fatalf("args=%v, want %v", args, want)
		}
	})
}

func TestCodexExecSupportsTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	withTimeout := filepath.Join(tmpDir, "codex-with-timeout")
	if err := os.WriteFile(withTimeout, []byte("#!/bin/sh\nprintf '%s\\n' 'Usage: codex exec [OPTIONS] [PROMPT]'\nprintf '%s\\n' '      --timeout <SECONDS>'\n"), 0o755); err != nil {
		t.Fatalf("writing timeout shim: %v", err)
	}
	if !codexExecSupportsTimeout(context.Background(), withTimeout) {
		t.Fatal("expected timeout shim to be detected as supporting --timeout")
	}

	withoutTimeout := filepath.Join(tmpDir, "codex-without-timeout")
	if err := os.WriteFile(withoutTimeout, []byte("#!/bin/sh\nprintf '%s\\n' 'Usage: codex exec [OPTIONS] [PROMPT]'\n"), 0o755); err != nil {
		t.Fatalf("writing no-timeout shim: %v", err)
	}
	if codexExecSupportsTimeout(context.Background(), withoutTimeout) {
		t.Fatal("expected no-timeout shim to be detected as not supporting --timeout")
	}
}

func TestCodexCLIDriver_Start(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_codex_cli.log")
	shimPath := filepath.Join(tmpDir, "fake-codex")
	shim := "#!/bin/sh\n" +
		"printf '%s\\n' 'Starting mock codex run'\n" +
		"printf '%s\\n' '{\"type\":\"output\",\"text\":\"json line\\\\n\"}'\n" +
		"printf '%s\\n' 'Final raw text output from mock codex'\n"
	if err := os.WriteFile(shimPath, []byte(shim), 0o755); err != nil {
		t.Fatalf("writing shim: %v", err)
	}

	driver := &CodexCLIDriver{BinaryPath: shimPath}
	run := Run{
		RunID:       "run-codex-123",
		AgentName:   "qa-agent",
		Role:        "qa",
		Model:       "gpt-5-codex",
		PromptText:  "Analyze this code",
		LogPath:     logPath,
		ProjectRoot: tmpDir,
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var events []ProgressEvent
	for ev := range proc.Progress() {
		events = append(events, ev)
	}

	if err := proc.Wait(); err != nil {
		t.Fatalf("process exited with error: %v", err)
	}

	if len(events) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(events))
	}
	if events[0].Raw != "started" || events[0].Event["type"] != "started" {
		t.Errorf("expected first event to be started, got %v", events[0])
	}
	if events[1].Event["type"] != "output" || events[1].Event["text"] != "Starting mock codex run\n" {
		t.Errorf("expected raw line to be wrapped, got %v", events[1].Event)
	}
	if events[2].Event["type"] != "output" || events[2].Event["text"] != "json line\\n" {
		t.Errorf("expected JSON line to be parsed, got %v", events[2].Event)
	}
	if events[3].Event["type"] != "output" || events[3].Event["text"] != "Final raw text output from mock codex\n" {
		t.Errorf("expected final raw line to be wrapped, got %v", events[3].Event)
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	logContent := string(logBytes)
	if !strings.Contains(logContent, "--cd "+tmpDir) {
		t.Errorf("expected log args to include --cd %s; content:\n%s", tmpDir, logContent)
	}
	if !strings.Contains(logContent, "# finished=") {
		t.Errorf("expected log content to contain finished marker")
	}
}

func TestCodexCLIDriver_DetachedChildHoldsPipes(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not found; skipping detached-child test")
	}

	tmpDir := t.TempDir()
	shimPath := filepath.Join(tmpDir, "fake-codex")
	shim := "#!/usr/bin/env " + python + "\n" +
		"import os, sys, time\n" +
		"sys.stderr.write('Error: mock codex exited early\\n')\n" +
		"sys.stderr.flush()\n" +
		"if os.fork() == 0:\n" +
		"    time.sleep(30)\n" +
		"    sys.exit(0)\n" +
		"sys.exit(1)\n"
	if err := os.WriteFile(shimPath, []byte(shim), 0o755); err != nil {
		t.Fatalf("writing shim: %v", err)
	}

	driver := &CodexCLIDriver{BinaryPath: shimPath}
	run := Run{
		RunID:       "run-detached-child",
		AgentName:   "qa",
		Role:        "qa",
		Model:       "gpt-5-codex",
		PromptText:  "trigger the hang",
		LogPath:     filepath.Join(tmpDir, "run.log"),
		ProjectRoot: tmpDir,
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	drainDone := make(chan struct{})
	go func() {
		for range proc.Progress() {
		}
		close(drainDone)
	}()

	select {
	case <-drainDone:
	case <-time.After(5 * time.Second):
		_ = proc.Kill()
		t.Fatal("progress channel was not closed within 5s after codex shim exited")
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- proc.Wait() }()
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("proc.Wait() did not return within 2s after drain completed")
	}

	logBytes, err := os.ReadFile(run.LogPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	if !strings.Contains(string(logBytes), "# finished=") {
		t.Errorf("log file missing # finished= footer; content:\n%s", string(logBytes))
	}
}

func TestCodexCLIDriver_LiveRoundTrip(t *testing.T) {
	if os.Getenv("KAOS_TEST_LIVE_CODEX") != "1" {
		t.Skip("set KAOS_TEST_LIVE_CODEX=1 to run the live Codex CLI smoke test")
	}
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex not found on PATH; skipping live Codex CLI smoke test")
	}

	repoRoot := findRepoRoot(t)
	beforeStatus := gitStatusPorcelain(t, repoRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	prompt := "Do not inspect or modify files. Reply with exactly: codex-roundtrip-ok"
	cmd := exec.CommandContext(ctx,
		"codex", "exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
		"--cd", repoRoot,
		prompt,
	)
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader("")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("live codex round trip failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if ctx.Err() != nil {
		t.Fatalf("live codex round trip timed out: %v", ctx.Err())
	}

	if got := gitStatusPorcelain(t, repoRoot); got != beforeStatus {
		t.Fatalf("live codex round trip changed git status\nbefore:\n%s\nafter:\n%s", beforeStatus, got)
	}

	foundThreadStarted := false
	foundTurnCompleted := false
	foundExpectedMessage := false
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("invalid JSONL event %q: %v\nstdout:\n%s\nstderr:\n%s", line, err, stdout.String(), stderr.String())
		}
		switch event["type"] {
		case "thread.started":
			foundThreadStarted = true
		case "turn.completed":
			foundTurnCompleted = true
		case "item.completed":
			item, _ := event["item"].(map[string]any)
			if item["type"] == "agent_message" && item["text"] == "codex-roundtrip-ok" {
				foundExpectedMessage = true
			}
		}
	}

	if !foundThreadStarted {
		t.Fatalf("live codex output missing thread.started event\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !foundTurnCompleted {
		t.Fatalf("live codex output missing turn.completed event\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !foundExpectedMessage {
		t.Fatalf("live codex output missing expected item.completed agent message\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func gitStatusPorcelain(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git status --porcelain: %v", err)
	}
	return string(out)
}
