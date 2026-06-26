// SPDX-License-Identifier: AGPL-3.0-or-later

package devops

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LogStore persists pipeline run events to JSON-lines files under
// baseDir/devops/<project-name>/<run_id>.log. It also provides read and list
// operations for completed and in-progress runs.
type LogStore struct {
	baseDir string // app-level data dir (e.g. ~/.kaos-control/data)
}

// NewLogStore creates a LogStore rooted at baseDir.
func NewLogStore(baseDir string) *LogStore {
	return &LogStore{baseDir: baseDir}
}

// logEntry is one line in the JSON-lines log file.
type logEntry struct {
	Time      time.Time `json:"time"`
	EventType string    `json:"event_type"`
	Payload   any       `json:"payload"`
}

// projectLogsDir returns the directory for a project's run logs.
func (ls *LogStore) projectLogsDir(projectName string) string {
	return filepath.Join(ls.baseDir, "devops", projectName)
}

// logPath returns the absolute path of the log file for runID.
func (ls *LogStore) logPath(projectName, runID string) string {
	return filepath.Join(ls.projectLogsDir(projectName), runID+".log")
}

// WriteEvent appends one JSON-lines record to the run log for runID.
// The directory is created automatically on first write.
func (ls *LogStore) WriteEvent(projectName, runID, eventType string, payload any) {
	dir := ls.projectLogsDir(projectName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("devops: create log dir", "dir", dir, "err", err)
		return
	}

	path := ls.logPath(projectName, runID)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("devops: open log file", "path", path, "err", err)
		return
	}
	defer f.Close()

	entry := logEntry{
		Time:      time.Now().UTC(),
		EventType: eventType,
		Payload:   payload,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		slog.Error("devops: marshal log entry", "err", err)
		return
	}
	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		slog.Error("devops: write log entry", "err", err)
	}
}

// ReadLog returns the full contents of the log file for runID (works for both
// in-progress and completed runs). Returns an error if the file does not exist.
func (ls *LogStore) ReadLog(projectName, runID string) ([]byte, error) {
	path := ls.logPath(projectName, runID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("devops: reading log %s: %w", runID, err)
	}
	return data, nil
}

// RunRecord is persisted as a sidecar JSON file (<run_id>.meta.json) next to
// the run log when a run reaches a terminal state.
type RunRecord struct {
	RunID      string `json:"run_id"`
	Slug       string `json:"slug"`
	StartedAt  string `json:"started_at"`  // RFC 3339
	EndedAt    string `json:"ended_at"`    // RFC 3339
	DurationMs int64  `json:"duration_ms"`
	Status     string `json:"status"`     // passed | failed | cancelled
	LogRef     string `json:"log_ref"`    // "<run_id>.log"
}

// metaPath returns the absolute path of the sidecar meta file for runID.
func (ls *LogStore) metaPath(projectName, runID string) string {
	return filepath.Join(ls.projectLogsDir(projectName), runID+".meta.json")
}

// WriteRecord atomically persists a RunRecord as a sidecar JSON file.
// Uses a temp-file + rename so a killed server never leaves a partial file.
func (ls *LogStore) WriteRecord(projectName string, rec RunRecord) error {
	dir := ls.projectLogsDir(projectName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("devops: create log dir: %w", err)
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("devops: marshal run record: %w", err)
	}

	destPath := ls.metaPath(projectName, rec.RunID)
	tmp, err := os.CreateTemp(dir, rec.RunID+".meta.json.tmp*")
	if err != nil {
		return fmt.Errorf("devops: create temp record file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("devops: write run record: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("devops: sync run record: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("devops: close run record: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("devops: rename run record: %w", err)
	}
	return nil
}

// readRecord decodes a single .meta.json sidecar file.
func readRecord(path string) (RunRecord, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunRecord{}, false
	}
	var rec RunRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return RunRecord{}, false
	}
	return rec, true
}

// Record looks up a run record by project and runID.
func (ls *LogStore) Record(projectName, runID string) (RunRecord, bool) {
	return readRecord(ls.metaPath(projectName, runID))
}

