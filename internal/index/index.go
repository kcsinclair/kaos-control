// Package index manages the SQLite artifact index (a cache; disk is authoritative).
package index

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/hub"
)

// Transitioner checks whether a workflow transition is permitted.
// It is satisfied by *workflow.Engine; the interface breaks the otherwise-circular
// index → workflow → index import chain.
type Transitioner interface {
	CanTransition(from, to string, userRoles []string) bool
}

const schemaVersion = 4

// Index wraps the SQLite database for one project.
type Index struct {
	db          *sql.DB
	projectRoot string
	git         *git.Repo    // optional; used for created-date backfill during scan
	ignore      []string     // glob patterns for files to skip during scan and indexing
	hub         *hub.Hub     // optional; used to broadcast auto-transition events
	wf          Transitioner // optional; used to validate auto-transitions
}

// Option configures an Index at construction time.
type Option func(*Index)

// WithGit supplies an optional git repository used to backfill the created
// date for artifacts that lack a created: frontmatter field.
func WithGit(repo *git.Repo) Option {
	return func(idx *Index) { idx.git = repo }
}

// WithIgnore supplies glob patterns (matched against the base name) for files
// that should be silently skipped during Scan and IndexFile.
func WithIgnore(patterns []string) Option {
	return func(idx *Index) { idx.ignore = patterns }
}

// WithHub supplies the WebSocket hub used to broadcast auto-transition events.
func WithHub(h *hub.Hub) Option {
	return func(idx *Index) { idx.hub = h }
}

// WithWorkflow supplies the workflow engine used to validate auto-transitions.
func WithWorkflow(wf Transitioner) Option {
	return func(idx *Index) { idx.wf = wf }
}

// Open opens (or creates) the SQLite index at dbPath for the given project root.
// The schema is created if missing; if the stored schema version differs, the
// index is dropped and rebuilt from disk. Additional options (e.g. WithGit)
// may be supplied and are applied before the startup scan runs.
func Open(dbPath, projectRoot string, stages []config.Stage, opts ...Option) (*Index, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating index dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer

	idx := &Index{db: db, projectRoot: projectRoot}
	for _, o := range opts {
		o(idx)
	}

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
	// agent_runs is not part of the versioned schema so it must be ensured separately.
	if err := idx.ensureAgentRunsTable(); err != nil {
		return nil, fmt.Errorf("ensuring agent_runs table: %w", err)
	}
	// events is not part of the versioned schema so it survives schema rebuilds.
	if err := idx.ensureEventsTable(); err != nil {
		return nil, fmt.Errorf("ensuring events table: %w", err)
	}
	// scheduler_jobs and scheduler_runs are not part of the versioned schema so they
	// survive schema rebuilds.
	if err := idx.ensureSchedulerTables(); err != nil {
		return nil, fmt.Errorf("ensuring scheduler tables: %w", err)
	}
	// Always scan on startup: the index is a cache and files may have changed
	// while the server was not running (watcher only covers live changes).
	if err := idx.Scan(stages); err != nil {
		return nil, fmt.Errorf("initial scan: %w", err)
	}
	// One-time backfill: rewrite any on-disk artifacts that carry a plain-date
	// created field (YYYY-MM-DD) to full RFC3339 format.
	if err := idx.NormaliseDates(); err != nil {
		slog.Warn("normalise dates failed", "err", err)
	}
	// Prune stale rows whose path escapes the project root (e.g. left over
	// from past firmlink-related Rel computations or a project-path change).
	if err := idx.pruneEscapingPaths(); err != nil {
		slog.Warn("pruning escaping paths failed", "err", err)
	}
	return idx, nil
}

