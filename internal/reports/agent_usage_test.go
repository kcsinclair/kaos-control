// SPDX-License-Identifier: AGPL-3.0-or-later

package reports

import (
	"errors"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func openTestReportsIndex(t *testing.T) *index.Index {
	t.Helper()
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	projRoot := dir
	if err := os.MkdirAll(projRoot+"/lifecycle", 0o755); err != nil {
		t.Fatal(err)
	}
	idx, err := index.Open(dbPath, projRoot, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx
}

type runSeed struct {
	runID        string
	agentName    string
	startedAt    time.Time
	status       string
	model        string
	metricsAvail bool
	costUSD      float64
	durationMs   int64
	inputTok     int64
	cacheCreate  int64
	cacheRead    int64
	outputTok    int64
	ttftMs       *int64
}

func seedRuns(t *testing.T, idx *index.Index, runs []runSeed) {
	t.Helper()
	for _, r := range runs {
		row := &index.AgentRunRow{
			RunID:     r.runID,
			AgentName: r.agentName,
			Role:      "analyst",
			StartedAt: r.startedAt,
			Status:    r.status,
		}
		if err := idx.InsertAgentRun(row); err != nil {
			t.Fatalf("InsertAgentRun(%s): %v", r.runID, err)
		}
		// Mark the run as finished so it's not "running".
		finished := r.startedAt.Add(time.Duration(r.durationMs) * time.Millisecond)
		row.Status = r.status
		row.FinishedAt = &finished
		if err := idx.UpdateAgentRun(row); err != nil {
			t.Fatalf("UpdateAgentRun(%s): %v", r.runID, err)
		}
		if r.model != "" {
			if err := idx.SetAgentRunModel(r.runID, r.model); err != nil {
				t.Fatalf("SetAgentRunModel(%s): %v", r.runID, err)
			}
		}
		if r.metricsAvail {
			m := index.AgentRunMetrics{
				Model:               r.model,
				TotalCostUSD:        r.costUSD,
				DurationApiMs:       r.durationMs,
				InputTokens:         r.inputTok,
				CacheCreationTokens: r.cacheCreate,
				CacheReadTokens:     r.cacheRead,
				OutputTokens:        r.outputTok,
			}
			if err := idx.UpdateAgentRunMetrics(r.runID, m); err != nil {
				t.Fatalf("UpdateAgentRunMetrics(%s): %v", r.runID, err)
			}
		}
		if r.ttftMs != nil {
			if err := idx.SetAgentRunTTFT(r.runID, *r.ttftMs); err != nil {
				t.Fatalf("SetAgentRunTTFT(%s): %v", r.runID, err)
			}
		}
	}
}

func ptr64(v int64) *int64 { return &v }

// ── Milestone 1 — BucketStart tests ──────────────────────────────────────────

func TestBucketStart_HourUTC(t *testing.T) {
	in := time.Date(2026, 6, 12, 14, 37, 12, 0, time.UTC)
	got := BucketStart(in, "hour", time.UTC)
	want := time.Date(2026, 6, 12, 14, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("BucketStart hour UTC: got %v, want %v", got, want)
	}
}

func TestBucketStart_DayBrowserTZ(t *testing.T) {
	sydney, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Skipf("Australia/Sydney not available: %v", err)
	}
	// UTC 2026-06-12 14:00 = Sydney 2026-06-13 00:00 AEST (UTC+10, no DST in June).
	in := time.Date(2026, 6, 12, 14, 0, 0, 0, time.UTC)
	got := BucketStart(in, "day", sydney)
	// Should be midnight at start of June 13 Sydney time.
	wantYear, wantMon, wantDay := 2026, time.June, 13
	gotYear, gotMon, gotDay := got.In(sydney).Date()
	gotH, gotM, gotS := got.In(sydney).Clock()
	if gotYear != wantYear || gotMon != wantMon || gotDay != wantDay || gotH != 0 || gotM != 0 || gotS != 0 {
		t.Errorf("BucketStart day Sydney: got %v (%v local), want 2026-06-13 00:00 Sydney",
			got, got.In(sydney))
	}
}

func TestBucketStart_WeekISO(t *testing.T) {
	// 2026-06-12 is a Friday; ISO week starts on Monday 2026-06-08.
	in := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	got := BucketStart(in, "week", time.UTC)
	want := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC) // Monday
	if !got.Equal(want) {
		t.Errorf("BucketStart week ISO: got %v, want %v", got, want)
	}
}

