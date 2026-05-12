// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/lock"
)

// ----- helpers ---------------------------------------------------------------

// makeEventsCh creates a buffered channel pre-loaded with the given progress events.
// Closing the channel simulates the process exiting after those events.
func makeEventsCh(events ...ProgressEvent) <-chan ProgressEvent {
	ch := make(chan ProgressEvent, len(events)+1)
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return ch
}

// initEvent builds a ProgressEvent that looks like a system/init event.
func initEvent(permissionMode string) ProgressEvent {
	ev := map[string]any{
		"type":    "system",
		"subtype": "init",
	}
	if permissionMode != "" {
		ev["permissionMode"] = permissionMode
	}
	b, _ := json.Marshal(ev)
	return ProgressEvent{Raw: string(b), Event: ev}
}

// nonInitEvent returns a plain result event (not an init event).
func nonInitEvent() ProgressEvent {
	ev := map[string]any{"type": "result", "subtype": "success"}
	b, _ := json.Marshal(ev)
	return ProgressEvent{Raw: string(b), Event: ev}
}

// ----- runPrecheck unit tests ------------------------------------------------

// TestPrecheck_BypassPasses verifies that a system/init event with
// permissionMode=bypassPermissions advances the state to precheckPassed
// and does not invoke the kill function.
func TestPrecheck_BypassPasses(t *testing.T) {
	killCalled := false
	state, mode := runPrecheck(
		makeEventsCh(initEvent("bypassPermissions")),
		5*time.Second,
		true,
		"run-test-1",
		nil,
		func() { killCalled = true },
	)
	if state != precheckPassed {
		t.Errorf("state = %v, want precheckPassed", state)
	}
	if mode != "bypassPermissions" {
		t.Errorf("observedMode = %q, want \"bypassPermissions\"", mode)
	}
	if killCalled {
		t.Error("kill was called unexpectedly")
	}
}

// TestPrecheck_DefaultFails verifies that an init event with permissionMode=default
// transitions to precheckFailedMode and calls the kill function.
func TestPrecheck_DefaultFails(t *testing.T) {
	killCalled := false
	state, mode := runPrecheck(
		makeEventsCh(initEvent("default")),
		5*time.Second,
		true,
		"run-test-2",
		nil,
		func() { killCalled = true },
	)
	if state != precheckFailedMode {
		t.Errorf("state = %v, want precheckFailedMode", state)
	}
	if mode != "default" {
		t.Errorf("observedMode = %q, want \"default\"", mode)
	}
	if !killCalled {
		t.Error("kill was not called for default permission mode")
	}
}

// TestPrecheck_AcceptEditsFails verifies that permissionMode=acceptEdits also fails.
func TestPrecheck_AcceptEditsFails(t *testing.T) {
	killCalled := false
	state, mode := runPrecheck(
		makeEventsCh(initEvent("acceptEdits")),
		5*time.Second,
		true,
		"run-test-3",
		nil,
		func() { killCalled = true },
	)
	if state != precheckFailedMode {
		t.Errorf("state = %v, want precheckFailedMode", state)
	}
	if mode != "acceptEdits" {
		t.Errorf("observedMode = %q, want \"acceptEdits\"", mode)
	}
	if !killCalled {
		t.Error("kill was not called for acceptEdits permission mode")
	}
}

// TestPrecheck_MissingFieldWarnsAndPasses verifies that an init event with no
// permissionMode field is treated as passed (with a warning log).
func TestPrecheck_MissingFieldWarnsAndPasses(t *testing.T) {
	killCalled := false
	state, _ := runPrecheck(
		makeEventsCh(initEvent("")), // empty string → field omitted from event
		5*time.Second,
		true,
		"run-test-4",
		nil,
		func() { killCalled = true },
	)
	if state != precheckPassed {
		t.Errorf("state = %v, want precheckPassed", state)
	}
	if killCalled {
		t.Error("kill was called unexpectedly for missing permissionMode")
	}
}

