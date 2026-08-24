// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"os"
	"path/filepath"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
)

// PropagateRename updates the `release` frontmatter field on every artifact
// currently assigned to oldName, writes the changes to disk, creates a single
// git commit, and re-indexes all changed paths. Any extraPaths passed (such
// as the release artifact file itself) are staged and committed in the same
// commit.
//
// h may be nil; when non-nil an "artifact.indexed" hub event is broadcast for
// each successfully re-indexed artifact so WebSocket clients observe the rename.
//
// It returns the count of artifact files that were updated.
func PropagateRename(projectRoot, oldName, newName string, idx *index.Index, repo *git.Repo, h *hub.Hub, extraPaths ...string) (int, error) {
	var changed []string
	changed = append(changed, extraPaths...)

	// Find all artifacts assigned to oldName.
	rows, _, err := idx.List(index.Filter{Release: oldName, Unlimited: true})
	if err != nil {
		return 0, err
	}

	renamedCount := 0
	for _, row := range rows {
		absPath := filepath.Join(projectRoot, row.Path)
		raw, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		patched, ok := artifact.PatchFrontmatterField(raw, "release", newName)
		if !ok {
			continue
		}
		if err := os.WriteFile(absPath, patched, 0o644); err != nil {
			continue
		}
		changed = append(changed, row.Path)
		renamedCount++
	}

	if len(changed) == 0 {
		return 0, nil
	}

	// Commit all changes in a single git commit.
	authorName, authorEmail := repo.ResolveIdentity()
	msg := `chore(releases): rename "` + oldName + `" → "` + newName + `"`
	if _, err := repo.AddAndCommit(changed, msg, authorName, authorEmail); err != nil {
		return 0, err
	}

	// Re-index all changed paths and broadcast artifact.indexed for each.
	for _, relPath := range changed {
		absPath := filepath.Join(projectRoot, relPath)
		if err := idx.IndexFile(absPath); err != nil {
			continue
		}
		if h != nil {
			h.Broadcast(hub.Event{
				Type:    "artifact.indexed",
				Payload: map[string]string{"path": relPath, "action": "updated"},
			})
		}
	}

	return renamedCount, nil
}

