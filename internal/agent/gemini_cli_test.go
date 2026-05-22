// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// init intercepts the test execution when spawned as a mock subprocess
// to print mock stdout/stderr streams and exit cleanly before the
// testing framework tries to parse custom CLI flags.
func init() {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		fmt.Println("Starting mock agy run")
		fmt.Println(`{"type":"output","text":"json line\n"}`)
		fmt.Println("Final raw text output from mock agy")
		os.Exit(0)
	}
}

func TestGeminiCliDriver_BuildArgs(t *testing.T) {
	driver := &GeminiCliDriver{}

	t.Run("withProjectRootAndUnlimitedTimeout", func(t *testing.T) {
		run := Run{
			ProjectRoot: "/Users/keith/Code/kaos-control",
			PromptText:  "Hello Gemini",
			// TimeoutMinutes=0 → 24h
		}
		args := driver.buildArgs(run)

		expectedArgs := []string{
			"--dangerously-skip-permissions",
			"--add-dir", "/Users/keith/Code/kaos-control",
			"--print-timeout", "24h",
			"--prompt", "Hello Gemini",
		}

		if len(args) != len(expectedArgs) {
			t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(args), args)
		}
		for i, arg := range args {
			if arg != expectedArgs[i] {
				t.Errorf("arg %d: expected %q, got %q", i, expectedArgs[i], arg)
			}
		}
	})

	t.Run("withExplicitTimeout", func(t *testing.T) {
		run := Run{
			ProjectRoot:    "/tmp/proj",
			PromptText:     "Hello Gemini",
			TimeoutMinutes: 30,
		}
		args := driver.buildArgs(run)

		expectedArgs := []string{
			"--dangerously-skip-permissions",
			"--add-dir", "/tmp/proj",
			"--print-timeout", "30m",
			"--prompt", "Hello Gemini",
		}

		if len(args) != len(expectedArgs) {
			t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(args), args)
		}
		for i, arg := range args {
			if arg != expectedArgs[i] {
				t.Errorf("arg %d: expected %q, got %q", i, expectedArgs[i], arg)
			}
		}
	})

	t.Run("withoutProjectRoot", func(t *testing.T) {
		run := Run{
			PromptText: "Hello Gemini",
		}
		args := driver.buildArgs(run)

		expectedArgs := []string{
			"--dangerously-skip-permissions",
			"--print-timeout", "24h",
			"--prompt", "Hello Gemini",
		}

		if len(args) != len(expectedArgs) {
			t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(args), args)
		}
		for i, arg := range args {
			if arg != expectedArgs[i] {
				t.Errorf("arg %d: expected %q, got %q", i, expectedArgs[i], arg)
			}
		}
	})
}

