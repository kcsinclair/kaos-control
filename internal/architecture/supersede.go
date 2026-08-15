// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// Supersede marks the ADR at priorADRRel as status: superseded and appends a
// "Superseded by" pointer to newADRRel (FR-16). Both paths are repo-relative
// (e.g. "lifecycle/architecture/decisions/adr-0001-adopt-modular-monolith.md").
func Supersede(projectRoot, priorADRRel, newADRRel string) error {
	absPath := filepath.Join(projectRoot, filepath.FromSlash(priorADRRel))
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading prior ADR %q: %w", priorADRRel, err)
	}

	patched, ok := artifact.PatchFrontmatterField(raw, "status", "superseded")
	if !ok {
		patched, ok = artifact.EnsureFrontmatterField(raw, "status", "superseded")
		if !ok {
			return fmt.Errorf("prior ADR %q has no frontmatter fence to patch", priorADRRel)
		}
	}

	pointer := fmt.Sprintf("\n**Superseded by:** [%s](%s)\n", filepath.Base(newADRRel), filepath.Base(newADRRel))
	content := append(patched, []byte(pointer)...)

	if werr := writeAtomic(absPath, content); werr != nil {
		return fmt.Errorf("writing superseded ADR %q: %w", priorADRRel, werr)
	}
	return nil
}

// SelectionChanged reports whether req names a different architecture or
// tech-stack catalog source than what is currently promoted at the
// lifecycle/architecture/ root, by comparing against the promoted root
// copies' parent: stamps (which Promote sets to the catalog source path).
// A project with nothing promoted yet reports true for both — a first run
// is always "changed" relative to no selection.
func SelectionChanged(projectRoot string, req PromotionRequest) (bool, error) {
	archChanged, err := oneSelectionChanged(projectRoot, req.ArchitectureCatalogPath, "architecture")
	if err != nil {
		return false, err
	}
	stackChanged, err := oneSelectionChanged(projectRoot, req.TechStackCatalogPath, "tech-stack")
	if err != nil {
		return false, err
	}
	return archChanged || stackChanged, nil
}

func oneSelectionChanged(projectRoot, sourceRel, kind string) (bool, error) {
	promoted, err := currentlyPromoted(projectRoot, kind)
	if err != nil {
		return false, err
	}
	if len(promoted) == 0 {
		return true, nil
	}

	sourceRepoPath := path.Join(architectureDir, filepath.ToSlash(sourceRel))
	for _, p := range promoted {
		absP := filepath.Join(projectRoot, filepath.FromSlash(p))
		raw, rerr := os.ReadFile(absP)
		if rerr != nil {
			continue
		}
		a := artifact.Parse(raw, p, time.Time{})
		if a.FM.Parent == sourceRepoPath {
			return false, nil
		}
	}
	return true, nil
}
