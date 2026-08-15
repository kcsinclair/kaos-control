// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
)

// Store provides CRUD operations for releases against SQLite.
// It wraps a *sql.DB shared with the index.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store backed by db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// List returns all releases for the given project, ordered by start_date
// (scheduled first, then unscheduled), then by name.
func (s *Store) List(projectID string) ([]*Release, error) {
	rows, err := s.db.Query(`
		SELECT id, project_id, name, slug, status, start_date, end_date, created_at, updated_at
		FROM releases
		WHERE project_id = ?
		ORDER BY
			CASE WHEN start_date IS NULL THEN 1 ELSE 0 END,
			start_date,
			name
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReleases(rows)
}

// Get returns a single release by project and ID, including idea and defect counts.
// Returns nil, nil when the release is not found.
func (s *Store) Get(projectID string, id int64) (*Release, error) {
	row := s.db.QueryRow(`
		SELECT
			r.id, r.project_id, r.name, r.slug, r.status, r.start_date, r.end_date,
			r.created_at, r.updated_at,
			(SELECT COUNT(*) FROM artifacts
			 WHERE json_extract(frontmatter_json, '$.release') = r.name
			   AND type = 'idea') AS idea_count,
			(SELECT COUNT(*) FROM artifacts
			 WHERE json_extract(frontmatter_json, '$.release') = r.name
			   AND type = 'defect') AS defect_count
		FROM releases r
		WHERE r.project_id = ? AND r.id = ?
	`, projectID, id)

	r, err := scanReleaseWithCounts(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// GetBySlug returns a release by project and slug, or nil if not found.
func (s *Store) GetBySlug(projectID, slug string) (*Release, error) {
	row := s.db.QueryRow(`
		SELECT id, project_id, name, slug, status, start_date, end_date, created_at, updated_at
		FROM releases
		WHERE project_id = ? AND slug = ?
	`, projectID, slug)
	var r Release
	var startDate, endDate sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(
		&r.ID, &r.ProjectID, &r.Name, &r.Slug, &r.Status,
		&startDate, &endDate, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.StartDate = parseDate(startDate)
	r.EndDate = parseDate(endDate)
	r.CreatedAt = mustParseTime(createdAt)
	r.UpdatedAt = mustParseTime(updatedAt)
	r.FilePath = relPath(r.Slug)
	return &r, nil
}

// GetByName returns a release by project and name, or nil if not found.
func (s *Store) GetByName(projectID, name string) (*Release, error) {
	row := s.db.QueryRow(`
		SELECT id, project_id, name, slug, status, start_date, end_date, created_at, updated_at
		FROM releases
		WHERE project_id = ? AND name = ?
	`, projectID, name)

	var r Release
	var startDate, endDate sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(
		&r.ID, &r.ProjectID, &r.Name, &r.Slug, &r.Status,
		&startDate, &endDate, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.StartDate = parseDate(startDate)
	r.EndDate = parseDate(endDate)
	r.CreatedAt = mustParseTime(createdAt)
	r.UpdatedAt = mustParseTime(updatedAt)
	r.FilePath = relPath(r.Slug)
	return &r, nil
}

// Count returns the number of releases for the given project.
func (s *Store) Count(projectID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM releases WHERE project_id = ?`, projectID).Scan(&n)
	return n, err
}