// TestPrecheck_Timeout verifies that when no init event is received within the
// timeout window, the state becomes precheckFailedTimeout and kill is called.
func TestPrecheck_Timeout(t *testing.T) {
	// Create a channel that never sends (simulates a process that never emits init).
	neverCh := make(chan ProgressEvent)

	killCalled := false
	done := make(chan struct{})
	var state precheckState
	go func() {
		defer close(done)
		state, _ = runPrecheck(
			neverCh,
			50*time.Millisecond, // very short timeout for testing
			true,
			"run-test-5",
			nil,
			func() { killCalled = true },
		)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runPrecheck did not return after timeout")
	}

	if state != precheckFailedTimeout {
		t.Errorf("state = %v, want precheckFailedTimeout", state)
	}
	if !killCalled {
		t.Error("kill was not called after timeout")
	}
}

// TestPrecheck_EscapeHatch verifies that when requireBypass=false, a non-bypass
// mode is accepted with a warning log and does NOT call kill.
func TestPrecheck_EscapeHatch(t *testing.T) {
	killCalled := false
	state, mode := runPrecheck(
		makeEventsCh(initEvent("default")),
		5*time.Second,
		false, // escape hatch: bypass not required
		"run-test-6",
		nil,
		func() { killCalled = true },
	)
	if state != precheckPassed {
		t.Errorf("state = %v, want precheckPassed", state)
	}
	if mode != "default" {
		t.Errorf("observedMode = %q, want \"default\"", mode)
	}
	if killCalled {
		t.Error("kill was called unexpectedly when escape hatch is enabled")
	}
}

// ----- killAndFail / log-line tests ------------------------------------------

// mockKiller records whether Kill was called and satisfies processKiller.
type mockKiller struct {
	killed int32 // atomic
}

func (mk *mockKiller) Kill(_ Process) {
	atomic.AddInt32(&mk.killed, 1)
}

// mockProcess is a minimal Process implementation for tests.
type mockProcess struct {
	progress chan ProgressEvent
	waitErr  error
}

func newMockProcess(events ...ProgressEvent) *mockProcess {
	ch := make(chan ProgressEvent, len(events)+1)
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return &mockProcess{progress: ch}
}

func (mp *mockProcess) Wait() error                    { return mp.waitErr }
func (mp *mockProcess) Kill() error                    { return nil }
func (mp *mockProcess) Progress() <-chan ProgressEvent { return mp.progress }
func (mp *mockProcess) StderrTail() string             { return "" }