func TestGeminiCliDriver_Start(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_gemini_cli.log")

	driver := &GeminiCliDriver{
		BinaryPath: os.Args[0], // Re-execute this test binary
	}

	run := Run{
		RunID:       "run-cli-123",
		AgentName:   "qa-agent",
		Role:        "qa",
		Model:       "gemini-cli-model",
		PromptText:  "Analyze this code",
		LogPath:     logPath,
		ProjectRoot: tmpDir,
	}

	// Inject environment variables to trigger the helper process in init()
	ctx := context.WithValue(context.Background(), "test", true)
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	proc, err := driver.Start(ctx, run)
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	// Consume progress events
	var events []ProgressEvent
	for ev := range proc.Progress() {
		events = append(events, ev)
	}

	// Wait for process to exit
	if err := proc.Wait(); err != nil {
		t.Fatalf("process exited with error: %v", err)
	}

	// Verify events
	// 1st event should be started, followed by 3 output lines
	if len(events) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(events))
	}

	if events[0].Raw != "started" || events[0].Event["type"] != "started" {
		t.Errorf("expected 1st event to be started, got %v", events[0])
	}

	// 2nd event: raw text line -> "type": "output"
	if events[1].Raw != "Starting mock agy run" {
		t.Errorf("expected 2nd event raw: %q, got: %q", "Starting mock agy run", events[1].Raw)
	}
	if events[1].Event["type"] != "output" || events[1].Event["text"] != "Starting mock agy run\n" {
		t.Errorf("expected 2nd event to be wrapped, got %v", events[1].Event)
	}

	// 3rd event: JSON string -> parsed successfully
	if events[2].Raw != `{"type":"output","text":"json line\n"}` {
		t.Errorf("expected 3rd event raw: %q, got: %q", `{"type":"output","text":"json line\n"}`, events[2].Raw)
	}
	if events[2].Event["type"] != "output" || events[2].Event["text"] != "json line\n" {
		t.Errorf("expected 3rd event parsed successfully, got %v", events[2].Event)
	}

	// 4th event: raw text line -> "type": "output"
	if events[3].Raw != "Final raw text output from mock agy" {
		t.Errorf("expected 4th event raw: %q, got: %q", "Final raw text output from mock agy", events[3].Raw)
	}
	if events[3].Event["type"] != "output" || events[3].Event["text"] != "Final raw text output from mock agy\n" {
		t.Errorf("expected 4th event to be wrapped, got %v", events[3].Event)
	}

	// Verify log file was written correctly
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(logBytes)
	if !strings.Contains(logContent, "Starting mock agy run") {
		t.Errorf("expected log content to contain first stdout line")
	}
	if !strings.Contains(logContent, "Final raw text output from mock agy") {
		t.Errorf("expected log content to contain final stdout line")
	}
	if !strings.Contains(logContent, "# finished=") {
		t.Errorf("expected log content to contain finished marker")
	}
}

// TestGeminiCliDriver_DetachedChildHoldsPipes is a regression test for the
// hang where the agy CLI exits but a detached grandchild keeps stdout/stderr
// FDs open. Real-world impact: supervise() drains the progress channel,
// which never closes because the pipe-drain goroutines are blocked on
// Read(), then would call proc.Wait() — but proc.Wait() is exactly the
// thing that would close the pipes and unblock the readers. The result is
// a permanent deadlock; the run stays marked "running" until the user
// manually kills it.
//
// The test mirrors supervise()'s sequencing exactly: drain progress FIRST,
// then call proc.Wait(). With the bug, the drain blocks forever. With the
// fix, cmd.Wait() runs in its own goroutine, closes the parent-side pipes
// when the shim exits, the readers EOF, the channel closes, drain exits,
// then proc.Wait() returns the stashed result.
//
// We reproduce the detached-grandchild scenario with a tiny Python script
// that fork()s an idle grandchild before the parent exits.
func TestGeminiCliDriver_DetachedChildHoldsPipes(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not found; skipping detached-child test")
	}

	tmpDir := t.TempDir()
	shimPath := filepath.Join(tmpDir, "fake-agy")
	shim := "#!/usr/bin/env " + python + "\n" +
		"import os, sys, time\n" +
		"sys.stderr.write('Error: timed out waiting for response\\n')\n" +
		"sys.stderr.flush()\n" +
		"if os.fork() == 0:\n" +
		"    time.sleep(30)\n" + // grandchild holds inherited stdout/stderr FDs
		"    sys.exit(0)\n" +
		"sys.exit(1)\n"
	if err := os.WriteFile(shimPath, []byte(shim), 0o755); err != nil {
		t.Fatalf("writing shim: %v", err)
	}

	driver := &GeminiCliDriver{BinaryPath: shimPath}
	run := Run{
		RunID:       "run-detached-child",
		AgentName:   "qa",
		Role:        "qa",
		Model:       "gemini-2.5-flash",
		PromptText:  "trigger the hang",
		LogPath:     filepath.Join(tmpDir, "run.log"),
		ProjectRoot: tmpDir,
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Mirror supervise(): drain progress first, then Wait. With the bug the
	// drain blocks forever; we cap it at 5 s.
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
		t.Fatal("progress channel was not closed within 5s after agy shim exited — daemon-child-holds-pipes regression")
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
