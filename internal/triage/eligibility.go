// SPDX-License-Identifier: AGPL-3.0-or-later

package triage

import (
	"context"
	"path"

	"github.com/kaos-control/kaos-control/internal/index"
)

// eligible checks whether the artifact at relPath should be triaged.
// Returns the indexed row (non-nil) when eligible.
// Returns (nil, reason, nil) when ineligible with a stable reason string.
// Returns (nil, "", err) on a database error.
func eligible(_ context.Context, idx IndexStore, relPath string) (*index.ArtifactRow, string, error) {
	// Path must be directly inside lifecycle/ideas/ with no subdirectory nesting.
	dir := path.Dir(relPath)
	if dir != "lifecycle/ideas" {
		return nil, "not_in_ideas_dir", nil
	}

	row, err := idx.Get(relPath)
	if err != nil {
		return nil, "", err
	}
	if row == nil {
		return nil, "not_indexed", nil
	}
	if row.Type != "idea" {
		return nil, "wrong_type", nil
	}
	if row.Status != "raw" {
		return nil, "wrong_status", nil
	}
	return row, "", nil
}