// newTestManager builds a minimal Manager backed by a temp SQLite index for use
// in unit tests that exercise killAndFail and related methods.
func newTestManager(t *testing.T) (*Manager, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")
	idx, err := index.Open(dbPath, dir, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	h := hub.New()
	locks := lock.New(idx, h)
	mk := &mockKiller{}
	m := &Manager{
		agents:             nil,
		drivers:            map[string]Driver{},
		sem:                make(chan struct{}, 4),
		active:             make(map[string]*activeRun),
		idx:                idx,
		hub:                h,
		locks:              locks,
		initEventTimeout:   5 * time.Second,
		requireBypassPerms: true,
		killer:             mk,
		logsDir:            dir,
	}
	// Fill semaphore slot (simulates a run acquired the semaphore).
	m.sem <- struct{}{}
	cleanup := func() { _ = idx.Close() }
	return m, cleanup
}

// TestPrecheck_LogLineAppended verifies that killAndFail appends a
// precheck_failure JSON line to the run's on-disk log file.
func TestPrecheck_LogLineAppended(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	runID := "logtest-run-01"
	logPath := filepath.Join(t.TempDir(), runID+".log")

	// Insert a run record so UpdateAgentRun can find it.
	now := time.Now()
	row := &index.AgentRunRow{
		RunID:     runID,
		AgentName: "test-agent",
		Role:      "backend-developer",
		StartedAt: now,
		Status:    "running",
	}
	if err := m.idx.InsertAgentRun(row); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	// Pre-populate m.active so killAndFail can remove the entry.
	proc := newMockProcess()
	m.mu.Lock()
	m.active[runID] = &activeRun{proc: proc, cancel: func() {}}
	m.mu.Unlock()

	// Acquire the lineage lock so killAndFail can release it.
	lineage := "test-lineage"
	if _, err := m.locks.Acquire(lineage, runID, "agent"); err != nil {
		t.Fatalf("Acquire lock: %v", err)
	}

	run := Run{
		RunID:     runID,
		AgentName: "test-agent",
		LogPath:   logPath,
	}

	m.killAndFail(run, row, proc, lineage, precheckFailedMode, "default")

	// Read and verify the log file.
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	logContent := string(data)
	if !strings.Contains(logContent, `"type":"precheck_failure"`) {
		t.Errorf("log file missing precheck_failure type, got: %s", logContent)
	}
	if !strings.Contains(logContent, `"reason":"permission_mode_default"`) {
		t.Errorf("log file missing reason field, got: %s", logContent)
	}
	if !strings.Contains(logContent, `"observed_permission_mode":"default"`) {
		t.Errorf("log file missing observed_permission_mode, got: %s", logContent)
	}
	if !strings.Contains(logContent, `"run_id":"logtest-run-01"`) {
		t.Errorf("log file missing run_id, got: %s", logContent)
	}
}

// ----- integration: Manager precheck with AppAgentConfig ---------------------

// TestPrecheck_ConfigRoundTrip verifies that init_event_timeout_seconds from
// AppAgentConfig is applied by the Manager. This test confirms:
//  1. The timeout field is correctly propagated from AppAgentConfig to the Manager.
//  2. runPrecheck uses the configured timeout (2 s here) rather than the default 10 s.
func TestPrecheck_ConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")
	idx, err := index.Open(dbPath, dir, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	defer idx.Close()

	h := hub.New()
	locks := lock.New(idx, h)

	// Use a 2-second timeout — short enough to fire well before the default 10 s
	// but long enough to avoid flakiness on loaded machines.
	timeoutSecs := 2
	requireBypass := true
	agentCfg := config.AppAgentConfig{
		InitEventTimeoutSeconds:  timeoutSecs,
		RequireBypassPermissions: &requireBypass,
	}

	m := New(nil, 4, idx, nil, h, locks, nil, dir, dir, nil, agentCfg)

	// 1. Confirm the timeout was propagated.
	wantTimeout := time.Duration(timeoutSecs) * time.Second
	if m.initEventTimeout != wantTimeout {
		t.Errorf("initEventTimeout = %v, want %v", m.initEventTimeout, wantTimeout)
	}
	if !m.requireBypassPerms {
		t.Error("requireBypassPerms should be true")
	}

	// 2. Run runPrecheck against a channel that never emits init; verify it fires
	//    at ~2 s (not the default 10 s).
	var killCount int32
	neverCh := make(chan ProgressEvent)
	start := time.Now()
	done := make(chan struct{})
	var result precheckState
	go func() {
		defer close(done)
		result, _ = runPrecheck(
			neverCh,
			m.initEventTimeout,
			m.requireBypassPerms,
			"config-roundtrip-run",
			nil,
			func() { atomic.AddInt32(&killCount, 1) },
		)
	}()

	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("runPrecheck did not return within 8 s")
	}
	elapsed := time.Since(start)

	if result != precheckFailedTimeout {
		t.Errorf("result = %v, want precheckFailedTimeout", result)
	}
	if atomic.LoadInt32(&killCount) == 0 {
		t.Error("kill was not called")
	}
	// Should fire at ~2 s, definitely not at 10 s.
	if elapsed >= 8*time.Second {
		t.Errorf("timeout fired too late: elapsed = %v (configured %d s)", elapsed, timeoutSecs)
	}
	t.Logf("elapsed: %v (configured %d s)", elapsed, timeoutSecs)
}
