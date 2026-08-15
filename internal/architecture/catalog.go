// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// CatalogItem is one architecture or tech-stack catalog entry, as read from
// lifecycle/architecture/architectures/*.md or tech-stacks/*.md.
type CatalogItem struct {
	Path      string // repo-relative, e.g. "lifecycle/architecture/architectures/modular-monolith.md"
	Slug      string // filename stem, e.g. "modular-monolith"
	Title     string
	Summary   string
	Type      string // "architecture" | "tech-stack"
	Labels    []string
	RelatedTo []string
	Pros      []string
	Cons      []string
}

// LoadCatalog reads every architecture and tech-stack catalog artefact under
// lifecycle/architecture/{architectures,tech-stacks}/*.md, in deterministic
// (slug-sorted) order.
func LoadCatalog(projectRoot string) (arches, stacks []CatalogItem, err error) {
	arches, err = loadCatalogDir(projectRoot, "architectures", "architecture")
	if err != nil {
		return nil, nil, err
	}
	stacks, err = loadCatalogDir(projectRoot, "tech-stacks", "tech-stack")
	if err != nil {
		return nil, nil, err
	}
	return arches, stacks, nil
}

func loadCatalogDir(projectRoot, subdir, kind string) ([]CatalogItem, error) {
	archDir := filepath.Join(projectRoot, filepath.FromSlash(architectureDir))
	absDir, err := sandbox.Resolve(archDir, subdir)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []CatalogItem
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		absPath := filepath.Join(absDir, e.Name())
		raw, rerr := os.ReadFile(absPath)
		if rerr != nil {
			continue
		}
		relPath := path.Join(architectureDir, subdir, e.Name())
		a := artifact.Parse(raw, relPath, time.Time{})
		if a.FM.Type != kind {
			continue
		}
		pros, cons := parseProsCons(a.Body)
		related := make([]string, len(a.FM.Related))
		for i, r := range a.FM.Related {
			related[i] = normaliseCatalogRef(r)
		}
		items = append(items, CatalogItem{
			Path:      relPath,
			Slug:      strings.TrimSuffix(e.Name(), ".md"),
			Title:     a.FM.Title,
			Summary:   a.FM.Summary,
			Type:      a.FM.Type,
			Labels:    a.FM.Labels,
			RelatedTo: related,
			Pros:      pros,
			Cons:      cons,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Slug < items[j].Slug })
	return items, nil
}

// normaliseCatalogRef resolves a related_to target (relative to lifecycle/,
// per artifact.normaliseLinkTarget's convention) to the repo-relative path
// LoadCatalog stamps on CatalogItem.Path, so the two are directly comparable.
func normaliseCatalogRef(target string) string {
	target = strings.TrimSpace(target)
	if !strings.HasPrefix(target, "lifecycle/") {
		target = "lifecycle/" + target
	}
	if !strings.HasSuffix(target, ".md") {
		target += ".md"
	}
	return target
}

// parseProsCons extracts the bullet items under "## Pros" and "## Cons"
// headings in a catalog artefact's markdown body. Best-effort: absent
// sections yield nil slices, and scoring never depends on the result.
func parseProsCons(body string) (pros, cons []string) {
	var current *[]string
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.EqualFold(trimmed, "## Pros"):
			current = &pros
			continue
		case strings.EqualFold(trimmed, "## Cons"):
			current = &cons
			continue
		case strings.HasPrefix(trimmed, "#"):
			current = nil
			continue
		}
		if current == nil {
			continue
		}
		if item, ok := strings.CutPrefix(trimmed, "- "); ok {
			*current = append(*current, strings.TrimSpace(item))
		}
	}
	return pros, cons
}