// Create writes the release's markdown file (the authoritative store) and then
// refreshes the SQLite cache from it, setting r.ID, r.Slug, r.FilePath,
// r.CreatedAt, r.UpdatedAt. The markdown file is written first: if the disk
// write fails, the cache is left untouched. Returns ErrConflict if a release
// with the same name or slug already exists within the project.
//
// See requirements/release-artefacts-9.md (DR-1): markdown is authoritative,
// the `releases` table is a rebuildable cache written after disk.
func (s *Store) Create(r *Release, sync *DiskSync, projectRoot string) error {
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now

	slug := Slugify(r.Name)
	if slug == "" {
		slug = fallbackSlug(r.Name)
	}
	r.Slug = slug
	r.FilePath = relPath(slug)

	// Conflict check against the cache before touching disk, so a duplicate
	// create never overwrites an existing release file.
	var existing int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM releases WHERE project_id = ? AND (name = ? OR slug = ?)`,
		r.ProjectID, r.Name, slug,
	).Scan(&existing); err != nil {
		return err
	}
	if existing > 0 {
		return ErrConflict
	}

	// Markdown first (authoritative). A disk failure aborts before any cache write.
	if sync != nil && projectRoot != "" {
		if _, err := sync.Write(projectRoot, r); err != nil {
			return fmt.Errorf("writing release file: %w", err)
		}
	}

	// Refresh the cache from the release we just wrote, then read back the
	// autoincrement id (cache-local identity) for the response.
	if err := s.UpsertBySlug(r); err != nil {
		return err
	}
	if got, err := s.GetBySlug(r.ProjectID, slug); err == nil && got != nil {
		r.ID = got.ID
		r.CreatedAt = got.CreatedAt
	}
	return nil
}

// Update rewrites the release's markdown file (renaming it if the slug changed)
// and then refreshes the SQLite cache row by id. Markdown is written first; if
// the disk write fails, the cache is left untouched and the error is returned.
// Returns the old name (before the update) so the caller can trigger rename
// propagation. See release-artefacts-9.md (DR-1).
func (s *Store) Update(r *Release, sync *DiskSync, projectRoot string) (string, error) {
	// Fetch the current name and slug first.
	var oldName, oldSlug string
	err := s.db.QueryRow(
		`SELECT name, slug FROM releases WHERE project_id = ? AND id = ?`, r.ProjectID, r.ID,
	).Scan(&oldName, &oldSlug)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	r.UpdatedAt = now

	newSlug := Slugify(r.Name)
	if newSlug == "" {
		// Emoji-only name: keep the existing slug if there is one (avoids churn),
		// otherwise derive a stable, id-independent fallback.
		if oldSlug != "" {
			newSlug = oldSlug
		} else {
			newSlug = fallbackSlug(r.Name)
		}
	}
	r.Slug = newSlug
	r.FilePath = relPath(newSlug)

	// Markdown first (authoritative). A disk failure aborts before any cache write.
	if sync != nil && projectRoot != "" {
		if oldSlug != newSlug {
			if _, err := sync.Rename(projectRoot, oldSlug, newSlug, r); err != nil {
				return "", fmt.Errorf("renaming release file: %w", err)
			}
		} else {
			if _, err := sync.Write(projectRoot, r); err != nil {
				return "", fmt.Errorf("writing release file: %w", err)
			}
		}
	}

	// Refresh the cache row in place by id (cache-local identity survives a
	// name/slug change, which an ON CONFLICT(name) upsert would not).
	if _, err := s.db.Exec(`
		UPDATE releases
		SET name = ?, slug = ?, status = ?, start_date = ?, end_date = ?, updated_at = ?
		WHERE project_id = ? AND id = ?
	`,
		r.Name, newSlug, r.Status,
		formatDate(r.StartDate), formatDate(r.EndDate),
		now.Format(time.RFC3339),
		r.ProjectID, r.ID,
	); err != nil {
		return "", err
	}
	return oldName, nil
}

// UpsertBySlug inserts or updates a release identified by projectID+slug.
// Used by the watcher to mirror file-system changes into the DB.
func (s *Store) UpsertBySlug(r *Release) error {
	now := time.Now().UTC()
	_, err := s.db.Exec(`
		INSERT INTO releases (project_id, name, slug, status, start_date, end_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, name) DO UPDATE SET
			slug       = excluded.slug,
			status     = excluded.status,
			start_date = excluded.start_date,
			end_date   = excluded.end_date,
			updated_at = excluded.updated_at
	`,
		r.ProjectID, r.Name, r.Slug, r.Status,
		formatDate(r.StartDate), formatDate(r.EndDate),
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	return err
}

// DeleteBySlug removes the release with the given slug from the project.
// Returns without error if the release does not exist.
func (s *Store) DeleteBySlug(projectID, slug string) error {
	_, err := s.db.Exec(
		`DELETE FROM releases WHERE project_id = ? AND slug = ?`, projectID, slug,
	)
	return err
}

// PruneExcept deletes every release for the project whose slug is not in keep.
// Used by Rehydrate to drop cache rows whose markdown file no longer exists on
// disk (e.g. a release file deleted while the server was not running), so the
// cache reflects disk. A nil/empty keep set deletes all releases for the project.
func (s *Store) PruneExcept(projectID string, keep map[string]struct{}) (int, error) {
	rows, err := s.db.Query(`SELECT slug FROM releases WHERE project_id = ?`, projectID)
	if err != nil {
		return 0, err
	}
	var toDelete []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			rows.Close()
			return 0, err
		}
		if _, ok := keep[slug]; !ok {
			toDelete = append(toDelete, slug)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	for _, slug := range toDelete {
		if _, err := s.db.Exec(
			`DELETE FROM releases WHERE project_id = ? AND slug = ?`, projectID, slug,
		); err != nil {
			return 0, err
		}
	}
	return len(toDelete), nil
}

// Delete removes the release and returns its name and the count of artifacts
// that referenced it (for use in warnings or reassignment).
// When sync is non-nil the markdown file is also removed from disk.
func (s *Store) Delete(projectID string, id int64, sync *DiskSync, projectRoot string) (string, int, error) {
	var name, slug string
	err := s.db.QueryRow(
		`SELECT name, slug FROM releases WHERE project_id = ? AND id = ?`, projectID, id,
	).Scan(&name, &slug)
	if err == sql.ErrNoRows {
		return "", 0, ErrNotFound
	}
	if err != nil {
		return "", 0, err
	}

	// Count artifacts that reference this release by name.
	var count int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM artifacts
		 WHERE json_extract(frontmatter_json, '$.release') = ?`, name,
	).Scan(&count); err != nil {
		return "", 0, err
	}

	// Markdown first (authoritative): remove the file, then the cache row. If the
	// file cannot be removed, abort before touching the cache — otherwise the
	// next rehydrate would resurrect the release from the surviving file.
	if sync != nil && projectRoot != "" && slug != "" {
		if err := sync.Delete(projectRoot, slug); err != nil {
			return "", 0, fmt.Errorf("removing release file: %w", err)
		}
	}

	if _, err := s.db.Exec(
		`DELETE FROM releases WHERE project_id = ? AND id = ?`, projectID, id,
	); err != nil {
		return "", 0, err
	}

	return name, count, nil
}

