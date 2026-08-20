// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/index"
)

// Role is the catalog-role discriminator (FR-9) an architecture-zone
// artifact is classified into. It is independent of the artifact's `type:`
// — e.g. the summary and standards carry `type: doc` but classify by path.
type Role string

const (
	RoleCatalog            Role = "catalog"
	RoleChosenArchitecture Role = "chosen-architecture"
	RoleChosenStack        Role = "chosen-stack"
	RoleSummary            Role = "summary"
	RoleStandard           Role = "standard"
	RoleADR                Role = "adr"
	RoleArchive            Role = "archive"
)

// OverviewItem is one architecture-zone artifact as surfaced by the overview
// model: light metadata plus its catalog-role. Panel content is never
// inlined — the frontend fetches bodies via GET /artifacts/*path (NFR-1).
type OverviewItem struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Created     string `json:"created,omitempty"`
	CatalogRole Role   `json:"catalog_role"`
}

// Overview is the assembled, read-only model of the whole architecture zone
// (FR-9), degrading to empty/null fields rather than erroring when parts are
// absent (FR-10, NFR-5).
type Overview struct {
	HasChosenArchitecture bool           `json:"has_chosen_architecture"`
	ChosenArchitecture    *OverviewItem  `json:"chosen_architecture"`
	ChosenStack           *OverviewItem  `json:"chosen_stack"`
	Summary               *OverviewItem  `json:"summary"`
	Standards             []OverviewItem `json:"standards"`
	ADRs                  []OverviewItem `json:"adrs"`
	Archive               []OverviewItem `json:"archive"`
	Catalog               []OverviewItem `json:"catalog"`
}

// LoadOverview assembles and classifies every artifact under
// lifecycle/architecture/ into an Overview. Discovery prefers the SQLite
// index (idx) for metadata, falling back to a direct disk read for any file
// present under a known architecture subdirectory but not yet reflected in
// the index — so a very recent on-disk change still appears (FR-12). The
// assembler performs no writes (NFR-2).
func LoadOverview(projectRoot string, idx *index.Index) (Overview, error) {
	byPath, err := indexedArchitectureRows(idx)
	if err != nil {
		return Overview{}, err
	}

	ov := Overview{
		Standards: []OverviewItem{},
		ADRs:      []OverviewItem{},
		Archive:   []OverviewItem{},
		Catalog:   []OverviewItem{},
	}

	for _, subdir := range []string{"architectures", "tech-stacks"} {
		items, err := loadRoleDir(projectRoot, byPath, subdir)
		if err != nil {
			return Overview{}, err
		}
		ov.Catalog = append(ov.Catalog, items...)
	}
	sort.Slice(ov.Catalog, func(i, j int) bool { return ov.Catalog[i].Path < ov.Catalog[j].Path })

	if item, ok := loadPromotedRoot(projectRoot, byPath, "architecture"); ok {
		ov.ChosenArchitecture = &item
		ov.HasChosenArchitecture = true
	}
	if item, ok := loadPromotedRoot(projectRoot, byPath, "tech-stack"); ok {
		ov.ChosenStack = &item
	}

	if item, ok := overviewItemFor(projectRoot, byPath, path.Join(architectureDir, "architecture-summary.md")); ok {
		ov.Summary = &item
	}

	if items, err := loadRoleDir(projectRoot, byPath, "standards"); err != nil {
		return Overview{}, err
	} else {
		ov.Standards = items
	}

	if items, err := loadRoleDir(projectRoot, byPath, "decisions"); err != nil {
		return Overview{}, err
	} else {
		ov.ADRs = items
		sortADRsNewestFirst(ov.ADRs)
	}

	if items, err := loadRoleDir(projectRoot, byPath, "archive"); err != nil {
		return Overview{}, err
	} else {
		ov.Archive = items
	}

	return ov, nil
}

