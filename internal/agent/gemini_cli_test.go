// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	run := Run{
		PromptText: "Hello Gemini",
	}
	args := driver.buildArgs(run)

	expectedArgs := []string{
		"--dangerously-skip-permissions",
		"--prompt",
		"Hello Gemini",
	}

	if len(args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d", len(expectedArgs), len(args))
	}
	for i, arg := range args {
		if arg != expectedArgs[i] {
			t.Errorf("arg %d: expected %q, got %q", i, expectedArgs[i], arg)
		}
	}
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
