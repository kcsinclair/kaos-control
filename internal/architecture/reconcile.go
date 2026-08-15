// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

// scaffoldSubdirs are the four subdirectories EnsureArchitectureScaffold
// guarantees exist under lifecycle/architecture/.
var scaffoldSubdirs = []string{"architectures", "tech-stacks", "decisions", "standards"}

// emptyTrackedDirs get a .gitkeep so they are tracked by git despite
// starting with no seeded content — decisions/ and standards/ are filled in
// later, by ADR authoring and by the wizard's post-selection seeding.
var emptyTrackedDirs = map[string]bool{"decisions": true, "standards": true}

// EnsureResult reports the outcome of scaffolding or seeding one directory
// or file under lifecycle/architecture/. Created is false when it already
// existed before the call.
type EnsureResult struct {
	Path    string // project-root-relative, e.g. "lifecycle/architecture/architectures/local-web.md"
	Created bool
}

// EnsureArchitectureScaffold makes sure lifecycle/architecture/ under
// projectRoot has its four standard subdirectories (architectures/,
// tech-stacks/, decisions/, standards/) and every embedded catalog file
// (README.md, architectures/*.md, tech-stacks/*.md). decisions/ and
// standards/ are created empty, tracked via .gitkeep.
//
// It is idempotent and skip-if-exists at the level of each individual
// directory and file, so a partially-populated tree (or a fully-populated
// one, like this repo's own lifecycle/architecture/) is completed rather
// than clobbered, and re-running is a no-op. Safe to call on every project
// `init` and every project `Open`.
func EnsureArchitectureScaffold(projectRoot string) ([]EnsureResult, error) {
	var results []EnsureResult
	archAbs := filepath.Join(projectRoot, filepath.FromSlash(architectureDir))

	for _, sub := range scaffoldSubdirs {
		absDir := filepath.Join(archAbs, sub)
		existed := isDir(absDir)
		if err := os.MkdirAll(absDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating %s directory: %w", sub, err)
		}
		if !existed {
			results = append(results, EnsureResult{Path: path.Join(architectureDir, sub), Created: true})
		}

		if !emptyTrackedDirs[sub] {
			continue
		}
		gitkeep := filepath.Join(absDir, ".gitkeep")
		if _, err := os.Stat(gitkeep); err == nil {
			continue
		}
		if err := os.WriteFile(gitkeep, []byte{}, 0o644); err != nil {
			return nil, fmt.Errorf("writing .gitkeep in %s: %w", sub, err)
		}
		results = append(results, EnsureResult{Path: path.Join(architectureDir, sub, ".gitkeep"), Created: true})
	}

	seedResults, err := seedCatalogFiles(archAbs)
	if err != nil {
		return nil, fmt.Errorf("seeding architecture catalog: %w", err)
	}
	results = append(results, seedResults...)

	return results, nil
}

// seedCatalogFiles writes every embedded catalog file (README.md,
// architectures/*.md, tech-stacks/*.md) under archAbs that is not already
// present on disk.
func seedCatalogFiles(archAbs string) ([]EnsureResult, error) {
	var results []EnsureResult

	err := fs.WalkDir(catalogfs.FS, ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		absPath := filepath.Join(archAbs, filepath.FromSlash(rel))
		if _, statErr := os.Stat(absPath); statErr == nil {
			return nil // already present, skip
		}

		content, rerr := fs.ReadFile(catalogfs.FS, rel)
		if rerr != nil {
			return fmt.Errorf("reading embedded %s: %w", rel, rerr)
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(absPath, content, 0o644); err != nil {
			return err
		}

		results = append(results, EnsureResult{Path: path.Join(architectureDir, rel), Created: true})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// isDir reports whether path exists and is a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
