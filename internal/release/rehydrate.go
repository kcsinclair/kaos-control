// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// RehydrateResult counts the outcome of a Rehydrate run.
type RehydrateResult struct {
	Inserted int      `json:"inserted"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// Rehydrate reads lifecycle/releases/*.md from disk and upserts each valid
// file into the store. Invalid files (failing DR-1 validation) are skipped
// with a WARN log and counted in result.Skipped. Duplicate runs are
// idempotent because UpsertBySlug uses ON CONFLICT … DO UPDATE.
func Rehydrate(ctx context.Context, store *Store, projectID, projectRoot string) (RehydrateResult, error) {
	dir := filepath.Join(projectRoot, "lifecycle", "releases")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return RehydrateResult{}, nil
		}
		return RehydrateResult{}, err
	}

	var result RehydrateResult
	for _, de := range entries {
		if ctx.Err() != nil {
			break
		}
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".md") {
			continue
		}

		absPath := filepath.Join(dir, de.Name())
		raw, err := os.ReadFile(absPath)
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, absPath+": "+err.Error())
			slog.Warn("rehydrate: cannot read file", "path", absPath, "err", err)
			continue
		}

		f, err := Parse(de.Name(), raw)
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, absPath+": "+err.Error())
			slog.Warn("rehydrate: skipping invalid release file", "path", absPath, "err", err)
			continue
		}

		r := &Release{
			ProjectID: projectID,
			Name:      f.Title,
			Slug:      f.Slug,
			Status:    f.Status,
			StartDate: f.StartDate,
			EndDate:   f.EndDate,
			UpdatedAt: f.UpdatedAt,
		}
		if err := store.UpsertBySlug(r); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, absPath+": "+err.Error())
			slog.Warn("rehydrate: upsert failed", "path", absPath, "err", err)
			continue
		}
		result.Inserted++
	}
	return result, nil
}
