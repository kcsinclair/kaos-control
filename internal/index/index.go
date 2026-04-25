// Package index manages the SQLite artifact index (a cache; disk is authoritative).
package index

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/config"
)

const schemaVersion = 1

// Index wraps the SQLite database for one project.
type Index struct {
	db          *sql.DB
	projectRoot string
}

// Open opens (or creates) the SQLite index at dbPath for the given project root.
// The schema is created if missing; if the stored schema version differs, the
// index is dropped and rebuilt from disk.
func Open(dbPath, projectRoot string, stages []config.Stage) (*Index, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating index dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer

	idx := &Index{db: db, projectRoot: projectRoot}

	needRebuild, err := idx.checkSchema()
	if err != nil {
		return nil, err
	}
	if needRebuild {
		slog.Info("index schema mismatch or missing — rebuilding from disk", "db", dbPath)
		if err := idx.dropAndRecreate(); err != nil {
			return nil, err
		}
	}
	// Always scan on startup: the index is a cache and files may have changed
	// while the server was not running (watcher only covers live changes).
	if err := idx.Scan(stages); err != nil {
		return nil, fmt.Errorf("initial scan: %w", err)
	}
	return idx, nil
}

// Close closes the underlying database connection.
func (idx *Index) Close() error {
	return idx.db.Close()
}

// Scan walks the lifecycle/ directories and upserts every .md file it finds.
func (idx *Index) Scan(stages []config.Stage) error {
	lifecycleRoot := filepath.Join(idx.projectRoot, "lifecycle")
	count := 0
	start := time.Now()

	for _, stage := range stages {
		dir := filepath.Join(lifecycleRoot, stage.Dir)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
				return err
			}
			if err := idx.IndexFile(path); err != nil {
				slog.Warn("index file error", "path", path, "err", err)
			}
			count++
			return nil
		})
		if err != nil {
			return fmt.Errorf("walking stage %s: %w", stage.Name, err)
		}
	}

	slog.Info("scan complete", "files", count, "duration", time.Since(start))
	return nil
}

// IndexFile reads, parses, and upserts one file into the index.
func (idx *Index) IndexFile(absPath string) error {
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	relPath, err := filepath.Rel(idx.projectRoot, absPath)
	if err != nil {
		return err
	}
	relPath = filepath.ToSlash(relPath) // normalise to forward slashes

	a := artifact.Parse(raw, relPath, info.ModTime())
	return idx.Upsert(a)
}

