// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// setupFakeClaudeWithOutput writes a stub `claude` shell script that emits
// custom NDJSON output before exiting 0. The script is prepended to PATH.
func setupFakeClaudeWithOutput(t *testing.T, ndjsonOutput string) {
	t.Helper()
	fakeDir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' '%s'\nexit 0\n", ndjsonOutput)
	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}

// setupFakeClaudeWithRawScript writes a multi-line shell script for the fake
// claude binary. The content is written to a temp file and set in PATH.
func setupFakeClaudeWithRawScript(t *testing.T, script string) {
	t.Helper()
	fakeDir := t.TempDir()
	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}

const ndjsonAssistantEvent = `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`
const ndjsonResultLine = `{"type":"result","subtype":"success","total_cost_usd":0.01,"duration_ms":1000,"duration_api_ms":900,"num_turns":1,"usage":{"input_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":50,"output_tokens":200}}`

// TestSupervisor_PersistsMetricsOnFinish verifies that when a fake claude
// emits a valid result line, the supervisor writes metrics to the index after
// the run completes.
func TestSupervisor_PersistsMetricsOnFinish(t *testing.T) {
	// Emit assistant event + result line.
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' '%s'\nprintf '%%s\\n' '%s'\nexit 0\n",
		ndjsonAssistantEvent, ndjsonResultLine)
	setupFakeClaudeWithRawScript(t, script)

	const artifactPath = "lifecycle/ideas/metrics-persist.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Metrics Persist Test", "idea", "draft", "metrics-persist", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)
	waitForRunCompletion(t, env, runID)

	row, err := env.proj.Idx.GetAgentRun(runID)
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if row == nil {
		t.Fatal("GetAgentRun returned nil")
	}
	if row.MetricsAvailable != 1 {
		t.Errorf("MetricsAvailable: got %d, want 1", row.MetricsAvailable)
	}
	if row.TotalCostUSD == nil {
		t.Error("TotalCostUSD should be non-nil after run with result line")
	}
	if row.DurationApiMs == nil {
		t.Error("DurationApiMs should be non-nil after run with result line")
	}
	if row.InputTokens == nil {
		t.Error("InputTokens should be non-nil after run with result line")
	}
	if row.OutputTokens == nil {
		t.Error("OutputTokens should be non-nil after run with result line")
	}
}

// TestSupervisor_NonClaudeRun_NoMetrics verifies that when a fake claude
// emits no NDJSON (exits 0 silently), metrics_available remains 0.
func TestSupervisor_NonClaudeRun_NoMetrics(t *testing.T) {
	// Plain exit 0 — no NDJSON output (no result line → no metrics).
	setupFakeClaudeSilent(t, 0)

	const artifactPath = "lifecycle/ideas/no-metrics.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("No Metrics Test", "idea", "draft", "no-metrics", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)
	waitForRunCompletion(t, env, runID)

	row, err := env.proj.Idx.GetAgentRun(runID)
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if row == nil {
		t.Fatal("GetAgentRun returned nil")
	}
	if row.MetricsAvailable != 0 {
		t.Errorf("MetricsAvailable: got %d, want 0 (no result line emitted)", row.MetricsAvailable)
	}
}

// TestSupervisor_RecordsTTFT verifies that the supervisor records a TTFT value
// when the first assistant event arrives after a short delay.
func TestSupervisor_RecordsTTFT(t *testing.T) {
	// Sleep ~120ms before emitting the first assistant event, then the result.
	script := fmt.Sprintf("#!/bin/sh\nsleep 0.12\nprintf '%%s\\n' '%s'\nprintf '%%s\\n' '%s'\nexit 0\n",
		ndjsonAssistantEvent, ndjsonResultLine)
	setupFakeClaudeWithRawScript(t, script)

	const artifactPath = "lifecycle/ideas/ttft-test.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TTFT Test", "idea", "draft", "ttft-test", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)
	waitForRunCompletion(t, env, runID)

	row, err := env.proj.Idx.GetAgentRun(runID)
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if row == nil {
		t.Fatal("GetAgentRun returned nil")
	}
	if row.TtftMs == nil {
		t.Fatal("TtftMs should be non-nil after run with assistant event")
	}
	// Generous bounds for CI: sleep 0.12 → expect 80ms ≤ ttft ≤ 500ms.
	ttft := *row.TtftMs
	if ttft < 80 || ttft > 500 {
		t.Errorf("TtftMs: got %d ms, expected in range [80, 500]", ttft)
	}
}

// TestSupervisor_RecordsTTFTOnce verifies that the supervisor records TTFT
// from the first assistant event only (firstTokenSeen guard).
func TestSupervisor_RecordsTTFTOnce(t *testing.T) {
	// Emit TWO assistant events; TTFT should be set from the first one.
	script := fmt.Sprintf(
		"#!/bin/sh\nprintf '%%s\\n' '%s'\nprintf '%%s\\n' '%s'\nprintf '%%s\\n' '%s'\nexit 0\n",
		ndjsonAssistantEvent, ndjsonAssistantEvent, ndjsonResultLine)
	setupFakeClaudeWithRawScript(t, script)

	const artifactPath = "lifecycle/ideas/ttft-once.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TTFT Once Test", "idea", "draft", "ttft-once", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)
	waitForRunCompletion(t, env, runID)

	row, err := env.proj.Idx.GetAgentRun(runID)
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if row == nil {
		t.Fatal("GetAgentRun returned nil")
	}
	if row.TtftMs == nil {
		t.Fatal("TtftMs should be non-nil after run with assistant event")
	}
	if *row.TtftMs <= 0 {
		t.Errorf("TtftMs should be > 0, got %d", *row.TtftMs)
	}
}