// classifyRole determines an architecture-zone artifact's catalog-role from
// its repo-relative path (and, for the root-level promoted files and ADRs,
// its `type:`) — never by `type:` alone (FR-9), since the summary and
// standards deliberately share `type: doc` (OQ-2).
func classifyRole(relPath string, fm artifact.Frontmatter) Role {
	rel, ok := strings.CutPrefix(relPath, architectureDir+"/")
	if !ok {
		return ""
	}
	switch {
	case rel == "archive" || strings.HasPrefix(rel, "archive/"):
		return RoleArchive
	case rel == "decisions" || strings.HasPrefix(rel, "decisions/"):
		if fm.Type == "adr" {
			return RoleADR
		}
		return ""
	case rel == "standards" || strings.HasPrefix(rel, "standards/"):
		return RoleStandard
	case rel == "architecture-summary.md":
		return RoleSummary
	case strings.HasPrefix(rel, "architectures/") || strings.HasPrefix(rel, "tech-stacks/"):
		return RoleCatalog
	case !strings.Contains(rel, "/"):
		switch fm.Type {
		case "architecture":
			return RoleChosenArchitecture
		case "tech-stack":
			return RoleChosenStack
		}
	}
	return ""
}

// indexedArchitectureRows returns every indexed row under lifecycle/architecture/,
// keyed by repo-relative path, as the primary metadata source for LoadOverview.
func indexedArchitectureRows(idx *index.Index) (map[string]*index.ArtifactRow, error) {
	rows, _, err := idx.List(index.Filter{Unlimited: true})
	if err != nil {
		return nil, err
	}
	prefix := architectureDir + "/"
	out := make(map[string]*index.ArtifactRow, len(rows))
	for _, r := range rows {
		if strings.HasPrefix(r.Path, prefix) {
			out[r.Path] = r
		}
	}
	return out, nil
}

// overviewItemFor builds the OverviewItem for relPath, preferring the
// indexed row and falling back to a direct disk read+parse when the path is
// not (yet) indexed. Returns ok=false when the file cannot be read/parsed or
// classifies to no role.
func overviewItemFor(projectRoot string, byPath map[string]*index.ArtifactRow, relPath string) (OverviewItem, bool) {
	var fm artifact.Frontmatter
	var title, status, typ string

	if row, found := byPath[relPath]; found {
		fm = row.FM
		title, status, typ = row.Title, row.Status, row.Type
	} else {
		absPath := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		raw, rerr := os.ReadFile(absPath)
		if rerr != nil {
			return OverviewItem{}, false
		}
		a := artifact.Parse(raw, relPath, time.Time{})
		fm = a.FM
		title, status, typ = a.FM.Title, a.FM.Status, a.FM.Type
	}

	role := classifyRole(relPath, fm)
	if role == "" {
		return OverviewItem{}, false
	}
	return OverviewItem{
		Path:        relPath,
		Title:       title,
		Status:      status,
		Type:        typ,
		Created:     fm.Created,
		CatalogRole: role,
	}, true
}

// loadRoleDir lists the .md files directly under
// lifecycle/architecture/<subdir> and classifies each via overviewItemFor.
// A missing directory yields an empty (non-nil) slice, not an error.
func loadRoleDir(projectRoot string, byPath map[string]*index.ArtifactRow, subdir string) ([]OverviewItem, error) {
	absDir := filepath.Join(projectRoot, filepath.FromSlash(architectureDir), subdir)
	entries, err := os.ReadDir(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []OverviewItem{}, nil
		}
		return nil, err
	}

	items := []OverviewItem{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		relPath := path.Join(architectureDir, subdir, e.Name())
		if item, ok := overviewItemFor(projectRoot, byPath, relPath); ok {
			items = append(items, item)
		}
	}
	return items, nil
}

// loadPromotedRoot resolves the current root-level promotion of the given
// kind ("architecture" or "tech-stack"), reusing the same disk-based
// promoted-root detection Promote/LoadPromotedStackProfile rely on.
func loadPromotedRoot(projectRoot string, byPath map[string]*index.ArtifactRow, kind string) (OverviewItem, bool) {
	promoted, err := currentlyPromoted(projectRoot, kind)
	if err != nil || len(promoted) == 0 {
		return OverviewItem{}, false
	}
	return overviewItemFor(projectRoot, byPath, promoted[0])
}

// sortADRsNewestFirst sorts items in place by descending ADR number (FR-7),
// tie-broken by descending created date.
func sortADRsNewestFirst(items []OverviewItem) {
	sort.Slice(items, func(i, j int) bool {
		ni, nj := adrNumberOf(items[i].Path), adrNumberOf(items[j].Path)
		if ni != nj {
			return ni > nj
		}
		return items[i].Created > items[j].Created
	})
}

// adrNumberOf parses the zero-padded number out of an ADR filename, or 0 if
// it doesn't match the adr-NNNN-*.md convention.
func adrNumberOf(relPath string) int {
	m := adrFileRe.FindStringSubmatch(filepath.Base(relPath))
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}
