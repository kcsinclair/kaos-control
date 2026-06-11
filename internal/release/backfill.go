// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
)

// BackfillResult counts the outcome of a Backfill run.
type BackfillResult struct {
	Written int      `json:"written"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

// Backfill writes one markdown file per DB row to lifecycle/releases/ when the
// directory is missing or empty (DR-5). A second call is idempotent because
// DiskSync.Write overwrites any existing file. Failure to write a file is
// logged as ERROR and does not block the overall operation; the project will
// still load even if backfill has errors.
func Backfill(ctx context.Context, store *Store, sync *DiskSync, projectID, projectRoot string) (BackfillResult, error) {
	dir := filepath.Join(projectRoot, "lifecycle", "releases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("backfill: cannot create releases directory", "dir", dir, "err", err)
		return BackfillResult{Errors: []string{err.Error()}}, nil
	}

	releases, err := store.List(projectID)
	if err != nil {
		return BackfillResult{}, err
	}

	var result BackfillResult
	for _, r := range releases {
		if ctx.Err() != nil {
			break
		}
		if r.Slug == "" {
			result.Skipped++
			continue
		}
		if _, err := sync.Write(projectRoot, r); err != nil {
			result.Errors = append(result.Errors, r.Slug+": "+err.Error())
			slog.Error("backfill: failed to write release file", "slug", r.Slug, "err", err)
			result.Skipped++
			continue
		}
		result.Written++
	}
	return result, nil
}
