// SPDX-License-Identifier: AGPL-3.0-or-later

// Package architecture implements the reusable promotion and ADR-authoring
// primitives that back lifecycle/architecture/: copying chosen catalog
// artefacts to the directory root and allocating/authoring ADRs. It owns no
// HTTP and no wizard UX — see internal/http/architecture.go for the callers.
package architecture

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// architectureDir is the repo-relative root of the architecture zone.
const architectureDir = "lifecycle/architecture"

// PromotionRequest names the catalog sources to promote. Both paths are
// repo-relative to lifecycle/architecture/ (e.g. "architectures/foo.md",
// "tech-stacks/bar.md").
type PromotionRequest struct {
	ArchitectureCatalogPath string
	TechStackCatalogPath    string
}

// PromotionResult reports where the promoted copies landed and, if a prior
// selection of either kind was replaced, where the old copies were archived.
type PromotionResult struct {
	PromotedArchitecture string
	PromotedTechStack    string
	Archived             []string
}

// Promote copies the chosen architecture and tech-stack catalog artefacts to
// the lifecycle/architecture/ root, stamping parent: back to their catalog
// source. Re-promoting the same selection is a no-op overwrite (idempotent);
// promoting a different selection archives the previously promoted root copy
// of that kind under lifecycle/architecture/archive/ rather than deleting it.
// Catalog entries are never modified.
func Promote(projectRoot string, req PromotionRequest) (PromotionResult, error) {
	var result PromotionResult

	archDest, archived, err := promoteOne(projectRoot, req.ArchitectureCatalogPath, "architecture")
	if err != nil {
		return PromotionResult{}, err
	}
	result.PromotedArchitecture = archDest
	result.Archived = append(result.Archived, archived...)

	stackDest, archived2, err := promoteOne(projectRoot, req.TechStackCatalogPath, "tech-stack")
	if err != nil {
		return PromotionResult{}, err
	}
	result.PromotedTechStack = stackDest
	result.Archived = append(result.Archived, archived2...)

	return result, nil
}

// promoteOne promotes a single catalog source of the given type (kind is the
// artefact `type:` value, "architecture" or "tech-stack") and returns the
// destination repo-relative path and any prior root copies of the same kind
// that were archived because they pointed at a different source.
func promoteOne(projectRoot, sourceRel, kind string) (destRel string, archived []string, err error) {
	archDir := filepath.Join(projectRoot, filepath.FromSlash(architectureDir))

	absSrc, rerr := sandbox.Resolve(archDir, sourceRel)
	if rerr != nil {
		return "", nil, fmt.Errorf("resolving %s source %q: %w", kind, sourceRel, rerr)
	}
	srcRaw, rerr := os.ReadFile(absSrc)
	if rerr != nil {
		return "", nil, fmt.Errorf("reading %s source %q: %w", kind, sourceRel, rerr)
	}

	basename := filepath.Base(filepath.ToSlash(sourceRel))
	destRel = path.Join(architectureDir, basename)
	absDest := filepath.Join(archDir, basename)
	sourceRepoPath := path.Join(architectureDir, filepath.ToSlash(sourceRel))

	promoted, perr := currentlyPromoted(projectRoot, kind)
	if perr != nil {
		return "", nil, fmt.Errorf("scanning promoted %s copies: %w", kind, perr)
	}

	for _, p := range promoted {
		absP := filepath.Join(projectRoot, filepath.FromSlash(p))
		raw, rerr := os.ReadFile(absP)
		if rerr != nil {
			continue
		}
		a := artifact.Parse(raw, p, time.Time{})
		if a.FM.Parent == sourceRepoPath {
			// Same selection already promoted — nothing to archive, even
			// when it happens to live at destRel (the idempotent case).
			continue
		}
		archivedPath, aerr := archiveFile(projectRoot, p)
		if aerr != nil {
			return "", nil, fmt.Errorf("archiving prior %s copy %q: %w", kind, p, aerr)
		}
		archived = append(archived, archivedPath)
	}

	content, serr := stampParent(srcRaw, sourceRepoPath)
	if serr != nil {
		return "", nil, fmt.Errorf("stamping parent on %s: %w", kind, serr)
	}

	if werr := writeAtomic(absDest, content); werr != nil {
		return "", nil, fmt.Errorf("writing %s: %w", kind, werr)
	}

	return destRel, archived, nil
}

// currentlyPromoted lists root-level lifecycle/architecture/*.md files (never
// descending into archive/ or any other subdirectory) whose `type:` matches kind.
func currentlyPromoted(projectRoot, kind string) ([]string, error) {
	archDir := filepath.Join(projectRoot, filepath.FromSlash(architectureDir))
	entries, err := os.ReadDir(archDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		absPath := filepath.Join(archDir, e.Name())
		raw, rerr := os.ReadFile(absPath)
		if rerr != nil {
			continue
		}
		relPath := path.Join(architectureDir, e.Name())
		a := artifact.Parse(raw, relPath, time.Time{})
		if a.FM.Type == kind {
			out = append(out, relPath)
		}
	}
	return out, nil
}

// archiveFile moves a root-level promoted file into
// lifecycle/architecture/archive/, never overwriting an existing archived
// file — a name collision gets the shortest available numeric disambiguator
// (-1, -2, ...) so archived history is preserved.
func archiveFile(projectRoot, relPath string) (string, error) {
	archiveAbsDir := filepath.Join(projectRoot, filepath.FromSlash(architectureDir), "archive")
	if err := os.MkdirAll(archiveAbsDir, 0o755); err != nil {
		return "", err
	}

	base := filepath.Base(relPath)
	stem := strings.TrimSuffix(base, ".md")
	dest := filepath.Join(archiveAbsDir, base)
	for n := 1; ; n++ {
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			break
		}
		dest = filepath.Join(archiveAbsDir, fmt.Sprintf("%s-%d.md", stem, n))
	}

	src := filepath.Join(projectRoot, filepath.FromSlash(relPath))
	if err := os.Rename(src, dest); err != nil {
		return "", err
	}

	destRel, err := filepath.Rel(projectRoot, dest)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(destRel), nil
}

// stampParent sets parent: on the frontmatter of raw, replacing the value if
// the key is already present or inserting it if absent.
func stampParent(raw []byte, parent string) ([]byte, error) {
	if patched, ok := artifact.PatchFrontmatterField(raw, "parent", parent); ok {
		return patched, nil
	}
	if ensured, ok := artifact.EnsureFrontmatterField(raw, "parent", parent); ok {
		return ensured, nil
	}
	return nil, fmt.Errorf("source has no frontmatter fence to stamp parent into")
}

// writeAtomic writes content to absPath via a temp file + rename in the same
// directory, mirroring the docs-write pattern (internal/docs.Write).
func writeAtomic(absPath string, content []byte) error {
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".architecture-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, werr := tmp.Write(content); werr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return werr
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