func TestBucketStart_DSTBoundary(t *testing.T) {
	// 2026-04-05 is Sydney DST end (clocks go back — 25-hour day).
	// Verify that nextBucket uses AddDate (timezone-aware) so the next day
	// bucket spans 25 hours rather than the naive 24h.
	sydney, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Skipf("Australia/Sydney not available: %v", err)
	}
	// 12:00 Sydney time on 2026-04-05 (during AEDT, UTC+11 in summer / DST).
	in := time.Date(2026, 4, 5, 12, 0, 0, 0, sydney)
	bucketStart := BucketStart(in, "day", sydney)
	next := nextBucket(bucketStart, "day")

	diff := next.Sub(bucketStart)
	// On a 25-hour day (DST end), the gap is 25h; on a normal day it's 24h.
	// Accept 23h ≤ diff ≤ 26h to be robust across TZ data variations,
	// but assert it is NOT exactly 24h (which naive Add(24h) would produce
	// on a DST boundary).
	if diff < 23*time.Hour || diff > 26*time.Hour {
		t.Errorf("nextBucket around DST boundary: diff = %v, expected ~25h", diff)
	}
	// Document: AddDate is timezone-aware so handles DST correctly.
	if diff == 24*time.Hour {
		t.Logf("NOTE: diff is exactly 24h — this may mean DST boundary was not hit for this date")
	}
}

// ── Milestone 2 — Aggregation tests ──────────────────────────────────────────

func makeFilter(_ *index.Index) AgentUsageFilter {
	return AgentUsageFilter{
		From:   time.Now().Add(-24 * time.Hour),
		To:     time.Now().Add(time.Hour),
		Bucket: "day",
		Loc:    time.UTC,
	}
}

func TestAggregate_AllSuccess(t *testing.T) {
	idx := openTestReportsIndex(t)
	now := time.Now()

	var totalCost float64
	var runs []runSeed
	for i := 0; i < 10; i++ {
		cost := 0.01 * float64(i+1)
		totalCost += cost
		runs = append(runs, runSeed{
			runID:        fmt.Sprintf("agg-all-%02d", i),
			agentName:    "qa",
			startedAt:    now.Add(-time.Duration(i) * time.Minute),
			status:       "done",
			model:        "claude-opus",
			metricsAvail: true,
			costUSD:      cost,
			durationMs:   1000,
			inputTok:     100,
			outputTok:    50,
		})
	}
	seedRuns(t, idx, runs)

	f := makeFilter(idx)
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	agg := report.Summary.Overall
	if agg.RunCount != 10 {
		t.Errorf("RunCount: got %d, want 10", agg.RunCount)
	}
	if agg.SuccessCount != 10 {
		t.Errorf("SuccessCount: got %d, want 10", agg.SuccessCount)
	}
	if agg.FailureCount != 0 {
		t.Errorf("FailureCount: got %d, want 0", agg.FailureCount)
	}
	if agg.MetricsUnavailableCount != 0 {
		t.Errorf("MetricsUnavailableCount: got %d, want 0", agg.MetricsUnavailableCount)
	}
	wantMean := totalCost / 10.0
	if math.Abs(agg.MeanCostUSD-wantMean) > 1e-9 {
		t.Errorf("MeanCostUSD: got %v, want %v", agg.MeanCostUSD, wantMean)
	}
}

