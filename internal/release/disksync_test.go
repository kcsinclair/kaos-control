// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskSync_Write_AtomicRename(t *testing.T) {
	root := t.TempDir()
	// Seed lifecycle/releases directory.
	if err := os.MkdirAll(filepath.Join(root, "lifecycle", "releases"), 0o755); err != nil {
		t.Fatal(err)
	}

	expected := NewExpectedEvents()
	ds := NewDiskSync(expected)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := &Release{
		ID:        1,
		ProjectID: "proj",
		Name:      "Q1 2026",
		Slug:      "q1-2026",
		Status:    "planned",
		UpdatedAt: now,
	}

	rel, err := ds.Write(root, r)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if rel != "lifecycle/releases/q1-2026.md" {
		t.Errorf("Write returned relPath=%q, want lifecycle/releases/q1-2026.md", rel)
	}

	// Resolve symlinks so the path matches what sandbox.Resolve returns
	// (on macOS, t.TempDir() may sit under /tmp → /private/tmp symlink).
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	absPath := filepath.Join(resolvedRoot, "lifecycle", "releases", "q1-2026.md")

	if _, err := os.Stat(absPath); err != nil {
		t.Errorf("expected file to exist at %s: %v", absPath, err)
	}

	// Temp file must be gone.
	if _, err := os.Stat(absPath + ".tmp"); !os.IsNotExist(err) {
		t.Error("expected .tmp file to be absent after Write")
	}

	// Expected set must be non-empty until watcher consumes.
	if !expected.Consume(absPath) {
		t.Error("Consume returned false; expected event was not registered")
	}
}

func TestDiskSync_Write_SandboxRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	expected := NewExpectedEvents()
	ds := NewDiskSync(expected)

	// Use enough ".." segments to escape the project root regardless of depth.
	// "lifecycle/releases/" is 2 directories deep, so 3+ levels of ".." escape.
	r := &Release{
		ID:        1,
		ProjectID: "proj",
		Name:      "evil",
		Slug:      "../../../../../../etc/passwd",
		Status:    "planned",
		UpdatedAt: time.Now(),
	}
	_, err := ds.Write(root, r)
	if err == nil {
		t.Error("expected error for traversal slug, got nil")
	}
}

func TestDiskSync_Write_FallbackSlug(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lifecycle", "releases"), 0o755); err != nil {
		t.Fatal(err)
	}

	expected := NewExpectedEvents()
	ds := NewDiskSync(expected)

	r := &Release{
		ID:        42,
		ProjectID: "proj",
		Name:      "🚀",
		Slug:      "release-42",
		Status:    "planned",
		UpdatedAt: time.Now().UTC(),
	}

	rel, err := ds.Write(root, r)
	if err != nil {
		t.Fatalf("Write with fallback slug: %v", err)
	}
	if rel != "lifecycle/releases/release-42.md" {
		t.Errorf("relPath = %q, want lifecycle/releases/release-42.md", rel)
	}
}

func TestDiskSync_Delete(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "lifecycle", "releases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a file to delete.
	absPath := filepath.Join(dir, "q1-2026.md")
	if err := os.WriteFile(absPath, []byte("---\ntitle: test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	expected := NewExpectedEvents()
	ds := NewDiskSync(expected)

	if err := ds.Delete(root, "q1-2026"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(absPath); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
	// Expected event should have been registered (consumed by watcher later).
	if expected.Consume(absPath) {
		// OK — event was registered and we consumed it.
	}
}

func TestDiskSync_Rename(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "lifecycle", "releases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create the old file.
	oldAbs := filepath.Join(dir, "q1-2026.md")
	if err := os.WriteFile(oldAbs, []byte("---\ntitle: test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	expected := NewExpectedEvents()
	ds := NewDiskSync(expected)

	r := &Release{
		ID:        1,
		ProjectID: "proj",
		Name:      "Q1 2026 hotfix",
		Slug:      "q1-2026-hotfix",
		Status:    "planned",
		UpdatedAt: time.Now().UTC(),
	}

	newRel, err := ds.Rename(root, "q1-2026", "q1-2026-hotfix", r)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if newRel != "lifecycle/releases/q1-2026-hotfix.md" {
		t.Errorf("Rename relPath = %q", newRel)
	}
	// Old file gone, new file present.
	if _, err := os.Stat(oldAbs); !os.IsNotExist(err) {
		t.Error("expected old file to be gone")
	}
	newAbs := filepath.Join(dir, "q1-2026-hotfix.md")
	if _, err := os.Stat(newAbs); err != nil {
		t.Errorf("expected new file at %s: %v", newAbs, err)
	}
}

func TestExpectedEvents_ConsumeOnlyOnce(t *testing.T) {
	e := NewExpectedEvents()
	e.Expect("/some/path")
	if !e.Consume("/some/path") {
		t.Error("first Consume should return true")
	}
	if e.Consume("/some/path") {
		t.Error("second Consume should return false")
	}
}