// DeletePath removes an artifact and its links by project-relative path.
func (idx *Index) DeletePath(relPath string) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.Exec(`DELETE FROM artifacts WHERE path = ?`, relPath); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM links WHERE src = ?`, relPath); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM labels_index WHERE artifact = ?`, relPath); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM parse_errors WHERE path = ?`, relPath); err != nil {
		return err
	}
	return tx.Commit()
}

// Upsert inserts or replaces one artifact and its links in the index.
func (idx *Index) Upsert(a *artifact.Artifact) error {
	fmJSON, err := json.Marshal(a.FM)
	if err != nil {
		return fmt.Errorf("marshalling frontmatter: %w", err)
	}

	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO artifacts
			(path, slug, lineage, idx, stage, type, status, title, frontmatter_json, body_sha256, mtime)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		a.Path, a.Slug, a.FM.Lineage, a.Index, a.Stage,
		a.FM.Type, a.FM.Status, a.FM.Title,
		string(fmJSON), a.SHA256[:], a.Mtime.Unix(),
	)
	if err != nil {
		return fmt.Errorf("upserting artifact: %w", err)
	}

	// Replace links for this source.
	if _, err := tx.Exec(`DELETE FROM links WHERE src = ?`, a.Path); err != nil {
		return err
	}
	for _, l := range a.Links {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO links (src, dst, kind, source) VALUES (?,?,?,?)`,
			l.From, l.To, l.Kind, l.Source,
		); err != nil {
			return err
		}
	}

	// Replace labels for this artifact.
	if _, err := tx.Exec(`DELETE FROM labels_index WHERE artifact = ?`, a.Path); err != nil {
		return err
	}
	for _, lbl := range a.FM.Labels {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO labels_index (label, artifact) VALUES (?,?)`,
			lbl, a.Path,
		); err != nil {
			return err
		}
	}

	// Clear and re-add parse errors.
	if _, err := tx.Exec(`DELETE FROM parse_errors WHERE path = ?`, a.Path); err != nil {
		return err
	}
	for _, msg := range a.ParseErrs {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO parse_errors (path, message, created_at) VALUES (?,?,?)`,
			a.Path, msg, time.Now().Unix(),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ----- query types -----

// Filter holds list/graph query parameters.
type Filter struct {
	Stage   string
	Status  string
	Label   string
	Lineage string
	Type    string
	Limit   int
	Offset  int
}

func (f *Filter) withDefaults() Filter {
	out := *f
	if out.Limit <= 0 {
		out.Limit = 50
	}
	if out.Limit > 500 {
		out.Limit = 500
	}
	return out
}

// ArtifactRow is a lightweight summary row returned from list/graph queries.
type ArtifactRow struct {
	Path     string          `json:"path"`
	Slug     string          `json:"slug"`
	Lineage  string          `json:"lineage"`
	Index    int             `json:"index"`
	Stage    string          `json:"stage"`
	Type     string          `json:"type"`
	Status   string          `json:"status"`
	Title    string          `json:"title"`
	FM       artifact.Frontmatter `json:"frontmatter"`
	Mtime    time.Time       `json:"mtime"`
}

// List returns a filtered, paginated list of artifacts and the total matching count.
func (idx *Index) List(f Filter) ([]*ArtifactRow, int, error) {
	f = f.withDefaults()
	where, args := buildWhere(f)

	var total int
	row := idx.db.QueryRow("SELECT COUNT(*) FROM artifacts"+where, args...)
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	rows, err := idx.db.Query(
		`SELECT path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime
		 FROM artifacts`+where+` ORDER BY lineage, idx, path LIMIT ? OFFSET ?`,
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanRows(rows)
}

// Get returns a single artifact by project-relative path, or nil if not found.
func (idx *Index) Get(relPath string) (*ArtifactRow, error) {
	rows, err := idx.db.Query(
		`SELECT path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime
		 FROM artifacts WHERE path = ?`, relPath,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, _, err := scanRows(rows)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return items[0], nil
}

// ----- graph -----

// GraphNode is a single node in the visualisation graph.
type GraphNode struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Stage   string `json:"stage"`
	Lineage string `json:"lineage"`
	Slug    string `json:"slug"`
	Index   int    `json:"index"`
}

// GraphEdge is a directed relationship between two nodes.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

// GraphData is the full graph payload ready for 3d-force-graph / Cytoscape.
type GraphData struct {
	Nodes []*GraphNode `json:"nodes"`
	Edges []*GraphEdge `json:"edges"`
}

// Graph returns all nodes and edges, optionally filtered.
func (idx *Index) Graph(f Filter) (*GraphData, error) {
	where, args := buildWhere(f)
	rows, err := idx.db.Query(
		`SELECT path, slug, lineage, idx, stage, type, status, title FROM artifacts`+where+
			` ORDER BY lineage, idx`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodeSet := map[string]bool{}
	var nodes []*GraphNode
	for rows.Next() {
		n := &GraphNode{}
		if err := rows.Scan(&n.ID, &n.Slug, &n.Lineage, &n.Index, &n.Stage, &n.Type, &n.Status, &n.Title); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
		nodeSet[n.ID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Edges: only include edges where both endpoints are in the node set.
	// If a filter is applied, some targets may be outside the visible set.
	linkRows, err := idx.db.Query(`SELECT src, dst, kind FROM links`)
	if err != nil {
		return nil, err
	}
	defer linkRows.Close()

	var edges []*GraphEdge
	for linkRows.Next() {
		e := &GraphEdge{}
		if err := linkRows.Scan(&e.Source, &e.Target, &e.Kind); err != nil {
			return nil, err
		}
		if !nodeSet[e.Source] && !nodeSet[e.Target] {
			continue
		}
		edges = append(edges, e)
	}
	if err := linkRows.Err(); err != nil {
		return nil, err
	}

	return &GraphData{Nodes: nodes, Edges: edges}, nil
}

// Labels returns all distinct label values across the project.
func (idx *Index) Labels() ([]string, error) {
	rows, err := idx.db.Query(`SELECT DISTINCT label FROM labels_index ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var labels []string
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

// LineageSummary is a summary of one lineage group.
type LineageSummary struct {
	Lineage  string        `json:"lineage"`
	Members  []string      `json:"members"`  // artifact paths
	Statuses map[string]int `json:"statuses"` // status → count
}

// Lineages returns a summary of each lineage group.
func (idx *Index) Lineages() ([]*LineageSummary, error) {
	rows, err := idx.db.Query(
		`SELECT lineage, path, status FROM artifacts ORDER BY lineage, idx`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byLineage := map[string]*LineageSummary{}
	var order []string
	for rows.Next() {
		var lineage, path, status string
		if err := rows.Scan(&lineage, &path, &status); err != nil {
			return nil, err
		}
		s, ok := byLineage[lineage]
		if !ok {
			s = &LineageSummary{Lineage: lineage, Statuses: map[string]int{}}
			byLineage[lineage] = s
			order = append(order, lineage)
		}
		s.Members = append(s.Members, path)
		s.Statuses[status]++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*LineageSummary, 0, len(order))
	for _, l := range order {
		result = append(result, byLineage[l])
	}
	return result, nil
}

// NextIndexForLineage returns the next monotonic index to use when creating a new
// artifact for lineage. Returns 0 if the lineage has no artifacts yet (originating
// file), 2 if only the originating file exists, or max+1 otherwise.
func (idx *Index) NextIndexForLineage(lineage string) (int, error) {
	var maxIdx sql.NullInt64
	err := idx.db.QueryRow(`SELECT MAX(idx) FROM artifacts WHERE lineage = ?`, lineage).Scan(&maxIdx)
	if err != nil {
		return 0, err
	}
	if !maxIdx.Valid {
		return 0, nil // no artifacts for this lineage yet → originating file
	}
	if maxIdx.Int64 == 0 {
		return 2, nil // only the originating file exists → next is -2
	}
	return int(maxIdx.Int64) + 1, nil
}

// InboundLinks returns the distinct source paths of all links pointing at dstPath.
func (idx *Index) InboundLinks(dstPath string) ([]string, error) {
	rows, err := idx.db.Query(`SELECT DISTINCT src FROM links WHERE dst = ?`, dstPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var srcs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		srcs = append(srcs, s)
	}
	return srcs, rows.Err()
}

// ----- agent runs -----

// ErrLocked is returned when a lineage lock is already held.
var ErrLocked = fmt.Errorf("lineage already locked")

// AgentRunRow is a record in the agent_runs table.
type AgentRunRow struct {
	RunID             string     `json:"run_id"`
	AgentName         string     `json:"agent_name"`
	Role              string     `json:"role"`
	TargetPath        string     `json:"target_path"`
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	Status            string     `json:"status"` // running|done|failed|killed
	ExitCode          *int       `json:"exit_code,omitempty"`
	StderrTail        string     `json:"stderr_tail"`
	ArtifactsProduced []string   `json:"artifacts_produced"`
}

// LockRow is a record in the lineage_locks table.
type LockRow struct {
	Lineage       string    `json:"lineage"`
	Holder        string    `json:"holder"`
	Kind          string    `json:"kind"` // editor|agent
	AcquiredAt    time.Time `json:"acquired_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// InsertAgentRun inserts a new agent run record with status=running.
func (idx *Index) InsertAgentRun(r *AgentRunRow) error {
	produced, _ := json.Marshal(r.ArtifactsProduced)
	_, err := idx.db.Exec(
		`INSERT INTO agent_runs (run_id, agent_name, role, target_path, started_at, status, stderr_tail, artifacts_produced_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.RunID, r.AgentName, r.Role, r.TargetPath,
		r.StartedAt.Unix(), r.Status, r.StderrTail, string(produced),
	)
	return err
}

// UpdateAgentRun updates the mutable fields of an existing run record.
func (idx *Index) UpdateAgentRun(r *AgentRunRow) error {
	produced, _ := json.Marshal(r.ArtifactsProduced)
	var finishedAt *int64
	if r.FinishedAt != nil {
		v := r.FinishedAt.Unix()
		finishedAt = &v
	}
	_, err := idx.db.Exec(
		`UPDATE agent_runs SET status=?, finished_at=?, exit_code=?, stderr_tail=?, artifacts_produced_json=?
		 WHERE run_id=?`,
		r.Status, finishedAt, r.ExitCode, r.StderrTail, string(produced), r.RunID,
	)
	return err
}

// GetAgentRun retrieves a single run by ID, or nil if not found.
func (idx *Index) GetAgentRun(runID string) (*AgentRunRow, error) {
	row := idx.db.QueryRow(
		`SELECT run_id, agent_name, role, target_path, started_at, finished_at, status, exit_code, stderr_tail, artifacts_produced_json
		 FROM agent_runs WHERE run_id = ?`, runID,
	)
	return scanAgentRun(row)
}

// ListAgentRuns returns runs optionally filtered by status, newest first.
func (idx *Index) ListAgentRuns(status string, limit int) ([]*AgentRunRow, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = idx.db.Query(
			`SELECT run_id, agent_name, role, target_path, started_at, finished_at, status, exit_code, stderr_tail, artifacts_produced_json
			 FROM agent_runs WHERE status = ? ORDER BY started_at DESC LIMIT ?`,
			status, limit,
		)
	} else {
		rows, err = idx.db.Query(
			`SELECT run_id, agent_name, role, target_path, started_at, finished_at, status, exit_code, stderr_tail, artifacts_produced_json
			 FROM agent_runs ORDER BY started_at DESC LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*AgentRunRow
	for rows.Next() {
		r, err := scanAgentRunRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RecoverRunningRuns marks any runs still in status=running as failed (called on startup).
func (idx *Index) RecoverRunningRuns() error {
	_, err := idx.db.Exec(
		`UPDATE agent_runs SET status='failed', finished_at=? WHERE status='running'`,
		time.Now().Unix(),
	)
	return err
}

func scanAgentRun(row *sql.Row) (*AgentRunRow, error) {
	var r AgentRunRow
	var startedAt int64
	var finishedAt sql.NullInt64
	var exitCode sql.NullInt64
	var producedJSON string
	err := row.Scan(
		&r.RunID, &r.AgentName, &r.Role, &r.TargetPath,
		&startedAt, &finishedAt, &r.Status, &exitCode,
		&r.StderrTail, &producedJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.StartedAt = time.Unix(startedAt, 0)
	if finishedAt.Valid {
		t := time.Unix(finishedAt.Int64, 0)
		r.FinishedAt = &t
	}
	if exitCode.Valid {
		v := int(exitCode.Int64)
		r.ExitCode = &v
	}
	_ = json.Unmarshal([]byte(producedJSON), &r.ArtifactsProduced)
	return &r, nil
}

func scanAgentRunRow(rows *sql.Rows) (*AgentRunRow, error) {
	var r AgentRunRow
	var startedAt int64
	var finishedAt sql.NullInt64
	var exitCode sql.NullInt64
	var producedJSON string
	err := rows.Scan(
		&r.RunID, &r.AgentName, &r.Role, &r.TargetPath,
		&startedAt, &finishedAt, &r.Status, &exitCode,
		&r.StderrTail, &producedJSON,
	)
	if err != nil {
		return nil, err
	}
	r.StartedAt = time.Unix(startedAt, 0)
	if finishedAt.Valid {
		t := time.Unix(finishedAt.Int64, 0)
		r.FinishedAt = &t
	}
	if exitCode.Valid {
		v := int(exitCode.Int64)
		r.ExitCode = &v
	}
	_ = json.Unmarshal([]byte(producedJSON), &r.ArtifactsProduced)
	return &r, nil
}

// ----- lineage locks -----

// AcquireLock attempts to insert a lineage lock. Returns ErrLocked if already held.
func (idx *Index) AcquireLock(lineage, holder, kind string) error {
	now := time.Now().Unix()
	_, err := idx.db.Exec(
		`INSERT INTO lineage_locks (lineage, holder, kind, acquired_at, last_heartbeat) VALUES (?, ?, ?, ?, ?)`,
		lineage, holder, kind, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrLocked
		}
		return err
	}
	return nil
}

// HeartbeatLock updates the last_heartbeat for the given lineage.
func (idx *Index) HeartbeatLock(lineage string) error {
	_, err := idx.db.Exec(
		`UPDATE lineage_locks SET last_heartbeat=? WHERE lineage=?`,
		time.Now().Unix(), lineage,
	)
	return err
}

// ReleaseLock removes the lock for the given lineage.
func (idx *Index) ReleaseLock(lineage string) error {
	_, err := idx.db.Exec(`DELETE FROM lineage_locks WHERE lineage=?`, lineage)
	return err
}

// GetLock returns the current lock for the lineage, or nil if unlocked.
func (idx *Index) GetLock(lineage string) (*LockRow, error) {
	var r LockRow
	var acquiredAt, lastHeartbeat int64
	err := idx.db.QueryRow(
		`SELECT lineage, holder, kind, acquired_at, last_heartbeat FROM lineage_locks WHERE lineage=?`, lineage,
	).Scan(&r.Lineage, &r.Holder, &r.Kind, &acquiredAt, &lastHeartbeat)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.AcquiredAt = time.Unix(acquiredAt, 0)
	r.LastHeartbeat = time.Unix(lastHeartbeat, 0)
	return &r, nil
}

// ListActiveLocks returns all current locks.
func (idx *Index) ListActiveLocks() ([]*LockRow, error) {
	rows, err := idx.db.Query(
		`SELECT lineage, holder, kind, acquired_at, last_heartbeat FROM lineage_locks`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*LockRow
	for rows.Next() {
		var r LockRow
		var acquiredAt, lastHeartbeat int64
		if err := rows.Scan(&r.Lineage, &r.Holder, &r.Kind, &acquiredAt, &lastHeartbeat); err != nil {
			return nil, err
		}
		r.AcquiredAt = time.Unix(acquiredAt, 0)
		r.LastHeartbeat = time.Unix(lastHeartbeat, 0)
		out = append(out, &r)
	}
	return out, rows.Err()
}

// ReapLocks deletes locks whose last_heartbeat is older than maxAge and returns their lineages.
func (idx *Index) ReapLocks(maxAge time.Duration) ([]string, error) {
	cutoff := time.Now().Add(-maxAge).Unix()
	rows, err := idx.db.Query(`SELECT lineage FROM lineage_locks WHERE last_heartbeat < ?`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lineages []string
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			return nil, err
		}
		lineages = append(lineages, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, l := range lineages {
		if _, err := idx.db.Exec(`DELETE FROM lineage_locks WHERE lineage=?`, l); err != nil {
			return lineages, err
		}
	}
	return lineages, nil
}

// ParseErrors returns all recorded parse errors.
type ParseErrorRow struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (idx *Index) ParseErrors() ([]*ParseErrorRow, error) {
	rows, err := idx.db.Query(`SELECT path, message FROM parse_errors ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ParseErrorRow
	for rows.Next() {
		r := &ParseErrorRow{}
		if err := rows.Scan(&r.Path, &r.Message); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ----- schema management -----

func (idx *Index) checkSchema() (needRebuild bool, err error) {
	var v int
	err = idx.db.QueryRow(`SELECT version FROM schema_version`).Scan(&v)
	if err != nil {
		// Table doesn't exist — needs creation.
		return true, nil
	}
	return v != schemaVersion, nil
}

func (idx *Index) dropAndRecreate() error {
	stmts := []string{
		`DROP TABLE IF EXISTS schema_version`,
		`DROP TABLE IF EXISTS artifacts`,
		`DROP TABLE IF EXISTS links`,
		`DROP TABLE IF EXISTS labels_index`,
		`DROP TABLE IF EXISTS agent_runs`,
		`DROP TABLE IF EXISTS lineage_locks`,
		`DROP TABLE IF EXISTS parse_errors`,
	}
	for _, s := range stmts {
		if _, err := idx.db.Exec(s); err != nil {
			return fmt.Errorf("drop: %w", err)
		}
	}
	return idx.createSchema()
}

func (idx *Index) createSchema() error {
	ddl := `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE schema_version (version INTEGER NOT NULL);
INSERT INTO schema_version VALUES (` + fmt.Sprint(schemaVersion) + `);

CREATE TABLE artifacts (
    path              TEXT PRIMARY KEY,
    slug              TEXT NOT NULL,
    lineage           TEXT NOT NULL,
    idx               INTEGER NOT NULL,
    stage             TEXT NOT NULL,
    type              TEXT NOT NULL,
    status            TEXT NOT NULL,
    title             TEXT NOT NULL,
    frontmatter_json  TEXT NOT NULL,
    body_sha256       BLOB NOT NULL,
    mtime             INTEGER NOT NULL
);
CREATE INDEX idx_artifacts_lineage ON artifacts(lineage);
CREATE INDEX idx_artifacts_stage   ON artifacts(stage);
CREATE INDEX idx_artifacts_status  ON artifacts(status);
CREATE INDEX idx_artifacts_slug    ON artifacts(slug);
CREATE INDEX idx_artifacts_type    ON artifacts(type);

CREATE TABLE links (
    src    TEXT NOT NULL,
    dst    TEXT NOT NULL,
    kind   TEXT NOT NULL,
    source TEXT NOT NULL,
    PRIMARY KEY (src, dst, kind, source)
);
CREATE INDEX idx_links_src ON links(src);
CREATE INDEX idx_links_dst ON links(dst);

CREATE TABLE labels_index (
    label    TEXT NOT NULL,
    artifact TEXT NOT NULL,
    PRIMARY KEY (label, artifact)
);

CREATE TABLE parse_errors (
    path       TEXT NOT NULL,
    message    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (path, message)
);

CREATE TABLE agent_runs (
    run_id                   TEXT PRIMARY KEY,
    agent_name               TEXT NOT NULL,
    role                     TEXT NOT NULL,
    target_path              TEXT,
    started_at               INTEGER NOT NULL,
    finished_at              INTEGER,
    status                   TEXT NOT NULL,
    exit_code                INTEGER,
    stderr_tail              TEXT,
    artifacts_produced_json  TEXT
);

CREATE TABLE lineage_locks (
    lineage         TEXT PRIMARY KEY,
    holder          TEXT NOT NULL,
    kind            TEXT NOT NULL,
    acquired_at     INTEGER NOT NULL,
    last_heartbeat  INTEGER NOT NULL
);
`
	_, err := idx.db.Exec(ddl)
	return err
}

// ----- helpers -----

func buildWhere(f Filter) (clause string, args []any) {
	var conds []string
	if f.Stage != "" {
		conds = append(conds, "stage = ?")
		args = append(args, f.Stage)
	}
	if f.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, f.Status)
	}
	if f.Lineage != "" {
		conds = append(conds, "lineage = ?")
		args = append(args, f.Lineage)
	}
	if f.Type != "" {
		conds = append(conds, "type = ?")
		args = append(args, f.Type)
	}
	if f.Label != "" {
		conds = append(conds, "path IN (SELECT artifact FROM labels_index WHERE label = ?)")
		args = append(args, f.Label)
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func scanRows(rows *sql.Rows) ([]*ArtifactRow, int, error) {
	var out []*ArtifactRow
	for rows.Next() {
		var r ArtifactRow
		var fmJSON string
		var mtimeUnix int64
		if err := rows.Scan(
			&r.Path, &r.Slug, &r.Lineage, &r.Index, &r.Stage,
			&r.Type, &r.Status, &r.Title, &fmJSON, &mtimeUnix,
		); err != nil {
			return nil, 0, err
		}
		r.Mtime = time.Unix(mtimeUnix, 0)
		_ = json.Unmarshal([]byte(fmJSON), &r.FM)
		out = append(out, &r)
	}
	return out, len(out), rows.Err()
}
