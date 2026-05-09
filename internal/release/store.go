// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"database/sql"
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
		SELECT id, project_id, name, status, start_date, end_date, created_at, updated_at
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
			r.id, r.project_id, r.name, r.status, r.start_date, r.end_date,
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

// GetByName returns a release by project and name, or nil if not found.
func (s *Store) GetByName(projectID, name string) (*Release, error) {
	row := s.db.QueryRow(`
		SELECT id, project_id, name, status, start_date, end_date, created_at, updated_at
		FROM releases
		WHERE project_id = ? AND name = ?
	`, projectID, name)

	var r Release
	var startDate, endDate sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(
		&r.ID, &r.ProjectID, &r.Name, &r.Status,
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
	return &r, nil
}

// Create inserts r into the database and sets r.ID, r.CreatedAt, r.UpdatedAt.
// Returns an error if the name is already taken within the project.
func (s *Store) Create(r *Release) error {
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now

	createdAt := now.Format(time.RFC3339)
	updatedAt := now.Format(time.RFC3339)

	res, err := s.db.Exec(`
		INSERT INTO releases (project_id, name, status, start_date, end_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		r.ProjectID, r.Name, r.Status,
		formatDate(r.StartDate), formatDate(r.EndDate),
		createdAt, updatedAt,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	r.ID = id
	return nil
}

// Update saves changes to name, status, and dates; bumps UpdatedAt.
// Returns the old name (before the update) so the caller can trigger rename propagation.
func (s *Store) Update(r *Release) (string, error) {
	// Fetch the current name first.
	var oldName string
	err := s.db.QueryRow(
		`SELECT name FROM releases WHERE project_id = ? AND id = ?`, r.ProjectID, r.ID,
	).Scan(&oldName)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	r.UpdatedAt = now

	_, err = s.db.Exec(`
		UPDATE releases
		SET name = ?, status = ?, start_date = ?, end_date = ?, updated_at = ?
		WHERE project_id = ? AND id = ?
	`,
		r.Name, r.Status,
		formatDate(r.StartDate), formatDate(r.EndDate),
		now.Format(time.RFC3339),
		r.ProjectID, r.ID,
	)
	if err != nil {
		return "", err
	}
	return oldName, nil
}

// Delete removes the release and returns its name and the count of artifacts
// that referenced it (for use in warnings or reassignment).
func (s *Store) Delete(projectID string, id int64) (string, int, error) {
	var name string
	err := s.db.QueryRow(
		`SELECT name FROM releases WHERE project_id = ? AND id = ?`, projectID, id,
	).Scan(&name)
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

	if _, err := s.db.Exec(
		`DELETE FROM releases WHERE project_id = ? AND id = ?`, projectID, id,
	); err != nil {
		return "", 0, err
	}

	return name, count, nil
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
		SELECT path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime, created
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
			&r.ID, &r.ProjectID, &r.Name, &r.Status,
			&startDate, &endDate, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		r.StartDate = parseDate(startDate)
		r.EndDate = parseDate(endDate)
		r.CreatedAt = mustParseTime(createdAt)
		r.UpdatedAt = mustParseTime(updatedAt)
		out = append(out, &r)
	}
	return out, rows.Err()
}

func scanReleaseWithCounts(row *sql.Row) (*Release, error) {
	var r Release
	var startDate, endDate sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(
		&r.ID, &r.ProjectID, &r.Name, &r.Status,
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
	return &r, nil
}
