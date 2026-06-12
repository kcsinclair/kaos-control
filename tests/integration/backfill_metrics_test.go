// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

var (
	backfillBinOnce sync.Once
	backfillBinPath string
)

func getBackfillBin(t *testing.T) string {
	t.Helper()
	backfillBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "kaos-backfill-bin-*")
		if err != nil {
			panic("MkdirTemp: " + err.Error())
		}
		backfillBinPath = filepath.Join(dir, "kaos-control")
		cmd := exec.Command("go", "build", "-o", backfillBinPath, "./cmd/kaos-control")
		cmd.Dir = filepath.Join("..", "..") // repo root
		if out, buildErr := cmd.CombinedOutput(); buildErr != nil {
			panic("go build failed:\n" + string(out))
		}
	})
	return backfillBinPath
}

func runBackfill(t *testing.T, bin, cfgPath, project string, extraArgs ...string) (stdout, stderr string, code int) {
	t.Helper()
	args := []string{"backfill", "agent-run-metrics", "--project", project, "--config", cfgPath}
	args = append(args, extraArgs...)
	cmd := exec.Command(bin, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("exec.Run: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), code
}

// writeBackfillConfig writes a minimal app config.yaml and a project registry
// YAML file. Returns cfgPath and dataDir.
func writeBackfillConfig(t *testing.T) (cfgPath, dataDir string) {
	t.Helper()
	baseDir := t.TempDir()
	dataDir = filepath.Join(baseDir, "data")
	projectsDir := filepath.Join(baseDir, "projects")
	projectRoot := filepath.Join(baseDir, "project-root")

	for _, d := range []string{dataDir, projectsDir, projectRoot} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// App config.yaml
	cfgPath = filepath.Join(baseDir, "config.yaml")
	cfgContent := fmt.Sprintf("data_dir: %q\nprojects_dir: %q\n", dataDir, projectsDir)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Project registry file.
	projFile := filepath.Join(projectsDir, "testproject.yaml")
	projContent := fmt.Sprintf("name: testproject\npath: %q\n", projectRoot)
	if err := os.WriteFile(projFile, []byte(projContent), 0o644); err != nil {
		t.Fatal(err)
	}

	return cfgPath, dataDir
}

// createAgentRunsTable creates the agent_runs table schema in the given DB.
func createAgentRunsTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS agent_runs (
		run_id                   TEXT PRIMARY KEY,
		agent_name               TEXT NOT NULL,
		role                     TEXT NOT NULL,
		target_path              TEXT,
		started_at               INTEGER NOT NULL,
		finished_at              INTEGER,
		status                   TEXT NOT NULL,
		exit_code                INTEGER,
		stderr_tail              TEXT,
		artifacts_produced_json  TEXT,
		metrics_available        INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("CREATE TABLE agent_runs: %v", err)
	}
}

// seedDBRun opens the SQLite DB at dbPath and inserts a minimal run row with
// metrics_available=0.
func seedDBRun(t *testing.T, dbPath, runID, status string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("sql.Open(%s): %v", dbPath, err)
	}
	defer db.Close()

	createAgentRunsTable(t, db)

	_, err = db.Exec(
		`INSERT OR IGNORE INTO agent_runs (run_id, agent_name, role, started_at, status, metrics_available)
		 VALUES (?, 'qa', 'analyst', strftime('%s','now'), ?, 0)`,
		runID, status,
	)
	if err != nil {
		t.Fatalf("INSERT run %s: %v", runID, err)
	}
}

// writeLogFile writes a run log file. If hasResult is true, includes a valid
// type:result JSON line that the backfill parser can extract.
func writeLogFile(t *testing.T, runsDir, runID string, hasResult bool) {
	t.Helper()
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(runsDir, runID+".log")
	var content string
	if hasResult {
		content = `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}` + "\n" +
			`{"type":"result","subtype":"success","total_cost_usd":0.015,"duration_ms":1200,"duration_api_ms":1100,"num_turns":2,"usage":{"input_tokens":150,"cache_creation_input_tokens":10,"cache_read_input_tokens":30,"output_tokens":80}}` + "\n"
	} else {
		content = `{"type":"system","subtype":"init"}` + "\n"
	}
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing log file %s: %v", logPath, err)
	}
}

func openBackfillDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("sql.Open(%s): %v", dbPath, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func getMetricsAvailable(t *testing.T, db *sql.DB, runID string) int {
	t.Helper()
	var v int
	err := db.QueryRow(`SELECT metrics_available FROM agent_runs WHERE run_id=?`, runID).Scan(&v)
	if err != nil {
		t.Fatalf("SELECT metrics_available for %s: %v", runID, err)
	}
	return v
}

// TestBackfill_PopulatesUnparseableRuns seeds 5 done runs with valid log files
// and asserts backfill sets metrics_available=1 for all.
func TestBackfill_PopulatesUnparseableRuns(t *testing.T) {
	bin := getBackfillBin(t)
	cfgPath, dataDir := writeBackfillConfig(t)

	dbPath := filepath.Join(dataDir, "testproject", "index.db")
	runsDir := filepath.Join(dataDir, "testproject", "runs")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("backfill-run-%02d", i)
		seedDBRun(t, dbPath, id, "done")
		writeLogFile(t, runsDir, id, true)
	}

	stdout, _, code := runBackfill(t, bin, cfgPath, "testproject")
	if code != 0 {
		t.Fatalf("backfill exited %d; stdout: %s", code, stdout)
	}

	db := openBackfillDB(t, dbPath)
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("backfill-run-%02d", i)
		if got := getMetricsAvailable(t, db, id); got != 1 {
			t.Errorf("run %s: metrics_available = %d, want 1", id, got)
		}
	}
}

// TestBackfill_SkipsMissingLogs seeds a run with no log file and asserts the
// command succeeds and the run is unchanged.
func TestBackfill_SkipsMissingLogs(t *testing.T) {
	bin := getBackfillBin(t)
	cfgPath, dataDir := writeBackfillConfig(t)

	dbPath := filepath.Join(dataDir, "testproject", "index.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}

	seedDBRun(t, dbPath, "no-log-run", "done")
	// No log file written.

	stdout, _, code := runBackfill(t, bin, cfgPath, "testproject")
	if code != 0 {
		t.Fatalf("backfill exited %d; stdout: %s", code, stdout)
	}

	db := openBackfillDB(t, dbPath)
	if got := getMetricsAvailable(t, db, "no-log-run"); got != 0 {
		t.Errorf("no-log-run: metrics_available = %d, want 0 (no log file)", got)
	}
}

// TestBackfill_Idempotent runs backfill twice; second run should report
// "would backfill 0" (or backfilled 0) because all rows already have
// metrics_available=1.
func TestBackfill_Idempotent(t *testing.T) {
	bin := getBackfillBin(t)
	cfgPath, dataDir := writeBackfillConfig(t)

	dbPath := filepath.Join(dataDir, "testproject", "index.db")
	runsDir := filepath.Join(dataDir, "testproject", "runs")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}

	seedDBRun(t, dbPath, "idempotent-run", "done")
	writeLogFile(t, runsDir, "idempotent-run", true)

	// First run — should backfill 1.
	_, _, code := runBackfill(t, bin, cfgPath, "testproject")
	if code != 0 {
		t.Fatal("first backfill run failed")
	}

	// Second run — should report backfilled 0.
	stdout2, _, code2 := runBackfill(t, bin, cfgPath, "testproject")
	if code2 != 0 {
		t.Fatalf("second backfill run exited %d; stdout: %s", code2, stdout2)
	}
	// The summary line should contain "backfilled 0".
	if !strings.Contains(stdout2, "backfilled 0") {
		t.Errorf("second backfill run should report 'backfilled 0'; got: %s", stdout2)
	}
}

// TestBackfill_DryRun verifies that --dry-run outputs "would backfill" but
// does not update the database.
func TestBackfill_DryRun(t *testing.T) {
	bin := getBackfillBin(t)
	cfgPath, dataDir := writeBackfillConfig(t)

	dbPath := filepath.Join(dataDir, "testproject", "index.db")
	runsDir := filepath.Join(dataDir, "testproject", "runs")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}

	seedDBRun(t, dbPath, "dryrun-run", "done")
	writeLogFile(t, runsDir, "dryrun-run", true)

	stdout, _, code := runBackfill(t, bin, cfgPath, "testproject", "--dry-run")
	if code != 0 {
		t.Fatalf("dry-run backfill exited %d; stdout: %s", code, stdout)
	}

	if !strings.Contains(stdout, "would backfill") {
		t.Errorf("dry-run output should contain 'would backfill'; got: %s", stdout)
	}

	// Database must NOT be updated.
	db := openBackfillDB(t, dbPath)
	if got := getMetricsAvailable(t, db, "dryrun-run"); got != 0 {
		t.Errorf("dryrun-run: metrics_available = %d after --dry-run, want 0", got)
	}
}
