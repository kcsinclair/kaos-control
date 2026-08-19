// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
)

func TestNextADRNumber_EmptyDecisionsDir(t *testing.T) {
	root := t.TempDir()
	n, err := architecture.NextADRNumber(root)
	if err != nil {
		t.Fatalf("NextADRNumber: %v", err)
	}
	if n != 1 {
		t.Errorf("want 1, got %d", n)
	}
}

func TestCreateADR_WritesADR0001AndDefaultsToDraft(t *testing.T) {
	root := t.TempDir()
	relPath, err := architecture.CreateADR(root, architecture.ADRRequest{
		Slug:  "adopt-postgres",
		Title: "Adopt Postgres",
		Body:  "Because reasons.",
	})
	if err != nil {
		t.Fatalf("CreateADR: %v", err)
	}
	if relPath != "lifecycle/architecture/decisions/adr-0001-adopt-postgres.md" {
		t.Errorf("relPath: got %q", relPath)
	}
	content := readFile(t, filepath.Join(root, relPath))
	if !strings.Contains(content, "status: draft") {
		t.Errorf("expected default status: draft, got:\n%s", content)
	}
}

func TestNextADRNumber_WithTwoPresent(t *testing.T) {
	root := t.TempDir()
	if _, err := architecture.CreateADR(root, architecture.ADRRequest{Slug: "one", Title: "One"}); err != nil {
		t.Fatalf("CreateADR one: %v", err)
	}
	if _, err := architecture.CreateADR(root, architecture.ADRRequest{Slug: "two", Title: "Two"}); err != nil {
		t.Fatalf("CreateADR two: %v", err)
	}

	n, err := architecture.NextADRNumber(root)
	if err != nil {
		t.Fatalf("NextADRNumber: %v", err)
	}
	if n != 3 {
		t.Fatalf("want 3, got %d", n)
	}

	// Deleting the highest-numbered file lowers the next allocation — numbering
	// derives from files present, not from a persisted counter.
	if err := os.Remove(filepath.Join(root, "lifecycle/architecture/decisions/adr-0002-two.md")); err != nil {
		t.Fatal(err)
	}
	n2, err := architecture.NextADRNumber(root)
	if err != nil {
		t.Fatalf("NextADRNumber after delete: %v", err)
	}
	if n2 != 2 {
		t.Errorf("want 2 after deleting adr-0002, got %d", n2)
	}
}

func TestNextADRNumber_SupersededStillCounts(t *testing.T) {
	root := t.TempDir()
	for i, slug := range []string{"one", "two", "three-superseded"} {
		status := "draft"
		if i == 2 {
			status = "superseded"
		}
		if _, err := architecture.CreateADR(root, architecture.ADRRequest{Slug: slug, Title: slug, Status: status}); err != nil {
			t.Fatalf("CreateADR %s: %v", slug, err)
		}
	}

	n, err := architecture.NextADRNumber(root)
	if err != nil {
		t.Fatalf("NextADRNumber: %v", err)
	}
	if n != 4 {
		t.Errorf("want 4 (superseded adr-0003 still counts), got %d", n)
	}
}

func TestWriteADR0001_IsIdempotent(t *testing.T) {
	root := t.TempDir()
	path1, err := architecture.WriteADR0001(root, "Postgres Modular Monolith", "Go + Vue", "Q: why? A: because.", []string{"Microservices"})
	if err != nil {
		t.Fatalf("first WriteADR0001: %v", err)
	}
	path2, err := architecture.WriteADR0001(root, "Postgres Modular Monolith", "Go + Vue", "Q: why? A: because.", []string{"Microservices"})
	if err != nil {
		t.Fatalf("second WriteADR0001: %v", err)
	}
	if path1 != path2 {
		t.Errorf("expected the same path both times, got %q then %q", path1, path2)
	}

	entries, err := os.ReadDir(filepath.Join(root, "lifecycle/architecture/decisions"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("expected exactly one adr-0001-*.md, got %v", names)
	}
	if !strings.HasPrefix(entries[0].Name(), "adr-0001-") {
		t.Errorf("expected adr-0001-*, got %q", entries[0].Name())
	}

	content := readFile(t, filepath.Join(root, path1))
	if !strings.Contains(content, "status: approved") {
		t.Errorf("expected status: approved, got:\n%s", content)
	}
	if !strings.Contains(content, "## Rejected alternatives") || !strings.Contains(content, "Microservices") {
		t.Errorf("expected rejected-alternatives section, got:\n%s", content)
	}
	if !strings.Contains(content, "created:") {
		t.Errorf("expected a created: field in the ADR frontmatter, got:\n%s", content)
	}
}

func TestCreateADR_DefaultStatusIsDraftWhenEmpty(t *testing.T) {
	root := t.TempDir()
	relPath, err := architecture.CreateADR(root, architecture.ADRRequest{Slug: "x", Title: "X", Status: ""})
	if err != nil {
		t.Fatalf("CreateADR: %v", err)
	}
	content := readFile(t, filepath.Join(root, relPath))
	if !strings.Contains(content, "status: draft") {
		t.Errorf("expected status: draft, got:\n%s", content)
	}
}