// backfillRecord derives a RunRecord from a legacy .log file that has no
// matching .meta.json sidecar. It persists the result so future calls are fast.
// Returns (record, true) on success; (zero, false) if the log cannot be parsed.
func (ls *LogStore) backfillRecord(projectName, runID string) (RunRecord, bool) {
	path := ls.logPath(projectName, runID)
	f, err := os.Open(path)
	if err != nil {
		return RunRecord{}, false
	}
	defer f.Close()

	var first, last *logEntry
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
		if first == nil {
			first = &e
		}
		last = &e
	}

	if first == nil {
		return RunRecord{}, false
	}

	var slug, status string
	startedAt := first.Time
	endedAt := last.Time

	if first.EventType == EventRunStarted {
		if m, ok := first.Payload.(map[string]any); ok {
			slug, _ = m["pipeline_slug"].(string)
		}
	}

	durationMs := endedAt.Sub(startedAt).Milliseconds()

	if last != nil && last.EventType == EventRunCompleted {
		if m, ok := last.Payload.(map[string]any); ok {
			status, _ = m["status"].(string)
			if d, ok := m["duration_seconds"].(float64); ok {
				durationMs = int64(d * 1000)
			}
		}
	} else {
		// No completion entry — infer best-effort status from last step.
		status = "unknown"
		if last != nil && last.EventType == EventStepCompleted {
			if m, ok := last.Payload.(map[string]any); ok {
				status, _ = m["status"].(string)
			}
		}
	}

	if slug == "" {
		return RunRecord{}, false
	}

	rec := RunRecord{
		RunID:      runID,
		Slug:       slug,
		StartedAt:  startedAt.UTC().Format(time.RFC3339),
		EndedAt:    endedAt.UTC().Format(time.RFC3339),
		DurationMs: durationMs,
		Status:     status,
		LogRef:     runID + ".log",
	}

	if err := ls.WriteRecord(projectName, rec); err != nil {
		slog.Warn("devops: backfill write record", "project", projectName, "run_id", runID, "err", err)
	}
	return rec, true
}

// ListPipelineRuns returns finished runs for one pipeline slug, newest-first.
// limit<=0 means all; callers should cap at 50. Corrupt records are skipped
// with a WARN log (NF3). Legacy .log files without a sidecar are back-filled
// lazily.
func (ls *LogStore) ListPipelineRuns(projectName, slug string, limit int) ([]RunRecord, error) {
	dir := ls.projectLogsDir(projectName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("devops: listing pipeline runs: %w", err)
	}

	// Build a set of run IDs that already have a sidecar.
	hasMeta := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".meta.json") {
			runID := strings.TrimSuffix(e.Name(), ".meta.json")
			hasMeta[runID] = true
		}
	}

	var records []RunRecord

	// Read existing sidecar records.
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		rec, ok := readRecord(path)
		if !ok {
			slog.Warn("devops: skipping corrupt run record", "project", projectName, "file", e.Name())
			continue
		}
		if rec.Slug != slug {
			continue
		}
		records = append(records, rec)
	}

	// Back-fill from .log files that have no sidecar.
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		runID := strings.TrimSuffix(e.Name(), ".log")
		if hasMeta[runID] {
			continue
		}
		rec, ok := ls.backfillRecord(projectName, runID)
		if !ok {
			slog.Warn("devops: cannot backfill run from log", "project", projectName, "run_id", runID)
			continue
		}
		if rec.Slug != slug {
			continue
		}
		records = append(records, rec)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt > records[j].StartedAt
	})

	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}
	return records, nil
}

// PruneOldRuns removes the oldest finished runs for slug beyond the keep
// threshold. Both .meta.json and .log files are deleted for each pruned run.
// isActive must return true for any runID that is currently executing; those
// runs are never pruned. Returns the count of run pairs removed.
func (ls *LogStore) PruneOldRuns(projectName, slug string, keep int, isActive func(runID string) bool) (int, error) {
	// Collect all sidecar records for this slug, sorted newest-first.
	records, err := ls.ListPipelineRuns(projectName, slug, 0)
	if err != nil {
		return 0, err
	}

	if len(records) <= keep {
		return 0, nil
	}

	toRemove := records[keep:]
	removed := 0
	for _, rec := range toRemove {
		if isActive != nil && isActive(rec.RunID) {
			continue
		}
		metaPath := ls.metaPath(projectName, rec.RunID)
		logPath := ls.logPath(projectName, rec.RunID)
		_ = os.Remove(metaPath)
		_ = os.Remove(logPath)
		removed++
	}

	if removed > 0 {
		slog.Info("devops: pruned old runs", "project", projectName, "slug", slug, "removed", removed)
	}
	return removed, nil
}

