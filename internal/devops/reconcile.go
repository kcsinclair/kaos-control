// SPDX-License-Identifier: AGPL-3.0-or-later

package devops

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// orphanCandidate is a run whose .log file has no matching .meta.json
// sidecar — a run that never reached a persisted terminal state — derived
// from the run.started header of its log.
type orphanCandidate struct {
	runID     string
	slug      string
	startedAt time.Time
}

// readLogHeader returns the first decoded entry of the log file at path, or
// ok=false if it cannot be opened or contains no valid JSON-lines entries.
func readLogHeader(path string) (logEntry, bool) {
	f, err := os.Open(path)
	if err != nil {
		return logEntry{}, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e logEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		return e, true
	}
	return logEntry{}, false
}

// orphanCandidates scans projectName's run log directory for .log files with
// no matching .meta.json sidecar. A completed run always gets a .meta.json
// written by the runner's event hook, so the absence of one means the run
// never reached a persisted terminal state — either it is still genuinely
// executing (a live handle exists), or the handle that would have completed
// it is gone (orphaned). Callers distinguish the two with isActive.
func (ls *LogStore) orphanCandidates(projectName string) ([]orphanCandidate, error) {
	dir := ls.projectLogsDir(projectName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("devops: scanning run logs: %w", err)
	}

	hasMeta := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".meta.json") {
			hasMeta[strings.TrimSuffix(e.Name(), ".meta.json")] = true
		}
	}

	var candidates []orphanCandidate
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		runID := strings.TrimSuffix(e.Name(), ".log")
		if hasMeta[runID] {
			continue
		}

		first, ok := readLogHeader(filepath.Join(dir, e.Name()))
		if !ok || first.EventType != EventRunStarted {
			continue
		}
		m, ok := first.Payload.(map[string]any)
		if !ok {
			continue
		}
		slug, _ := m["pipeline_slug"].(string)
		if slug == "" {
			continue
		}
		candidates = append(candidates, orphanCandidate{runID: runID, slug: slug, startedAt: first.Time})
	}
	return candidates, nil
}

// markOrphanCancelled persists a "cancelled" RunRecord for an orphaned run
// and appends a matching run.completed entry to its log, so both the log
// viewer and run-history listing show a resolved terminal state instead of
// leaving the run looking stuck in progress.
func (ls *LogStore) markOrphanCancelled(projectName string, c orphanCandidate) (RunRecord, error) {
	endedAt := time.Now().UTC()
	duration := endedAt.Sub(c.startedAt)

	ls.WriteEvent(projectName, c.runID, EventRunCompleted, RunCompletedPayload{
		RunID:           c.runID,
		Pipeline:        c.slug,
		Project:         projectName,
		Status:          string(StepCancelled),
		DurationSeconds: duration.Seconds(),
	})

	rec := RunRecord{
		RunID:      c.runID,
		Slug:       c.slug,
		StartedAt:  c.startedAt.UTC().Format(time.RFC3339),
		EndedAt:    endedAt.Format(time.RFC3339),
		DurationMs: duration.Milliseconds(),
		Status:     string(StepCancelled),
		LogRef:     c.runID + ".log",
	}
	if err := ls.WriteRecord(projectName, rec); err != nil {
		return RunRecord{}, err
	}
	return rec, nil
}

// ReconcileOrphanedRuns marks every orphaned run for projectName — a run
// whose .log file has no terminal .meta.json sidecar and is not covered by
// isActive — as "cancelled". It is intended to run once at startup: with a
// freshly created Runner, no handle is live yet, so any run still showing as
// in-progress from a prior process cannot actually still be running.
// Returns the number of runs reconciled.
func (ls *LogStore) ReconcileOrphanedRuns(projectName string, isActive func(runID string) bool) (int, error) {
	candidates, err := ls.orphanCandidates(projectName)
	if err != nil {
		return 0, err
	}

	reconciled := 0
	for _, c := range candidates {
		if isActive != nil && isActive(c.runID) {
			continue
		}
		if _, err := ls.markOrphanCancelled(projectName, c); err != nil {
			slog.Warn("devops: reconcile orphaned run", "project", projectName, "run_id", c.runID, "err", err)
			continue
		}
		reconciled++
	}

	if reconciled > 0 {
		slog.Info("devops: reconciled orphaned runs", "project", projectName, "count", reconciled)
	}
	return reconciled, nil
}

// ReconcileSlugOrphan finds the most recently started orphaned run for slug
// (a .log file with no terminal .meta.json sidecar) and marks it
// "cancelled". Returns ok=false if slug has no orphaned run to reconcile.
// Callers are expected to have already confirmed no live handle exists for
// slug (e.g. via Runner.ActiveRunID) before calling this.
func (ls *LogStore) ReconcileSlugOrphan(projectName, slug string) (RunRecord, bool, error) {
	candidates, err := ls.orphanCandidates(projectName)
	if err != nil {
		return RunRecord{}, false, err
	}

	var latest *orphanCandidate
	for i := range candidates {
		c := &candidates[i]
		if c.slug != slug {
			continue
		}
		if latest == nil || c.startedAt.After(latest.startedAt) {
			latest = c
		}
	}
	if latest == nil {
		return RunRecord{}, false, nil
	}

	rec, err := ls.markOrphanCancelled(projectName, *latest)
	if err != nil {
		return RunRecord{}, false, err
	}
	return rec, true, nil
}