// pruneEscapingPaths removes rows whose path is not a valid artifact path:
// either it escapes the project root ("..", "/", "/../") or it isn't a
// markdown file inside lifecycle/. Such rows can come from past firmlink/Rel
// mismatches or from agent commits before non-artifact filtering was added.
func (idx *Index) pruneEscapingPaths() error {
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	conds := []struct {
		table, column string
	}{
		{"artifacts", "path"},
		{"parse_errors", "path"},
		{"links", "src"},
		{"links", "dst"},
		{"labels_index", "artifact"},
	}
	total := int64(0)
	for _, c := range conds {
		// Drop:
		//   * paths that escape the project root
		//   * any path that isn't lifecycle/**.md
		// SQLite LIKE is case-sensitive by default for these patterns.
		query := fmt.Sprintf(
			`DELETE FROM %s WHERE
				%s LIKE '..%%' OR %s LIKE '/%%' OR %s LIKE '%%/../%%'
				OR %s NOT LIKE 'lifecycle/%%'
				OR %s NOT LIKE '%%.md'`,
			c.table,
			c.column, c.column, c.column,
			c.column,
			c.column,
		)
		res, err := tx.Exec(query)
		if err != nil {
			return fmt.Errorf("prune %s.%s: %w", c.table, c.column, err)
		}
		if n, err := res.RowsAffected(); err == nil {
			total += n
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if total > 0 {
		slog.Info("pruned non-artifact paths from index", "rows", total)
	}
	return nil
}

// Close closes the underlying database connection.
func (idx *Index) Close() error {
	return idx.db.Close()
}

// plainDateInFMRe matches a `created:` YAML line whose value is a bare date
// (YYYY-MM-DD), with or without surrounding quotes. Used by NormaliseDates.
var plainDateInFMRe = regexp.MustCompile(`(?m)^(created:[ \t]*)["']?([0-9]{4}-[0-9]{2}-[0-9]{2})["']?[ \t]*$`)

// NormaliseDates scans all indexed artifacts and rewrites any that have a
// plain-date (YYYY-MM-DD) created field to RFC3339 format (midnight in the
// server's local timezone). Only the `created:` line is changed; all other
// file content is preserved. Called once after the startup Scan.
func (idx *Index) NormaliseDates() error {
	rows, err := idx.db.Query(`SELECT path, frontmatter_json FROM artifacts`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type pending struct {
		path    string
		rfc3339 string
	}
	var toFix []pending

	for rows.Next() {
		var relPath, fmJSON string
		if err := rows.Scan(&relPath, &fmJSON); err != nil {
			return err
		}
		var fm struct {
			Created string `json:"created"`
		}
		if err := json.Unmarshal([]byte(fmJSON), &fm); err != nil || fm.Created == "" {
			continue
		}
		// Skip entries that are already RFC3339.
		if _, err := time.Parse(time.RFC3339, fm.Created); err == nil {
			continue
		}
		t, err := time.Parse("2006-01-02", fm.Created)
		if err != nil {
			continue // unrecognised format — leave as-is
		}
		localMidnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Now().Location())
		toFix = append(toFix, pending{path: relPath, rfc3339: localMidnight.Format(time.RFC3339)})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	fixed := 0
	for _, e := range toFix {
		absPath := filepath.Join(idx.projectRoot, filepath.FromSlash(e.path))
		if err := rewriteCreatedField(absPath, e.rfc3339); err != nil {
			slog.Warn("normalise dates: failed to rewrite file", "path", e.path, "err", err)
			continue
		}
		fixed++
	}
	if fixed > 0 {
		slog.Info("normalised plain-date created fields to RFC3339 on disk", "count", fixed)
	}
	return nil
}

// rewriteCreatedField rewrites the `created:` line in the YAML frontmatter of
// absPath, replacing a plain-date value with the provided RFC3339 string.
// All other file content (including frontmatter field order and the body) is
// preserved unchanged.
func rewriteCreatedField(absPath, rfc3339 string) error {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	// Require an opening frontmatter delimiter.
	if !bytes.HasPrefix(raw, []byte("---\n")) {
		return nil // no frontmatter — skip silently
	}

	// Locate the closing "---" line. It starts with "\n---" after the opening delimiter.
	closingIdx := bytes.Index(raw[4:], []byte("\n---"))
	if closingIdx < 0 {
		return fmt.Errorf("no closing frontmatter delimiter found in %s", absPath)
	}
	closingIdx += 4 // adjust relative-to-raw[4:] offset back to raw

	// Replace the created: line only within the frontmatter block.
	fmBytes := raw[4:closingIdx]
	newFMBytes := plainDateInFMRe.ReplaceAll(fmBytes, []byte(`${1}"`+rfc3339+`"`))
	if bytes.Equal(fmBytes, newFMBytes) {
		return nil // already correct or pattern not found
	}

	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(newFMBytes)
	out.Write(raw[closingIdx:])
	return atomicWrite(absPath, out.Bytes())
}

// loadStoredMtimes returns a map of project-relative path → stored mtime (Unix seconds)
// for all rows currently in the artifacts table. Used by Scan to skip unchanged files.
func (idx *Index) loadStoredMtimes() (map[string]int64, error) {
	rows, err := idx.db.Query(`SELECT path, mtime FROM artifacts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]int64)
	for rows.Next() {
		var path string
		var mtime int64
		if err := rows.Scan(&path, &mtime); err != nil {
			return nil, err
		}
		m[path] = mtime
	}
	return m, rows.Err()
}

// Scan walks the lifecycle/ directories and upserts every .md file it finds.
// Files whose mtime (truncated to seconds) matches the value stored in the index
// are skipped, avoiding redundant disk reads and markdown parses on startup.
func (idx *Index) Scan(stages []config.Stage) error {
	lifecycleRoot := filepath.Join(idx.projectRoot, "lifecycle")
	indexed := 0
	skipped := 0
	start := time.Now()

	// Resolve the project root through symlinks once (handles macOS firmlinks).
	resolvedRoot, err := filepath.EvalSymlinks(idx.projectRoot)
	if err != nil {
		resolvedRoot = filepath.Clean(idx.projectRoot)
	}

	// Bulk-load stored mtimes so we can skip unchanged files without per-file DB queries.
	storedMtimes, err := idx.loadStoredMtimes()
	if err != nil {
		slog.Warn("could not load stored mtimes; will re-index all files", "err", err)
		storedMtimes = map[string]int64{}
	}

	for _, stage := range stages {
		dir := filepath.Join(lifecycleRoot, stage.Dir)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
				return err
			}
			if config.ShouldIgnore(path, idx.ignore) {
				return nil
			}
			// d.Info() reuses the DirEntry's cached stat — no extra syscall.
			info, statErr := d.Info()
			if statErr != nil {
				slog.Warn("stat error during scan", "path", path, "err", statErr)
				return nil
			}
			// Compute the project-relative path using the already-resolved root
			// so firmlink paths (e.g. /Users vs /System/Volumes/Data/Users) match
			// what IndexFile stores.
			resolvedFile, ferr := filepath.EvalSymlinks(path)
			if ferr != nil {
				resolvedFile = filepath.Clean(path)
			}
			relPath, rerr := filepath.Rel(resolvedRoot, resolvedFile)
			if rerr == nil {
				relPath = filepath.ToSlash(relPath)
				if stored, ok := storedMtimes[relPath]; ok && stored == info.ModTime().Unix() {
					skipped++
					return nil
				}
			}
			if err := idx.IndexFile(path); err != nil {
				slog.Warn("index file error", "path", path, "err", err)
			}
			indexed++
			return nil
		})
		if walkErr != nil {
			return fmt.Errorf("walking stage %s: %w", stage.Name, walkErr)
		}
	}

	slog.Info("scan complete",
		"indexed", indexed,
		"skipped", skipped,
		"files", indexed+skipped,
		"duration", time.Since(start).Round(time.Millisecond).String(),
	)
	return nil
}

// IndexFile reads, parses, and upserts one file into the index.
// Computes relPath against the symlink-resolved project root so firmlinks
// (e.g. macOS /Users → /System/Volumes/Data/Users) don't produce `..` paths.
func (idx *Index) IndexFile(absPath string) error {
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	// Resolve both sides through EvalSymlinks so the prefix matches.
	resolvedRoot, err := filepath.EvalSymlinks(idx.projectRoot)
	if err != nil {
		resolvedRoot = filepath.Clean(idx.projectRoot)
	}
	resolvedFile, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		resolvedFile = filepath.Clean(absPath)
	}

	relPath, err := filepath.Rel(resolvedRoot, resolvedFile)
	if err != nil {
		return err
	}
	relPath = filepath.ToSlash(relPath)

	if relPath == ".." || strings.HasPrefix(relPath, "../") || filepath.IsAbs(relPath) {
		return fmt.Errorf("refusing to index file outside project root: %s", absPath)
	}
	// Only `.md` files inside lifecycle/ are artifacts. Code files committed
	// by developer agents must not end up in the artifact index.
	if !strings.HasPrefix(relPath, "lifecycle/") || !strings.HasSuffix(relPath, ".md") {
		return fmt.Errorf("refusing to index non-artifact path: %s", relPath)
	}
	// Defence-in-depth: reject ignored files even when called directly from an
	// HTTP handler, bypassing the Scan and watcher pre-filters.
	if config.ShouldIgnore(absPath, idx.ignore) {
		return fmt.Errorf("refusing to index ignored file: %s", relPath)
	}

	a := artifact.Parse(raw, relPath, info.ModTime())

	// Backfill CreatedAt for artifacts that lack a created: frontmatter field.
	// The on-disk file is NOT modified; the derived value is index-only.
	if a.FM.Created == "" {
		if idx.git != nil {
			if t, err := idx.git.FirstCommitDate(relPath); err == nil {
				a.CreatedAt = t
			}
		}
		// Fall back to filesystem mtime if git lookup was unavailable or failed.
		if a.CreatedAt.IsZero() {
			a.CreatedAt = info.ModTime()
		}
	}

	// SHA-256 guard (Milestone 4): if the stored hash matches the incoming
	// content, the file has not meaningfully changed — skip Upsert and
	// applyOpenQuestionTransition to prevent circular re-index loops caused
	// by atomicWrite inside applyOpenQuestionTransition triggering the watcher.
	var storedHash []byte
	if err := idx.db.QueryRow(
		`SELECT body_sha256 FROM artifacts WHERE path = ?`, relPath,
	).Scan(&storedHash); err == nil {
		if bytes.Equal(storedHash, a.SHA256[:]) {
			return nil
		}
	}

	if err := idx.Upsert(a); err != nil {
		return err
	}

	// Auto-transition on open questions (Milestone 3): fires on every
	// successful Upsert, covering watcher events, API writes, and startup scan.
	if idx.hub != nil && idx.wf != nil {
		if err := idx.applyOpenQuestionTransition(a, absPath); err != nil {
			slog.Warn("auto-transition failed", "path", absPath, "err", err)
		}
	}
	return nil
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

	// Parse the created timestamp from frontmatter if present.
	var createdUnix int64
	if a.FM.Created != "" {
		if t, err := time.Parse(time.RFC3339, a.FM.Created); err == nil {
			createdUnix = t.Unix()
		} else if t, err := time.Parse("2006-01-02", a.FM.Created); err == nil {
			// Plain date (no timezone): treat as midnight in the server's local timezone.
			localMidnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Now().Location())
			createdUnix = localMidnight.Unix()
			slog.Warn("artifact has plain-date created field; normalising to RFC3339 for index",
				"path", a.Path, "created", a.FM.Created)
		}
	}
	if createdUnix == 0 && !a.CreatedAt.IsZero() {
		createdUnix = a.CreatedAt.Unix()
	}

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO artifacts
			(path, slug, lineage, idx, stage, type, status, title, priority, frontmatter_json, body_sha256, mtime, created)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.Path, a.Slug, a.FM.Lineage, a.Index, a.Stage,
		a.FM.Type, a.FM.Status, a.FM.Title, a.FM.Priority,
		string(fmJSON), a.SHA256[:], a.Mtime.Unix(), createdUnix,
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
	Stage     string
	Status    string
	Label     string
	Lineage   string
	Type      string
	Priority  string
	Q         string // free-text substring match across title, slug, lineage, type, status
	// Release filters to artifacts whose frontmatter release field equals this value.
	// The special value "__unassigned__" matches artifacts with a null or empty release field.
	Release   string
	Limit     int
	Offset    int
	Unlimited bool // when true, no LIMIT is applied (returns all matching rows)
}

func (f *Filter) withDefaults() Filter {
	out := *f
	if out.Unlimited {
		return out
	}
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
	Path     string               `json:"path"`
	Slug     string               `json:"slug"`
	Lineage  string               `json:"lineage"`
	Index    int                  `json:"index"`
	Stage    string               `json:"stage"`
	Type     string               `json:"type"`
	Status   string               `json:"status"`
	Title    string               `json:"title"`
	FM       artifact.Frontmatter `json:"frontmatter"`
	Mtime    time.Time            `json:"mtime"`
	Created  time.Time            `json:"created"`
}

// List returns a filtered, paginated list of artifacts and the total matching count.
// When f.Unlimited is true all matching rows are returned regardless of Limit/Offset.
func (idx *Index) List(f Filter) ([]*ArtifactRow, int, error) {
	f = f.withDefaults()
	where, args := buildWhere(f)

	var total int
	row := idx.db.QueryRow("SELECT COUNT(*) FROM artifacts"+where, args...)
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	const sel = `SELECT path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime, created
		 FROM artifacts`
	var rows *sql.Rows
	var err error
	if f.Unlimited {
		rows, err = idx.db.Query(sel+where+` ORDER BY lineage, idx, path`, args...)
	} else {
		rows, err = idx.db.Query(sel+where+` ORDER BY lineage, idx, path LIMIT ? OFFSET ?`,
			append(args, f.Limit, f.Offset)...)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanRows(rows)
}

// Get returns a single artifact by project-relative path, or nil if not found.
func (idx *Index) Get(relPath string) (*ArtifactRow, error) {
	rows, err := idx.db.Query(
		`SELECT path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime, created
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
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Stage    string   `json:"stage"`
	Lineage  string   `json:"lineage"`
	Slug     string   `json:"slug"`
	Index    int      `json:"index"`
	Priority string   `json:"priority,omitempty"`
	Labels   []string `json:"labels"`
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
		`SELECT path, slug, lineage, idx, stage, type, status, title, priority FROM artifacts`+where+
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
		if err := rows.Scan(&n.ID, &n.Slug, &n.Lineage, &n.Index, &n.Stage, &n.Type, &n.Status, &n.Title, &n.Priority); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
		nodeSet[n.ID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Populate labels for each node from labels_index.
	if len(nodes) > 0 {
		labelMap := map[string][]string{}
		placeholders := make([]string, len(nodes))
		labelArgs := make([]any, len(nodes))
		for i, n := range nodes {
			placeholders[i] = "?"
			labelArgs[i] = n.ID
		}
		lq := `SELECT artifact, label FROM labels_index WHERE artifact IN (` +
			strings.Join(placeholders, ",") + `) ORDER BY artifact, label`
		lrows, err := idx.db.Query(lq, labelArgs...)
		if err != nil {
			return nil, err
		}
		defer lrows.Close()
		for lrows.Next() {
			var art, lbl string
			if err := lrows.Scan(&art, &lbl); err != nil {
				return nil, err
			}
			labelMap[art] = append(labelMap[art], lbl)
		}
		if err := lrows.Err(); err != nil {
			return nil, err
		}
		for _, n := range nodes {
			if lbls, ok := labelMap[n.ID]; ok {
				n.Labels = lbls
			} else {
				n.Labels = []string{}
			}
		}
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

// Priorities returns all distinct non-empty priority values across the project.
func (idx *Index) Priorities() ([]string, error) {
	rows, err := idx.db.Query(
		`SELECT DISTINCT priority FROM artifacts WHERE priority != '' ORDER BY priority`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
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

// ListByLineage returns all artifacts for the given lineage slug.
// When slug is empty, all artifacts across every lineage are returned.
// Results are ordered by lineage, index, then path — matching the List ordering.
func (idx *Index) ListByLineage(slug string) ([]*ArtifactRow, error) {
	rows, _, err := idx.List(Filter{Lineage: slug, Unlimited: true})
	return rows, err
}

// ListAllGroupedByLineage returns all artifacts grouped by their lineage slug.
// It is equivalent to calling ListByLineage("") and partitioning the result.
func (idx *Index) ListAllGroupedByLineage() (map[string][]*ArtifactRow, error) {
	all, err := idx.ListByLineage("")
	if err != nil {
		return nil, err
	}
	grouped := make(map[string][]*ArtifactRow, len(all))
	for _, r := range all {
		grouped[r.Lineage] = append(grouped[r.Lineage], r)
	}
	return grouped, nil
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
// When limit <= 0 all matching runs are returned (no server-side truncation).
func (idx *Index) ListAgentRuns(status string, limit int) ([]*AgentRunRow, error) {
	const sel = `SELECT run_id, agent_name, role, target_path, started_at, finished_at, status, exit_code, stderr_tail, artifacts_produced_json
			 FROM agent_runs`
	var rows *sql.Rows
	var err error
	if limit > 0 {
		if status != "" {
			rows, err = idx.db.Query(sel+` WHERE status = ? ORDER BY started_at DESC LIMIT ?`, status, limit)
		} else {
			rows, err = idx.db.Query(sel+` ORDER BY started_at DESC LIMIT ?`, limit)
		}
	} else {
		if status != "" {
			rows, err = idx.db.Query(sel+` WHERE status = ? ORDER BY started_at DESC`, status)
		} else {
			rows, err = idx.db.Query(sel + ` ORDER BY started_at DESC`)
		}
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

// ListAgentRunsByTargetPath returns all runs whose target_path matches the given path, newest first.
func (idx *Index) ListAgentRunsByTargetPath(targetPath string) ([]*AgentRunRow, error) {
	rows, err := idx.db.Query(
		`SELECT run_id, agent_name, role, target_path, started_at, finished_at, status, exit_code, stderr_tail, artifacts_produced_json
		 FROM agent_runs WHERE target_path = ? ORDER BY started_at DESC`, targetPath,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*AgentRunRow, 0)
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
	rows, err := idx.db.Query(`SELECT lineage FROM lineage_locks WHERE last_heartbeat <= ?`, cutoff)
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
	// agent_runs is not a cache — it cannot be rebuilt from disk. Exclude it here
	// so run history survives schema rebuilds. ensureAgentRunsTable creates it
	// when it doesn't exist yet.
	stmts := []string{
		`DROP TABLE IF EXISTS schema_version`,
		`DROP TABLE IF EXISTS artifacts`,
		`DROP TABLE IF EXISTS links`,
		`DROP TABLE IF EXISTS labels_index`,
		`DROP TABLE IF EXISTS lineage_locks`,
		`DROP TABLE IF EXISTS parse_errors`,
		`DROP TABLE IF EXISTS releases`,
	}
	for _, s := range stmts {
		if _, err := idx.db.Exec(s); err != nil {
			return fmt.Errorf("drop: %w", err)
		}
	}
	return idx.createSchema()
}

// ensureAgentRunsTable creates the agent_runs table if it doesn't already exist.
// It is called unconditionally on Open so the table survives schema rebuilds.
func (idx *Index) ensureAgentRunsTable() error {
	_, err := idx.db.Exec(`CREATE TABLE IF NOT EXISTS agent_runs (
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
	)`)
	if err != nil {
		return err
	}
	_, err = idx.db.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_runs_target_path ON agent_runs(target_path)`)
	return err
}

// ensureEventsTable creates the events table and its indices if they don't already exist.
// It is called unconditionally on Open so the table survives schema rebuilds.
func (idx *Index) ensureEventsTable() error {
	_, err := idx.db.Exec(`CREATE TABLE IF NOT EXISTS events (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type      TEXT NOT NULL,
		timestamp       INTEGER NOT NULL,
		actor           TEXT NOT NULL,
		artifact_path   TEXT,
		run_id          TEXT,
		summary         TEXT NOT NULL,
		payload_json    TEXT
	)`)
	if err != nil {
		return err
	}
	if _, err := idx.db.Exec(`CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp DESC)`); err != nil {
		return err
	}
	_, err = idx.db.Exec(`CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type)`)
	return err
}

// DB returns the underlying *sql.DB so packages such as scheduler can share the
// same database connection without opening a second WAL connection.
func (idx *Index) DB() *sql.DB { return idx.db }

// ensureSchedulerTables creates the scheduler_jobs and scheduler_runs tables if
// they don't already exist. Called unconditionally on Open so they survive
// schema-version cache rebuilds.
func (idx *Index) ensureSchedulerTables() error {
	_, err := idx.db.Exec(`CREATE TABLE IF NOT EXISTS scheduler_jobs (
		name                TEXT PRIMARY KEY,
		target_type         TEXT NOT NULL,
		target              TEXT NOT NULL,
		args_json           TEXT,
		schedule            TEXT NOT NULL,
		preconditions_json  TEXT,
		enabled             INTEGER NOT NULL DEFAULT 1,
		priority            INTEGER NOT NULL DEFAULT 5,
		timeout_sec         INTEGER NOT NULL,
		created_at          TEXT NOT NULL,
		updated_at          TEXT NOT NULL
	)`)
	if err != nil {
		return err
	}
	_, err = idx.db.Exec(`CREATE TABLE IF NOT EXISTS scheduler_runs (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		job_name    TEXT NOT NULL REFERENCES scheduler_jobs(name) ON DELETE CASCADE,
		start_time  TEXT NOT NULL,
		end_time    TEXT,
		status      TEXT NOT NULL,
		log_path    TEXT,
		created_at  TEXT NOT NULL
	)`)
	if err != nil {
		return err
	}
	_, err = idx.db.Exec(`CREATE INDEX IF NOT EXISTS idx_runs_job_start ON scheduler_runs(job_name, start_time DESC)`)
	return err
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
    priority          TEXT NOT NULL DEFAULT '',
    frontmatter_json  TEXT NOT NULL,
    body_sha256       BLOB NOT NULL,
    mtime             INTEGER NOT NULL,
    created           INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_artifacts_lineage  ON artifacts(lineage);
CREATE INDEX idx_artifacts_stage    ON artifacts(stage);
CREATE INDEX idx_artifacts_status   ON artifacts(status);
CREATE INDEX idx_artifacts_slug     ON artifacts(slug);
CREATE INDEX idx_artifacts_type     ON artifacts(type);
CREATE INDEX idx_artifacts_priority ON artifacts(priority);

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


CREATE TABLE lineage_locks (
    lineage         TEXT PRIMARY KEY,
    holder          TEXT NOT NULL,
    kind            TEXT NOT NULL,
    acquired_at     INTEGER NOT NULL,
    last_heartbeat  INTEGER NOT NULL
);

CREATE TABLE releases (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id  TEXT NOT NULL,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'planned',
    start_date  TEXT,
    end_date    TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    UNIQUE(project_id, name)
);

CREATE INDEX idx_artifacts_release ON artifacts(json_extract(frontmatter_json, '$.release'));
`
	_, err := idx.db.Exec(ddl)
	return err
}

// ----- helpers -----

// escapeLike escapes SQLite LIKE special characters (%, _) in s so they
// match literally. The caller must use ESCAPE '\' in the SQL expression.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}

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
	if f.Priority != "" {
		conds = append(conds, "priority = ?")
		args = append(args, f.Priority)
	}
	if f.Q != "" {
		pattern := "%" + escapeLike(f.Q) + "%"
		conds = append(conds,
			`(title LIKE ? ESCAPE '\' OR slug LIKE ? ESCAPE '\' OR lineage LIKE ? ESCAPE '\' OR type LIKE ? ESCAPE '\' OR status LIKE ? ESCAPE '\')`,
		)
		args = append(args, pattern, pattern, pattern, pattern, pattern)
	}
	if f.Release == "__unassigned__" {
		conds = append(conds,
			`(json_extract(frontmatter_json, '$.release') IS NULL OR json_extract(frontmatter_json, '$.release') = '')`,
		)
	} else if f.Release != "" {
		conds = append(conds, "json_extract(frontmatter_json, '$.release') = ?")
		args = append(args, f.Release)
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ----- event feed -----

// EventRow is one row from the events table.
type EventRow struct {
	ID           int64   `json:"id"`
	EventType    string  `json:"event_type"`
	Timestamp    int64   `json:"timestamp"`
	Actor        string  `json:"actor"`
	ArtifactPath *string `json:"artifact_path,omitempty"`
	RunID        *string `json:"run_id,omitempty"`
	Summary      string  `json:"summary"`
	PayloadJSON  *string `json:"payload_json,omitempty"`
}

// InsertEvent inserts e into the events table and sets e.ID from LastInsertId.
func (idx *Index) InsertEvent(e *EventRow) error {
	res, err := idx.db.Exec(
		`INSERT INTO events (event_type, timestamp, actor, artifact_path, run_id, summary, payload_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.EventType, e.Timestamp, e.Actor, e.ArtifactPath, e.RunID, e.Summary, e.PayloadJSON,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

// ListEvents returns events in reverse-chronological order.
// beforeID > 0 applies cursor pagination (WHERE id < beforeID).
// types non-empty filters to the given event types.
// limit is capped at 200.
func (idx *Index) ListEvents(limit int, beforeID int64, types []string) ([]*EventRow, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var conds []string
	var args []any

	if beforeID > 0 {
		conds = append(conds, "id < ?")
		args = append(args, beforeID)
	}
	if len(types) > 0 {
		placeholders := make([]string, len(types))
		for i, t := range types {
			placeholders[i] = "?"
			args = append(args, t)
		}
		conds = append(conds, "event_type IN ("+strings.Join(placeholders, ",")+")")
	}

	q := `SELECT id, event_type, timestamp, actor, artifact_path, run_id, summary, payload_json FROM events`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := idx.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*EventRow
	for rows.Next() {
		var e EventRow
		if err := rows.Scan(&e.ID, &e.EventType, &e.Timestamp, &e.Actor, &e.ArtifactPath, &e.RunID, &e.Summary, &e.PayloadJSON); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

// PruneEvents deletes events older than maxAgeDays days OR exceeding maxCount total
// (keeping the newest), whichever removes more rows. Both deletions run in a transaction.
func (idx *Index) PruneEvents(maxAgeDays int, maxCount int) error {
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays).Unix()

	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	// Delete rows older than cutoff.
	if _, err := tx.Exec(`DELETE FROM events WHERE timestamp < ?`, cutoff); err != nil {
		return err
	}

	// Trim to maxCount most recent rows.
	if _, err := tx.Exec(
		`DELETE FROM events WHERE id NOT IN (SELECT id FROM events ORDER BY id DESC LIMIT ?)`,
		maxCount,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// ----- dashboard -----

// DashboardStatsRow holds summary counts returned by the dashboard stats endpoint.
type DashboardStatsRow struct {
	TotalTickets      int `json:"total_tickets"`
	InProgress        int `json:"in_progress"`
	Blocked           int `json:"blocked"`
	CompletedThisWeek int `json:"completed_this_week"`
}

// DashboardStats returns summary ticket counts for the dashboard.
// sinceTime is the start of the current ISO week, used to compute
// completed_this_week (tickets that had a done-transition since that instant).
func (idx *Index) DashboardStats(sinceTime time.Time) (*DashboardStatsRow, error) {
	row := &DashboardStatsRow{}

	if err := idx.db.QueryRow(
		`SELECT COUNT(*) FROM artifacts WHERE type='ticket' AND status != 'abandoned'`,
	).Scan(&row.TotalTickets); err != nil {
		return nil, fmt.Errorf("counting total tickets: %w", err)
	}

	if err := idx.db.QueryRow(
		`SELECT COUNT(*) FROM artifacts WHERE type='ticket' AND status = 'in-development'`,
	).Scan(&row.InProgress); err != nil {
		return nil, fmt.Errorf("counting in-progress tickets: %w", err)
	}

	if err := idx.db.QueryRow(
		`SELECT COUNT(*) FROM artifacts WHERE type='ticket' AND status IN ('blocked', 'clarifying')`,
	).Scan(&row.Blocked); err != nil {
		return nil, fmt.Errorf("counting blocked tickets: %w", err)
	}

	if err := idx.db.QueryRow(
		`SELECT COUNT(DISTINCT artifact_path) FROM events
		 WHERE event_type = 'status_transition'
		 AND summary LIKE '%→ done%'
		 AND timestamp >= ?
		 AND artifact_path IN (SELECT path FROM artifacts WHERE type='ticket')`,
		sinceTime.Unix(),
	).Scan(&row.CompletedThisWeek); err != nil {
		return nil, fmt.Errorf("counting completed this week: %w", err)
	}

	return row, nil
}

// StatusCount holds one status and its ticket count for the distribution endpoint.
type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// StatusDistribution returns ticket counts grouped by status, excluding
// tickets with status "done" or "abandoned". Returns an empty (non-nil)
// slice when no matching tickets exist.
func (idx *Index) StatusDistribution() ([]StatusCount, error) {
	rows, err := idx.db.Query(
		`SELECT status, COUNT(*) FROM artifacts
		 WHERE type='ticket' AND status NOT IN ('done','abandoned')
		 GROUP BY status
		 ORDER BY status`,
	)
	if err != nil {
		return nil, fmt.Errorf("querying status distribution: %w", err)
	}
	defer rows.Close()

	out := []StatusCount{}
	for rows.Next() {
		var sc StatusCount
		if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// VelocityBucket holds a time period label and the count of completions in it.
type VelocityBucket struct {
	Period string `json:"period"`
	Count  int    `json:"count"`
}

// CompletionVelocity returns time-bucketed counts of artifacts that
// transitioned to "done" within the lookback window.
//
// granularity must be "daily", "weekly", or "monthly"; any other value
// is coerced to "weekly".
// days controls the lookback window (default 90, capped at 365).
// All periods within the window are included even when count is zero.
func (idx *Index) CompletionVelocity(granularity string, days int) ([]VelocityBucket, error) {
	switch granularity {
	case "daily", "weekly", "monthly":
	default:
		granularity = "weekly"
	}
	if days <= 0 {
		days = 90
	}
	if days > 365 {
		days = 365
	}

	since := time.Now().AddDate(0, 0, -days)
	rows, err := idx.db.Query(
		`SELECT timestamp FROM events
		 WHERE event_type = 'status_transition'
		 AND summary LIKE '%→ done%'
		 AND timestamp >= ?
		 ORDER BY timestamp ASC`,
		since.Unix(),
	)
	if err != nil {
		return nil, fmt.Errorf("querying velocity events: %w", err)
	}
	defer rows.Close()

	var timestamps []time.Time
	for rows.Next() {
		var ts int64
		if err := rows.Scan(&ts); err != nil {
			return nil, err
		}
		timestamps = append(timestamps, time.Unix(ts, 0))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build all period keys in range, then count events per period.
	periods := velocityPeriods(granularity, since, time.Now())
	counts := make(map[string]int, len(periods))
	for _, p := range periods {
		counts[p] = 0
	}
	for _, ts := range timestamps {
		key := velocityPeriodKey(granularity, ts)
		if _, ok := counts[key]; ok {
			counts[key]++
		}
	}

	result := make([]VelocityBucket, len(periods))
	for i, p := range periods {
		result[i] = VelocityBucket{Period: p, Count: counts[p]}
	}
	return result, nil
}

// velocityPeriods returns the ordered list of period keys from since to now
// (inclusive) for the given granularity.
func velocityPeriods(granularity string, since, now time.Time) []string {
	var periods []string
	switch granularity {
	case "daily":
		cur := time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, since.Location())
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		for !cur.After(end) {
			periods = append(periods, cur.Format("2006-01-02"))
			cur = cur.AddDate(0, 0, 1)
		}
	case "weekly":
		cur := isoWeekMonday(since)
		end := isoWeekMonday(now)
		for !cur.After(end) {
			year, week := cur.ISOWeek()
			periods = append(periods, fmt.Sprintf("%04d-W%02d", year, week))
			cur = cur.AddDate(0, 0, 7)
		}
	case "monthly":
		cur := time.Date(since.Year(), since.Month(), 1, 0, 0, 0, 0, since.Location())
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		for !cur.After(end) {
			periods = append(periods, cur.Format("2006-01"))
			cur = cur.AddDate(0, 1, 0)
		}
	}
	return periods
}

// velocityPeriodKey returns the bucket label for a given timestamp under the
// requested granularity.
func velocityPeriodKey(granularity string, t time.Time) string {
	switch granularity {
	case "daily":
		return t.Format("2006-01-02")
	case "monthly":
		return t.Format("2006-01")
	default: // weekly
		year, week := t.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week)
	}
}

// isoWeekMonday returns the Monday of the ISO week that contains t,
// truncated to midnight in t's location.
func isoWeekMonday(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7 in ISO 8601
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, t.Location())
}

// ScanArtifactRows scans a *sql.Rows result set of the standard artifact
// projection (path, slug, lineage, idx, stage, type, status, title,
// frontmatter_json, mtime, created) into []*ArtifactRow. It is exported so
// other packages (e.g. internal/release) can reuse the scan logic.
func ScanArtifactRows(rows *sql.Rows) ([]*ArtifactRow, error) {
	out, _, err := scanRows(rows)
	return out, err
}

func scanRows(rows *sql.Rows) ([]*ArtifactRow, int, error) {
	var out []*ArtifactRow
	for rows.Next() {
		var r ArtifactRow
		var fmJSON string
		var mtimeUnix int64
		var createdUnix int64
		if err := rows.Scan(
			&r.Path, &r.Slug, &r.Lineage, &r.Index, &r.Stage,
			&r.Type, &r.Status, &r.Title, &fmJSON, &mtimeUnix, &createdUnix,
		); err != nil {
			return nil, 0, err
		}
		r.Mtime = time.Unix(mtimeUnix, 0)
		if createdUnix != 0 {
			r.Created = time.Unix(createdUnix, 0)
		}
		_ = json.Unmarshal([]byte(fmJSON), &r.FM)
		out = append(out, &r)
	}
	return out, len(out), rows.Err()
}