// RunSummary is a brief description of a past pipeline run extracted from its
// log file header.
type RunSummary struct {
	RunID     string    `json:"run_id"`
	Pipeline  string    `json:"pipeline"`
	StartTime time.Time `json:"start_time"`
	Status    string    `json:"status"`
}

// ListRuns returns a summary for every run log found in the project's log
// directory. Unreadable or malformed files are silently skipped.
func (ls *LogStore) ListRuns(projectName string) ([]RunSummary, error) {
	dir := ls.projectLogsDir(projectName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("devops: listing runs: %w", err)
	}

	var summaries []RunSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		runID := strings.TrimSuffix(e.Name(), ".log")
		s := summariseLog(filepath.Join(dir, e.Name()), runID)
		if s != nil {
			summaries = append(summaries, *s)
		}
	}
	return summaries, nil
}

// ReadLogNDJSON returns the run log reformatted as NDJSON suitable for the
// frontend split-pane viewer.  Each output line is a flat JSON object with a
// "type" field (the pipeline event name) merged with the payload fields.
// Internal log-store fields (time, event_type) are not forwarded.
func (ls *LogStore) ReadLogNDJSON(projectName, runID string) ([]byte, error) {
	path := ls.logPath(projectName, runID)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("devops: reading log %s: %w", runID, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry logEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		// Start building the output object from the payload fields.
		// The payload was stored as a struct; when read back via `any` it is a
		// map[string]any.  We copy those fields then inject the "type" key.
		out := make(map[string]any)
		if payload, ok := entry.Payload.(map[string]any); ok {
			for k, v := range payload {
				out[k] = v
			}
		}
		out["type"] = entry.EventType

		data, err := json.Marshal(out)
		if err != nil {
			continue
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

// StreamLogNDJSON writes the run log to w as NDJSON, flushing after each line
// so callers receive data progressively without buffering the whole file.
func (ls *LogStore) StreamLogNDJSON(projectName, runID string, w io.Writer) error {
	path := ls.logPath(projectName, runID)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("devops: reading log %s: %w", runID, err)
	}
	defer f.Close()

	flusher, canFlush := w.(interface{ Flush() error })

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry logEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		out := make(map[string]any)
		if payload, ok := entry.Payload.(map[string]any); ok {
			for k, v := range payload {
				out[k] = v
			}
		}
		out["type"] = entry.EventType

		data, err := json.Marshal(out)
		if err != nil {
			continue
		}
		if _, err := w.Write(append(data, '\n')); err != nil {
			return err
		}
		if canFlush {
			_ = flusher.Flush()
		}
	}
	return scanner.Err()
}

// summariseLog reads the first and last relevant log entries to build a RunSummary.
func summariseLog(path, runID string) *RunSummary {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var firstEntry *logEntry
	var lastEntry *logEntry

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry logEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if firstEntry == nil {
			firstEntry = &entry
		}
		lastEntry = &entry
	}

	if firstEntry == nil {
		return nil
	}

	summary := &RunSummary{
		RunID:     runID,
		StartTime: firstEntry.Time,
		Status:    "unknown",
	}

	// Extract pipeline name from the run.started payload if present.
	if firstEntry.EventType == EventRunStarted {
		if m, ok := firstEntry.Payload.(map[string]any); ok {
			if p, ok := m["pipeline_slug"].(string); ok {
				summary.Pipeline = p
			}
		}
	}

	// Extract final status from the run.completed payload if present.
	if lastEntry != nil && lastEntry.EventType == EventRunCompleted {
		if m, ok := lastEntry.Payload.(map[string]any); ok {
			if s, ok := m["status"].(string); ok {
				summary.Status = s
			}
		}
	}

	return summary
}
