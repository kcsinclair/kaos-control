// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOperations_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	ops, err := LoadOperations(dir)
	if err != nil {
		t.Fatalf("LoadOperations (missing file): %v", err)
	}

	if err := ops.SetAgentState(AgentOperationalState{
		Agent:        "requirements-analyst",
		Primary:      ProviderModel{Provider: "anthropic-cloud", Model: "claude-3-7-sonnet"},
		Active:       ProviderModel{Provider: "gemini-cloud", Model: "gemini-2.5-flash"},
		SwitchedAt:   1000,
		Reason:       "HTTP 529 Overloaded",
		ResetsAtUnix: 2000,
		Bucket:       "hourly",
	}); err != nil {
		t.Fatalf("SetAgentState: %v", err)
	}
	if err := ops.SetReachability("anthropic-cloud", false, time.Unix(1500, 0)); err != nil {
		t.Fatalf("SetReachability: %v", err)
	}
	if _, err := ops.RecordDisconnect("gemini-cloud", time.Unix(1600, 0), 30*time.Second); err != nil {
		t.Fatalf("RecordDisconnect: %v", err)
	}
	if err := ops.AppendHistory(FailoverHistoryEntry{
		At: 1700, Agent: "requirements-analyst", Action: "failover",
		FromProvider: "anthropic-cloud", ToProvider: "gemini-cloud", Reason: "HTTP 529 Overloaded",
	}); err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}

	reloaded, err := LoadOperations(dir)
	if err != nil {
		t.Fatalf("LoadOperations (reload): %v", err)
	}

	state, ok := reloaded.AgentState("requirements-analyst")
	if !ok {
		t.Fatal("expected agent state to round-trip")
	}
	if state.Primary.Provider != "anthropic-cloud" || state.Active.Provider != "gemini-cloud" {
		t.Errorf("agent state mismatch after reload: %+v", state)
	}
	if !state.IsFailedOver() {
		t.Error("expected IsFailedOver() true")
	}

	reach, ok := reloaded.GetReachability("anthropic-cloud")
	if !ok || reach.Healthy {
		t.Errorf("reachability mismatch after reload: %+v ok=%v", reach, ok)
	}

	if got := reloaded.DisconnectCountSince("gemini-cloud", time.Unix(0, 0)); got != 1 {
		t.Errorf("DisconnectCountSince: got %d, want 1", got)
	}

	hist := reloaded.HistorySnapshot()
	if len(hist) != 1 || hist[0].Action != "failover" {
		t.Errorf("history mismatch after reload: %+v", hist)
	}
}

func TestOperations_ClearAgentState(t *testing.T) {
	dir := t.TempDir()
	ops, err := LoadOperations(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ops.SetAgentState(AgentOperationalState{Agent: "a1", Primary: ProviderModel{Provider: "p1"}, Active: ProviderModel{Provider: "p2"}}); err != nil {
		t.Fatal(err)
	}
	if err := ops.ClearAgentState("a1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := ops.AgentState("a1"); ok {
		t.Error("expected agent state cleared")
	}

	reloaded, err := LoadOperations(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reloaded.AgentState("a1"); ok {
		t.Error("expected cleared agent state to persist across reload")
	}
}

func TestOperations_RecordDisconnect_CollapsesWithinWindow(t *testing.T) {
	dir := t.TempDir()
	ops, err := LoadOperations(dir)
	if err != nil {
		t.Fatal(err)
	}

	window := 30 * time.Second
	recorded, err := ops.RecordDisconnect("p1", time.Unix(1000, 0), window)
	if err != nil || !recorded {
		t.Fatalf("first disconnect should record: recorded=%v err=%v", recorded, err)
	}
	// Inside the collapse window — must not count as a second occurrence.
	recorded, err = ops.RecordDisconnect("p1", time.Unix(1010, 0), window)
	if err != nil || recorded {
		t.Fatalf("disconnect inside window should collapse: recorded=%v err=%v", recorded, err)
	}
	// Outside the window — must count.
	recorded, err = ops.RecordDisconnect("p1", time.Unix(1200, 0), window)
	if err != nil || !recorded {
		t.Fatalf("disconnect outside window should record: recorded=%v err=%v", recorded, err)
	}

	if got := ops.DisconnectCountSince("p1", time.Unix(0, 0)); got != 2 {
		t.Errorf("DisconnectCountSince: got %d, want 2", got)
	}
}

func TestOperations_AtomicWrite_CrashLeavesPreviousFileIntact(t *testing.T) {
	dir := t.TempDir()
	ops, err := LoadOperations(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ops.SetAgentState(AgentOperationalState{Agent: "a1", Active: ProviderModel{Provider: "p1"}}); err != nil {
		t.Fatal(err)
	}

	path := operationsPath(dir)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a crash between temp-write and rename: leave a stray, possibly
	// truncated temp file next to the real one and confirm Load still yields
	// the last successfully-renamed content.
	if err := os.WriteFile(path+".tmp", []byte("truncated garbage that would not"), 0o600); err != nil {
		t.Fatal(err)
	}

	reloaded, err := LoadOperations(dir)
	if err != nil {
		t.Fatalf("Load after simulated crash: %v", err)
	}
	if _, ok := reloaded.AgentState("a1"); !ok {
		t.Error("expected previous operations.yaml to remain intact and parseable")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("real operations.yaml should be untouched by the stray .tmp file")
	}
}

func TestOperations_Save_ContainsNoAPIKeys(t *testing.T) {
	dir := t.TempDir()
	ops, err := LoadOperations(dir)
	if err != nil {
		t.Fatal(err)
	}
	const fakeAPIKey = "sk-ant-TESTSECRET-do-not-leak-12345"
	if err := ops.SetAgentState(AgentOperationalState{
		Agent:   "requirements-analyst",
		Primary: ProviderModel{Provider: "anthropic-cloud", Model: "claude-3-7-sonnet"},
		Active:  ProviderModel{Provider: "gemini-cloud", Model: "gemini-2.5-flash"},
		Reason:  "HTTP 529 Overloaded",
	}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(operationsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), fakeAPIKey) {
		t.Error("operations.yaml must never contain API key material")
	}
}

func TestOperations_NeverAppearsUntrackedInGit(t *testing.T) {
	// operations.yaml is listed in the repo-root .gitignore; assert that
	// entry is present so a project checkout never accidentally tracks it.
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "operations.yaml" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected .gitignore to list operations.yaml")
	}
}