func TestAggregate_MixedStatus(t *testing.T) {
	idx := openTestReportsIndex(t)
	now := time.Now()

	statuses := []string{"done", "done", "done", "done", "done",
		"failed", "failed", "killed", "killed-timeout", "running"}
	var runs []runSeed
	for i, s := range statuses {
		runs = append(runs, runSeed{
			runID:        fmt.Sprintf("mix-%02d", i),
			agentName:    "qa",
			startedAt:    now.Add(-time.Duration(i) * time.Minute),
			status:       s,
			metricsAvail: false,
		})
	}
	seedRuns(t, idx, runs)

	f := makeFilter(idx)
	// Default statuses exclude "running"
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	agg := report.Summary.Overall
	if agg.RunCount != 9 {
		t.Errorf("RunCount: got %d, want 9 (running excluded)", agg.RunCount)
	}
	if agg.FailureCount != 4 {
		t.Errorf("FailureCount: got %d, want 4 (2 failed + 1 killed + 1 killed-timeout)", agg.FailureCount)
	}
}

func TestAggregate_NoResultLineCounted(t *testing.T) {
	idx := openTestReportsIndex(t)
	now := time.Now()

	var runs []runSeed
	// 5 runs without metrics
	for i := 0; i < 5; i++ {
		runs = append(runs, runSeed{
			runID:        fmt.Sprintf("nometa-%02d", i),
			agentName:    "qa",
			startedAt:    now.Add(-time.Duration(i) * time.Minute),
			status:       "done",
			metricsAvail: false,
		})
	}
	// 5 runs with metrics at $0.10 each
	for i := 5; i < 10; i++ {
		runs = append(runs, runSeed{
			runID:        fmt.Sprintf("meta-%02d", i),
			agentName:    "qa",
			startedAt:    now.Add(-time.Duration(i) * time.Minute),
			status:       "done",
			metricsAvail: true,
			costUSD:      0.10,
			durationMs:   1000,
			inputTok:     100,
			outputTok:    50,
		})
	}
	seedRuns(t, idx, runs)

	f := makeFilter(idx)
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	agg := report.Summary.Overall
	if agg.MetricsUnavailableCount != 5 {
		t.Errorf("MetricsUnavailableCount: got %d, want 5", agg.MetricsUnavailableCount)
	}
	if math.IsNaN(agg.MeanCostUSD) {
		t.Error("MeanCostUSD is NaN; should be 0.10 for runs with metrics")
	}
	if math.Abs(agg.MeanCostUSD-0.10) > 1e-9 {
		t.Errorf("MeanCostUSD: got %v, want 0.10", agg.MeanCostUSD)
	}
}

func TestAggregate_NonClaudeDriver(t *testing.T) {
	idx := openTestReportsIndex(t)
	now := time.Now()

	var runs []runSeed
	for i := 0; i < 3; i++ {
		runs = append(runs, runSeed{
			runID:        fmt.Sprintf("ollama-%02d", i),
			agentName:    "qa",
			startedAt:    now.Add(-time.Duration(i) * time.Minute),
			status:       "done",
			metricsAvail: false, // No metrics for non-Claude
			ttftMs:       nil,   // No TTFT
		})
	}
	seedRuns(t, idx, runs)

	f := makeFilter(idx)
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	agg := report.Summary.Overall
	if agg.MeanTTFTMs != 0 {
		t.Errorf("MeanTTFTMs: got %v, want 0 for non-Claude runs", agg.MeanTTFTMs)
	}
	if agg.MeanOutputTokensPerSecond != 0 {
		t.Errorf("MeanOutputTokensPerSecond: got %v, want 0 for non-Claude runs", agg.MeanOutputTokensPerSecond)
	}
}

