// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
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
	if row.CacheCreationTokens == nil {
		t.Error("CacheCreationTokens should be non-nil after run with result line")
	}
	if row.CacheReadTokens == nil {
		t.Error("CacheReadTokens should be non-nil after run with result line")
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
	// Generous bounds for CI: sleep 0.12 → expect 80ms ≤ ttft ≤ 2000ms.
	ttft := *row.TtftMs
	if ttft < 80 || ttft > 2000 {
		t.Errorf("TtftMs: got %d ms, expected in range [80, 2000]", ttft)
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

// TestSupervisor_RecordsTTFTUnderLoad verifies that TTFT is still recorded as
// a sane, positive value while the test process is under real CPU contention
// (busy-spinning goroutines on every core, matching the resource-contention
// scenario from lifecycle/defects/agent-usage-analytics-report-10-defect.md).
// Unlike TestSupervisor_RecordsTTFT, it deliberately asserts no upper bound:
// scheduling delays under contention are expected to inflate TTFT, so a
// strict ceiling would itself be flaky.
func TestSupervisor_RecordsTTFTUnderLoad(t *testing.T) {
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
			}
		}()
	}
	defer func() {
		close(stop)
		wg.Wait()
	}()

	script := fmt.Sprintf("#!/bin/sh\nsleep 0.12\nprintf '%%s\\n' '%s'\nprintf '%%s\\n' '%s'\nexit 0\n",
		ndjsonAssistantEvent, ndjsonResultLine)
	setupFakeClaudeWithRawScript(t, script)

	const artifactPath = "lifecycle/ideas/ttft-under-load.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TTFT Under Load Test", "idea", "draft", "ttft-under-load", "", "Body."),
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
		t.Errorf("TtftMs should be > 0 under load, got %d", *row.TtftMs)
	}
}

// TestReportsAgentUsage_AggregatedTTFS verifies that the analytics report
// correctly calculates mean and P95 TTFT from a series of runs.
func TestReportsAgentUsage_AggregatedTTFS(t *testing.T) {
	env := newAgentTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	now := time.Now()
	// We'll add 5 runs with TTFT values: 100, 200, 300, 400, 500 ms.
	ttftValues := []int64{100, 200, 300, 400, 500}
	for i, ttft := range ttftValues {
		runID := fmt.Sprintf("agg-ttft-%d", i)
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' '%s'\nprintf '%%s\\n' '%s'\nexit 0\n",
			ndjsonAssistantEvent, ndjsonResultLineWithTTFT(ttft))
		setupFakeClaudeWithRawScript(t, script)

		env.proj.Idx.InsertAgentRun(&index.AgentRunRow{
			RunID:      runID,
			AgentName:  "qa",
			Role:       "analyst",
			Status:     "running",
			StartedAt:  now.Add(-time.Duration(i+1) * time.Hour),
			TargetPath: fmt.Sprintf("lifecycle/ideas/agg-test-%d.md", i),
		})

		// We need to simulate the driver finishing and emitting results.
		finishedAt := now.Add(-time.Duration(i) * time.Hour)
		m := index.AgentRunMetrics{
			TotalCostUSD:   0.1,
			DurationApiMs:  100,
			InputTokens:    100,
			OutputTokens:   100,
		}
		err := env.proj.Idx.UpdateAgentRunMetrics(runID, m)
		if err != nil {
			t.Fatalf("failed to update metrics: %v", err)
		}

		// TTFT is persisted via a dedicated setter (not part of
		// AgentRunMetrics); set it so the report can aggregate it.
		if err := env.proj.Idx.SetAgentRunTTFT(runID, ttft); err != nil {
			t.Fatalf("failed to set ttft: %v", err)
		}

		// Mark as done manually for the run
		var finishedAtTime time.Time = finishedAt
		err = env.proj.Idx.UpdateAgentRun(&index.AgentRunRow{
			RunID:      runID,
			Status:     "done",
			FinishedAt: &finishedAtTime,
		})
		if err != nil {
			t.Fatalf("failed to update run status: %v", err)
		}
	}

	resp := env.doRequest("GET", "/api/p/testproject/reports/agent-usage", nil)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	data := readJSON(t, resp)

	summary, _ := data["summary"].(map[string]any)
	overall, _ := summary["overall"].(map[string]any)
	meanTTFT, ok := overall["mean_ttft_ms"].(float64)
	if !ok {
		t.Fatalf("missing mean_ttft_ms in summary: %v", overall)
	}
	if fmt.Sprintf("%.0f", meanTTFT) != "300" {
		t.Errorf("mean_ttft: got %v, want 300", meanTTFT)
	}

	p95TTFT, ok := overall["p95_ttft_ms"].(float64)
	if !ok {
		t.Fatalf("missing p95_ttft_ms in summary: %v", overall)
	}
	if fmt.Sprintf("%.0f", p95TTFT) != "500" {
		t.Errorf("p95_ttft: got %v, want 500", p95TTFT)
	}
}

func ndjsonResultLineWithTTFT(ttft int64) string {
	return fmt.Sprintf(`{"type":"result","subtype":"success","total_cost_usd":0.01,"duration_ms":1000,"duration_api_ms":900,"num_turns":1,"usage":{"input_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":50,"output_tokens":200},"ttft_ms":%d}`, ttft)
}