// fallbackSlug derives a stable, DB-independent slug for a release whose name
// produces an empty slug (e.g. emoji-only). It is deterministic in the name, so
// it needs no autoincrement id and survives a cache rebuild.
func fallbackSlug(name string) string {
	sum := sha256.Sum256([]byte(name))
	return "release-" + hex.EncodeToString(sum[:4])
}

// ListArtifacts returns all indexed artifacts whose release frontmatter field
// matches the name of the release identified by releaseID.
func (s *Store) ListArtifacts(projectID string, releaseID int64) ([]*index.ArtifactRow, error) {
	var name string
	err := s.db.QueryRow(
		`SELECT name FROM releases WHERE project_id = ? AND id = ?`, projectID, releaseID,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("release %d not found in project %q", releaseID, projectID)
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime, created, rel_path
		FROM artifacts
		WHERE json_extract(frontmatter_json, '$.release') = ?
		ORDER BY lineage, idx, path
	`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return index.ScanArtifactRows(rows)
}

// ----- helpers -----

const dateLayout = "2006-01-02"

func formatDate(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.Format(dateLayout), Valid: true}
}

func parseDate(s sql.NullString) *time.Time {
	if !s.Valid || s.String == "" {
		return nil
	}
	t, err := time.Parse(dateLayout, s.String)
	if err != nil {
		return nil
	}
	return &t
}

func mustParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func scanReleases(rows *sql.Rows) ([]*Release, error) {
	var out []*Release
	for rows.Next() {
		var r Release
		var startDate, endDate sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(
			&r.ID, &r.ProjectID, &r.Name, &r.Slug, &r.Status,
			&startDate, &endDate, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		r.StartDate = parseDate(startDate)
		r.EndDate = parseDate(endDate)
		r.CreatedAt = mustParseTime(createdAt)
		r.UpdatedAt = mustParseTime(updatedAt)
		r.FilePath = relPath(r.Slug)
		out = append(out, &r)
	}
	return out, rows.Err()
}

func scanReleaseWithCounts(row *sql.Row) (*Release, error) {
	var r Release
	var startDate, endDate sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(
		&r.ID, &r.ProjectID, &r.Name, &r.Slug, &r.Status,
		&startDate, &endDate, &createdAt, &updatedAt,
		&r.IdeaCount, &r.DefectCount,
	)
	if err != nil {
		return nil, err
	}
	r.StartDate = parseDate(startDate)
	r.EndDate = parseDate(endDate)
	r.CreatedAt = mustParseTime(createdAt)
	r.UpdatedAt = mustParseTime(updatedAt)
	r.FilePath = relPath(r.Slug)
	return &r, nil
}
