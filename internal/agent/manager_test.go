// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/lock"
)

// newMinimalManager creates a Manager backed by a real SQLite index for use in
// driver-map and semaphore tests. The returned cleanup must be called at test end.
func newMinimalManager(t *testing.T, agents []config.AgentConfig, maxConcurrent int) (*Manager, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	h := hub.New()
	idx, err := index.Open(filepath.Join(tmpDir, "mgr-test.db"), tmpDir, nil,
		index.WithHub(h),
	)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	locks := lock.New(idx, h)
	mgr := New(agents, maxConcurrent, idx, nil, h, locks, nil, tmpDir, "", nil, config.AppAgentConfig{})
	return mgr, func() { idx.Close() }
}

// ---------------------------------------------------------------------------
// T-6: Driver-map wiring
// ---------------------------------------------------------------------------

// TestManager_DriverMapComplete verifies that every expected driver is present
// in the manager's driver map after New() (T-6, T-7 regression guard).
func TestManager_DriverMapComplete(t *testing.T) {
	mgr, cleanup := newMinimalManager(t, nil, 4)
	defer cleanup()

	required := []string{
		"claude-code-cli", "claude-mediated", "claude-env",
		"codex-cli", "ollama", "gemini", "gemini-cli", "shell-stub",
	}
	for _, name := range required {
		if _, ok := mgr.drivers[name]; !ok {
			t.Errorf("driver %q missing from driver map", name)
		}
	}
}

// TestManager_ClaudeEnvDriverWired verifies that StartRun with a claude-env
// agent does not return an "unknown driver" error (the driver is wired).
func TestManager_ClaudeEnvDriverWired(t *testing.T) {
	agents := []config.AgentConfig{
		{
			Name:      "env-agent",
			Roles:     []string{"analyst"},
			Driver:    "claude-env",
			BaseURL:   "http://localhost:11434",
			AuthToken: "test-token",
			Model:     "claude-opus-4-6",
			PromptTemplates: map[string]string{
				"analyst": "test prompt {target_path}",
			},
		},
	}
	mgr, cleanup := newMinimalManager(t, agents, 4)
	defer cleanup()

	_, err := mgr.StartRun(context.Background(), "env-agent", "lifecycle/ideas/test.md", "analyst", nil)
	if err != nil && strings.Contains(err.Error(), "unknown driver") {
		t.Errorf("StartRun returned 'unknown driver' for claude-env: %v", err)
	}
	// The run may fail for other reasons (e.g. claude binary absent), which is fine.
}

// TestManager_UnknownDriverReturnsError verifies that an agent referencing a
// truly unknown driver returns an error containing "unknown driver" (T-6).
func TestManager_UnknownDriverReturnsError(t *testing.T) {
	agents := []config.AgentConfig{
		{
			Name:   "mystery-agent",
			Roles:  []string{"analyst"},
			Driver: "truly-unknown-driver",
			PromptTemplates: map[string]string{
				"analyst": "test prompt",
			},
		},
	}
	mgr, cleanup := newMinimalManager(t, agents, 4)
	defer cleanup()

	_, err := mgr.StartRun(context.Background(), "mystery-agent", "", "analyst", nil)
	if err == nil {
		t.Fatal("expected error for unknown driver, got nil")
	}
	if !strings.Contains(err.Error(), "unknown driver") {
		t.Errorf("expected 'unknown driver' in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// T-6: Semaphore behaviour
// ---------------------------------------------------------------------------

// stubAgent returns an AgentConfig for the shell-stub driver with an optional
// shell command. An empty shellCmd uses the stub's default (quick result event).
func stubAgent(name, shellCmd string) config.AgentConfig {
	return config.AgentConfig{
		Name:         name,
		Roles:        []string{"analyst"},
		Driver:       "shell-stub",
		ShellCommand: shellCmd,
		PromptTemplates: map[string]string{
			"analyst": "test",
		},
	}
}

// waitForRunStatus polls GetRun until the run's status differs from "running"
// or the deadline is reached. Returns true when the status changed.
func waitForRunStatus(mgr *Manager, runID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		row, err := mgr.GetRun(runID)
		if err == nil && row != nil && row.Status != "running" {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// TestManager_SemaphoreErrBusy verifies that when max_concurrent_agents=1 is
// reached, the next StartRun returns ErrBusy (T-6).
func TestManager_SemaphoreErrBusy(t *testing.T) {
	mgr, cleanup := newMinimalManager(t, []config.AgentConfig{
		stubAgent("stub", "sleep 60"),
	}, 1)
	defer cleanup()

	ctx := context.Background()

	// Run 1: long-running, holds the semaphore.
	runID1, err := mgr.StartRun(ctx, "stub", "target/one.md", "analyst", nil)
	if err != nil {
		t.Fatalf("StartRun run1: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.Kill(runID1)
		waitForRunStatus(mgr, runID1, 3*time.Second)
	})

	// Run 2: semaphore is full → ErrBusy.
	_, err2 := mgr.StartRun(ctx, "stub", "target/two.md", "analyst", nil)
	if err2 != ErrBusy {
		t.Errorf("expected ErrBusy when semaphore is full, got: %v", err2)
	}
}

// TestManager_SemaphoreReleasedAfterRun verifies that the semaphore slot is
// released once a run finishes (T-6).
func TestManager_SemaphoreReleasedAfterRun(t *testing.T) {
	mgr, cleanup := newMinimalManager(t, []config.AgentConfig{
		stubAgent("stub", ""), // default: quick exit
	}, 1)
	defer cleanup()

	ctx := context.Background()

	// Run 1: quick exit.
	runID1, err := mgr.StartRun(ctx, "stub", "target/one.md", "analyst", nil)
	if err != nil {
		t.Fatalf("StartRun run1: %v", err)
	}

	// Wait for run1 to complete and release the semaphore.
	if !waitForRunStatus(mgr, runID1, 5*time.Second) {
		t.Fatal("run1 did not complete within 5s")
	}

	// Run 2: semaphore should be free — must not get ErrBusy.
	runID2, err2 := mgr.StartRun(ctx, "stub", "target/two.md", "analyst", nil)
	if err2 == ErrBusy {
		t.Error("semaphore not released after run1 completed")
	}
	// Clean up run2 if it started successfully.
	if err2 == nil {
		t.Cleanup(func() {
			_ = mgr.Kill(runID2)
			waitForRunStatus(mgr, runID2, 3*time.Second)
		})
	}
}
