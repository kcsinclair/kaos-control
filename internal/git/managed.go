// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"fmt"
	"os"
	"path/filepath"
)

// managedMarkerName is a sentinel written inside .git/ (never tracked or
// committed) recording that kaos-control created and therefore manages this
// repository's git. It is the discriminator between a fresh folder kaos-control
// set up (auto-commit what we generate) and a pre-existing user repo (never
// touch their history — hand back the commands instead).
const managedMarkerName = "kaos-control-managed"

// MarkManaged records that kaos-control created/manages this repo. Call it
// immediately after initialising a new repository (git init), before any
// later scaffolding runs.
func MarkManaged(root string) error {
	marker := filepath.Join(root, ".git", managedMarkerName)
	if err := os.WriteFile(marker, []byte("kaos-control created this repository\n"), 0o644); err != nil {
		return fmt.Errorf("writing managed marker: %w", err)
	}
	return nil
}

// IsManaged reports whether kaos-control created/manages this repo's git.
func IsManaged(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".git", managedMarkerName))
	return err == nil
}

// TrackGenerated commits relPaths when kaos-control manages the repo (a fresh
// one it created), or returns the equivalent `git add`/`git commit` commands
// for the user to run when the repo pre-existed — so a user's own history is
// never modified automatically. Returns committed=false, commands=nil when
// root is not a git repo or relPaths is empty. relPaths are project-root
// relative with forward slashes.
func TrackGenerated(root string, relPaths []string, commitMsg string) (committed bool, commands []string, err error) {
	if len(relPaths) == 0 || !IsRepo(root) {
		return false, nil, nil
	}
	if !IsManaged(root) {
		add := "git -C " + root + " add"
		for _, p := range relPaths {
			add += " " + filepath.ToSlash(p)
		}
		return false, []string{add, fmt.Sprintf("git -C %s commit -m %q", root, commitMsg)}, nil
	}
	repo, err := Open(root)
	if err != nil {
		return false, nil, err
	}
	name, email := repo.ResolveIdentity()
	if _, err := repo.AddAndCommit(relPaths, commitMsg, name, email); err != nil {
		return false, nil, err
	}
	return true, nil, nil
}