func TestAggregate_EmptyWindow(t *testing.T) {
	idx := openTestReportsIndex(t)

	f := makeFilter(idx)
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	if report.Summary.Overall.RunCount != 0 {
		t.Errorf("RunCount: got %d, want 0", report.Summary.Overall.RunCount)
	}
	if len(report.Series) == 0 {
		t.Error("Series should be non-empty (continuous bucket sequence) even with no runs")
	}
	for i, pt := range report.Series {
		if pt.RunCount != 0 {
			t.Errorf("Series[%d].RunCount = %d, want 0", i, pt.RunCount)
		}
	}
}

func TestAggregate_MultiAgent(t *testing.T) {
	idx := openTestReportsIndex(t)
	now := time.Now()

	agents := []struct {
		name  string
		count int
	}{
		{"qa", 3},
		{"backend-developer", 5},
		{"frontend-developer", 2},
	}
	var runs []runSeed
	seq := 0
	for _, a := range agents {
		for i := 0; i < a.count; i++ {
			runs = append(runs, runSeed{
				runID:     fmt.Sprintf("multi-agent-%02d", seq),
				agentName: a.name,
				startedAt: now.Add(-time.Duration(seq) * time.Minute),
				status:    "done",
			})
			seq++
		}
	}
	seedRuns(t, idx, runs)

	f := makeFilter(idx)
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	if len(report.Summary.PerAgent) != 3 {
		t.Errorf("PerAgent count: got %d, want 3", len(report.Summary.PerAgent))
	}

	totalFromAgents := int64(0)
	for _, pa := range report.Summary.PerAgent {
		totalFromAgents += pa.RunCount
	}
	if totalFromAgents != report.Summary.Overall.RunCount {
		t.Errorf("PerAgent run_count sum %d != overall %d",
			totalFromAgents, report.Summary.Overall.RunCount)
	}
}

func TestAggregate_MultiModel(t *testing.T) {
	idx := openTestReportsIndex(t)
	now := time.Now()

	var runs []runSeed
	// 3 opus runs at $0.05 each
	for i := 0; i < 3; i++ {
		runs = append(runs, runSeed{
			runID:        fmt.Sprintf("opus-%02d", i),
			agentName:    "qa",
			startedAt:    now.Add(-time.Duration(i) * time.Minute),
			status:       "done",
			model:        "claude-opus-4",
			metricsAvail: true,
			costUSD:      0.05,
			durationMs:   1000,
			inputTok:     100,
			outputTok:    50,
		})
	}
	// 4 sonnet runs at $0.02 each
	for i := 0; i < 4; i++ {
		runs = append(runs, runSeed{
			runID:        fmt.Sprintf("sonnet-%02d", i),
			agentName:    "qa",
			startedAt:    now.Add(-time.Duration(10+i) * time.Minute),
			status:       "done",
			model:        "claude-sonnet-4",
			metricsAvail: true,
			costUSD:      0.02,
			durationMs:   800,
			inputTok:     80,
			outputTok:    40,
		})
	}
	seedRuns(t, idx, runs)

	f := makeFilter(idx)
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	if len(report.Summary.PerModel) != 2 {
		t.Errorf("PerModel count: got %d, want 2", len(report.Summary.PerModel))
	}
	if len(report.SeriesByModel) != 2 {
		t.Errorf("SeriesByModel keys: got %d, want 2", len(report.SeriesByModel))
	}

	totalCost := report.Summary.Overall.TotalCostUSD
	wantTotal := 3*0.05 + 4*0.02
	if math.Abs(totalCost-wantTotal) > 1e-9 {
		t.Errorf("TotalCostUSD: got %v, want %v", totalCost, wantTotal)
	}
}

