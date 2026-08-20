// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/index"
)

func writeOverviewFixture(t *testing.T, relPath, content string) {
	t.Helper()
	// caller supplies an absolute path already joined to a temp root.
	if err := os.MkdirAll(filepath.Dir(relPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(relPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestClassifyRole(t *testing.T) {
	cases := []struct {
		name    string
		relPath string
		fm      artifact.Frontmatter
		want    Role
	}{
		{"catalog architecture", "lifecycle/architecture/architectures/modular-monolith.md", artifact.Frontmatter{Type: "architecture"}, RoleCatalog},
		{"catalog tech-stack", "lifecycle/architecture/tech-stacks/go-vue.md", artifact.Frontmatter{Type: "tech-stack"}, RoleCatalog},
		{"promoted root architecture", "lifecycle/architecture/modular-monolith.md", artifact.Frontmatter{Type: "architecture"}, RoleChosenArchitecture},
		{"promoted root tech-stack", "lifecycle/architecture/go-vue.md", artifact.Frontmatter{Type: "tech-stack"}, RoleChosenStack},
		{"summary by path despite type doc", "lifecycle/architecture/architecture-summary.md", artifact.Frontmatter{Type: "doc"}, RoleSummary},
		{"standard by path despite type doc", "lifecycle/architecture/standards/secrets.md", artifact.Frontmatter{Type: "doc"}, RoleStandard},
		{"adr", "lifecycle/architecture/decisions/adr-0001-adopt-x.md", artifact.Frontmatter{Type: "adr"}, RoleADR},
		{"non-adr in decisions ignored", "lifecycle/architecture/decisions/notes.md", artifact.Frontmatter{Type: "doc"}, ""},
		{"archived file", "lifecycle/architecture/archive/old-modular-monolith.md", artifact.Frontmatter{Type: "architecture"}, RoleArchive},
		{"outside architecture zone", "lifecycle/requirements/foo.md", artifact.Frontmatter{Type: "idea"}, ""},
		{"unrecognised root type", "lifecycle/architecture/README.md", artifact.Frontmatter{Type: "doc"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := classifyRole(c.relPath, c.fm)
			if got != c.want {
				t.Errorf("classifyRole(%q): got %q, want %q", c.relPath, got, c.want)
			}
		})
	}
}

// newOverviewTestIndex opens a real temp-dir SQLite index over root, which
// runs the startup Scan — any fixture files already written under root are
// indexed before this returns.
func newOverviewTestIndex(t *testing.T, root string) *index.Index {
	t.Helper()
	idx, err := index.Open(filepath.Join(root, "test.db"), root, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx
}

func TestLoadOverview_FullModel(t *testing.T) {
	root := t.TempDir()
	arch := filepath.Join(root, "lifecycle/architecture")

	writeOverviewFixture(t, filepath.Join(arch, "architectures/modular-monolith.md"),
		"---\ntitle: Modular Monolith\ntype: architecture\nstatus: approved\nlabels:\n  - catalog\n---\n\nBody.\n")
	writeOverviewFixture(t, filepath.Join(arch, "tech-stacks/go-vue.md"),
		"---\ntitle: Go + Vue\ntype: tech-stack\nstatus: approved\nlabels:\n  - catalog\n---\n\nBody.\n")
	writeOverviewFixture(t, filepath.Join(arch, "modular-monolith.md"),
		"---\ntitle: Modular Monolith\ntype: architecture\nstatus: approved\nparent: lifecycle/architecture/architectures/modular-monolith.md\n---\n\nBody.\n")
	writeOverviewFixture(t, filepath.Join(arch, "go-vue.md"),
		"---\ntitle: Go + Vue\ntype: tech-stack\nstatus: approved\nparent: lifecycle/architecture/tech-stacks/go-vue.md\n---\n\nBody.\n")
	writeOverviewFixture(t, filepath.Join(arch, "architecture-summary.md"),
		"---\ntitle: Architecture Summary\ntype: doc\nstatus: approved\n---\n\nSummary.\n")
	writeOverviewFixture(t, filepath.Join(arch, "standards/secrets.md"),
		"---\ntitle: Secrets Handling\ntype: doc\nstatus: approved\n---\n\nStandard.\n")
	writeOverviewFixture(t, filepath.Join(arch, "standards/usability.md"),
		"---\ntitle: Usability\ntype: doc\nstatus: approved\n---\n\nStandard.\n")
	writeOverviewFixture(t, filepath.Join(arch, "decisions/adr-0001-adopt-modular-monolith.md"),
		"---\ntitle: Adopt Modular Monolith\ntype: adr\nstatus: approved\ncreated: \"2026-01-01T00:00:00Z\"\n---\n\nADR.\n")
	writeOverviewFixture(t, filepath.Join(arch, "decisions/adr-0002-no-header-ip-trust.md"),
		"---\ntitle: No Header-Based Client IP Trust\ntype: adr\nstatus: approved\ncreated: \"2026-01-02T00:00:00Z\"\n---\n\nADR.\n")
	writeOverviewFixture(t, filepath.Join(arch, "archive/old-modular-monolith.md"),
		"---\ntitle: Old Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nArchived.\n")

	idx := newOverviewTestIndex(t, root)

	ov, err := LoadOverview(root, idx)
	if err != nil {
		t.Fatalf("LoadOverview: %v", err)
	}

	if !ov.HasChosenArchitecture {
		t.Error("HasChosenArchitecture: want true")
	}
	if ov.ChosenArchitecture == nil || ov.ChosenArchitecture.CatalogRole != RoleChosenArchitecture {
		t.Fatalf("ChosenArchitecture: got %+v", ov.ChosenArchitecture)
	}
	if ov.ChosenStack == nil || ov.ChosenStack.CatalogRole != RoleChosenStack {
		t.Fatalf("ChosenStack: got %+v", ov.ChosenStack)
	}
	if ov.Summary == nil || ov.Summary.CatalogRole != RoleSummary {
		t.Fatalf("Summary: got %+v", ov.Summary)
	}
	if len(ov.Standards) != 2 {
		t.Errorf("Standards: got %d, want 2", len(ov.Standards))
	}
	for _, s := range ov.Standards {
		if s.CatalogRole != RoleStandard {
			t.Errorf("standard %q: catalog_role = %q", s.Path, s.CatalogRole)
		}
	}
	if len(ov.ADRs) != 2 {
		t.Fatalf("ADRs: got %d, want 2", len(ov.ADRs))
	}
	if ov.ADRs[0].Path != "lifecycle/architecture/decisions/adr-0002-no-header-ip-trust.md" {
		t.Errorf("ADRs not newest-first: got %q first", ov.ADRs[0].Path)
	}
	if len(ov.Archive) != 1 || ov.Archive[0].CatalogRole != RoleArchive {
		t.Fatalf("Archive: got %+v", ov.Archive)
	}
	if len(ov.Catalog) != 2 {
		t.Errorf("Catalog: got %d, want 2", len(ov.Catalog))
	}
	for _, c := range ov.Catalog {
		if c.CatalogRole != RoleCatalog {
			t.Errorf("catalog item %q: catalog_role = %q", c.Path, c.CatalogRole)
		}
	}
}

func TestLoadOverview_EmptyStandards(t *testing.T) {
	root := t.TempDir()
	arch := filepath.Join(root, "lifecycle/architecture")
	writeOverviewFixture(t, filepath.Join(arch, "modular-monolith.md"),
		"---\ntitle: Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nBody.\n")
	writeOverviewFixture(t, filepath.Join(arch, "go-vue.md"),
		"---\ntitle: Go + Vue\ntype: tech-stack\nstatus: approved\n---\n\nBody.\n")
	if err := os.MkdirAll(filepath.Join(arch, "standards"), 0o755); err != nil {
		t.Fatal(err)
	}

	idx := newOverviewTestIndex(t, root)
	ov, err := LoadOverview(root, idx)
	if err != nil {
		t.Fatalf("LoadOverview: %v", err)
	}
	if ov.Standards == nil || len(ov.Standards) != 0 {
		t.Errorf("Standards: want empty non-nil slice, got %#v", ov.Standards)
	}
	if ov.ADRs == nil || len(ov.ADRs) != 0 {
		t.Errorf("ADRs: want empty non-nil slice, got %#v", ov.ADRs)
	}
}

func TestLoadOverview_NoADRs(t *testing.T) {
	root := t.TempDir()
	arch := filepath.Join(root, "lifecycle/architecture")
	writeOverviewFixture(t, filepath.Join(arch, "modular-monolith.md"),
		"---\ntitle: Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nBody.\n")

	idx := newOverviewTestIndex(t, root)
	ov, err := LoadOverview(root, idx)
	if err != nil {
		t.Fatalf("LoadOverview: %v", err)
	}
	if len(ov.ADRs) != 0 {
		t.Errorf("ADRs: want none, got %d", len(ov.ADRs))
	}
}

func TestLoadOverview_NoChosenArchitecture(t *testing.T) {
	root := t.TempDir()
	arch := filepath.Join(root, "lifecycle/architecture")
	writeOverviewFixture(t, filepath.Join(arch, "architectures/modular-monolith.md"),
		"---\ntitle: Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nBody.\n")

	idx := newOverviewTestIndex(t, root)
	ov, err := LoadOverview(root, idx)
	if err != nil {
		t.Fatalf("LoadOverview: %v", err)
	}
	if ov.HasChosenArchitecture {
		t.Error("HasChosenArchitecture: want false")
	}
	if ov.ChosenArchitecture != nil {
		t.Errorf("ChosenArchitecture: want nil, got %+v", ov.ChosenArchitecture)
	}
	if ov.ChosenStack != nil {
		t.Errorf("ChosenStack: want nil, got %+v", ov.ChosenStack)
	}
	if ov.Summary != nil {
		t.Errorf("Summary: want nil, got %+v", ov.Summary)
	}
	if ov.Standards == nil || len(ov.Standards) != 0 {
		t.Errorf("Standards: want empty non-nil slice, got %#v", ov.Standards)
	}
	if ov.ADRs == nil || len(ov.ADRs) != 0 {
		t.Errorf("ADRs: want empty non-nil slice, got %#v", ov.ADRs)
	}
}

func TestLoadOverview_ArchivePresent(t *testing.T) {
	root := t.TempDir()
	arch := filepath.Join(root, "lifecycle/architecture")
	writeOverviewFixture(t, filepath.Join(arch, "archive/old-modular-monolith.md"),
		"---\ntitle: Old Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nArchived.\n")

	idx := newOverviewTestIndex(t, root)
	ov, err := LoadOverview(root, idx)
	if err != nil {
		t.Fatalf("LoadOverview: %v", err)
	}
	if len(ov.Archive) != 1 {
		t.Fatalf("Archive: got %d, want 1", len(ov.Archive))
	}
	if ov.Archive[0].CatalogRole != RoleArchive {
		t.Errorf("Archive[0].CatalogRole: got %q", ov.Archive[0].CatalogRole)
	}
	if ov.HasChosenArchitecture {
		t.Error("HasChosenArchitecture: want false — archived copies are not chosen")
	}
}

// TestLoadOverview_UnindexedFileStillAppears pins the disk-fallback path
// (FR-12): a file written directly to disk without going through IndexFile
// still appears in the model on the very next LoadOverview call.
func TestLoadOverview_UnindexedFileStillAppears(t *testing.T) {
	root := t.TempDir()
	arch := filepath.Join(root, "lifecycle/architecture")
	writeOverviewFixture(t, filepath.Join(arch, "modular-monolith.md"),
		"---\ntitle: Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nBody.\n")

	idx := newOverviewTestIndex(t, root)

	// Written after Open/Scan, so it is not (yet) in the SQLite index.
	writeOverviewFixture(t, filepath.Join(arch, "standards/secrets.md"),
		"---\ntitle: Secrets Handling\ntype: doc\nstatus: approved\n---\n\nStandard.\n")

	ov, err := LoadOverview(root, idx)
	if err != nil {
		t.Fatalf("LoadOverview: %v", err)
	}
	if len(ov.Standards) != 1 || ov.Standards[0].Title != "Secrets Handling" {
		t.Fatalf("Standards: want the unindexed standard to appear, got %#v", ov.Standards)
	}
}
