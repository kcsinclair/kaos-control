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
	Pruned   int      `json:"pruned"`
	Errors   []string `json:"errors"`
}

// Rehydrate rebuilds the releases cache from disk: it reads
// lifecycle/releases/*.md and upserts each valid file into the store, then
// prunes any cache rows whose file is no longer on disk. Disk is authoritative,
// so this is safe to run on every project load regardless of whether the cache
// already has rows (release-artefacts-9.md DR-2). A missing releases directory
// means zero releases — the cache is pruned to empty. Invalid files (failing
// DR-1 validation) are skipped with a WARN log and counted in result.Skipped.
func Rehydrate(ctx context.Context, store *Store, projectID, projectRoot string) (RehydrateResult, error) {
	dir := filepath.Join(projectRoot, "lifecycle", "releases")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// No releases on disk → prune the cache to empty so a stale table
			// cannot resurrect releases that no longer exist on disk.
			pruned, perr := store.PruneExcept(projectID, nil)
			return RehydrateResult{Pruned: pruned}, perr
		}
		return RehydrateResult{}, err
	}

	seen := make(map[string]struct{})
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
		seen[f.Slug] = struct{}{}
		result.Inserted++
	}

	// Prune cache rows whose file was removed on disk while the cache persisted.
	pruned, err := store.PruneExcept(projectID, seen)
	if err != nil {
		result.Errors = append(result.Errors, "prune: "+err.Error())
		slog.Warn("rehydrate: prune failed", "project", projectID, "err", err)
	}
	result.Pruned = pruned
	return result, nil
}