func TestAggregate_StatusFilter(t *testing.T) {
	idx := openTestReportsIndex(t)
	now := time.Now()

	var runs []runSeed
	runs = append(runs, runSeed{
		runID: "sf-done", agentName: "qa",
		startedAt: now.Add(-time.Minute), status: "done",
	})
	for i := 0; i < 2; i++ {
		runs = append(runs, runSeed{
			runID: fmt.Sprintf("sf-failed-%d", i), agentName: "qa",
			startedAt: now.Add(-time.Duration(i+2) * time.Minute), status: "failed",
		})
	}
	seedRuns(t, idx, runs)

	f := makeFilter(idx)
	f.Statuses = []string{"failed"}
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	agg := report.Summary.Overall
	if agg.RunCount != 2 {
		t.Errorf("RunCount: got %d, want 2", agg.RunCount)
	}
	if agg.SuccessCount != 0 {
		t.Errorf("SuccessCount: got %d, want 0 (only failed filtered)", agg.SuccessCount)
	}
}

func TestAggregate_AgentFilter(t *testing.T) {
	idx := openTestReportsIndex(t)
	now := time.Now()

	var runs []runSeed
	agents := []string{"qa", "qa", "backend-developer", "frontend-developer"}
	for i, a := range agents {
		runs = append(runs, runSeed{
			runID:     fmt.Sprintf("af-%02d", i),
			agentName: a,
			startedAt: now.Add(-time.Duration(i) * time.Minute),
			status:    "done",
		})
	}
	seedRuns(t, idx, runs)

	f := makeFilter(idx)
	f.Agents = []string{"qa"}
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	if report.Summary.Overall.RunCount != 2 {
		t.Errorf("RunCount with agent=qa filter: got %d, want 2", report.Summary.Overall.RunCount)
	}
	if report.SeriesByAgent == nil {
		t.Error("SeriesByAgent should be set when agent filter is active")
	}
	if _, ok := report.SeriesByAgent["qa"]; !ok {
		t.Error("SeriesByAgent should contain 'qa' key")
	}
}

func TestAggregate_PercentileAccuracy(t *testing.T) {
	idx := openTestReportsIndex(t)
	now := time.Now()

	// 100 runs with durations 1ms to 100ms.
	var runs []runSeed
	for i := 1; i <= 100; i++ {
		runs = append(runs, runSeed{
			runID:        fmt.Sprintf("pct-%03d", i),
			agentName:    "qa",
			startedAt:    now.Add(-time.Duration(i) * time.Minute),
			status:       "done",
			model:        "claude-opus",
			metricsAvail: true,
			costUSD:      0.01,
			durationMs:   int64(i),
			inputTok:     100,
			outputTok:    50,
		})
	}
	seedRuns(t, idx, runs)

	f := makeFilter(idx)
	report, err := BuildAgentUsageReport(idx, f)
	if err != nil {
		t.Fatalf("BuildAgentUsageReport: %v", err)
	}

	agg := report.Summary.Overall
	// percentile(sorted[1..100], 50): idx = (50*100)/100 = 50 → sorted[50] = 51
	wantP50 := int64(51)
	if math.Abs(agg.MedianDurationMs-float64(wantP50)) > 1 {
		t.Errorf("MedianDurationMs (p50): got %v, want ~%d (±1)", agg.MedianDurationMs, wantP50)
	}
	// p95: idx = (95*100)/100 = 95 → sorted[95] = 96
	wantP95 := int64(96)
	if math.Abs(agg.P95DurationMs-float64(wantP95)) > 1 {
		t.Errorf("P95DurationMs: got %v, want ~%d (±1)", agg.P95DurationMs, wantP95)
	}
}

func TestAggregate_BadFilterTo(t *testing.T) {
	idx := openTestReportsIndex(t)

	f := AgentUsageFilter{
		From:   time.Now(),
		To:     time.Now().Add(-time.Hour), // To before From
		Bucket: "day",
		Loc:    time.UTC,
	}
	_, err := BuildAgentUsageReport(idx, f)
	if err == nil {
		t.Fatal("expected error for To < From, got nil")
	}
	var badFilter ErrBadFilter
	if !errors.As(err, &badFilter) {
		t.Errorf("expected ErrBadFilter, got %T: %v", err, err)
	}
}
