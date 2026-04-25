//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestExternalEditPickedUp verifies that dropping a file into lifecycle/
// directly triggers the fsnotify watcher, which re-indexes the artifact
// within a reasonable time.
// Test plan §7: "External edit" scenario.
func TestExternalEditPickedUp(t *testing.T) {
	env := newTestEnv(t, nil)

	// Verify the file does not exist in the index.
	row, err := env.proj.Idx.Get("lifecycle/ideas/external.md")
	if err != nil {
		t.Fatal(err)
	}
	if row != nil {
		t.Fatal("expected external.md to not be in index before external edit")
	}

	// Write a file directly on disk (simulating an external editor).
	content := makeArtifact("External Edit", "idea", "draft", "external", "", "Written by an external tool.")
	absPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "external.md")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to pick it up (150ms debounce + processing time).
	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		row, err = env.proj.Idx.Get("lifecycle/ideas/external.md")
		if err != nil {
			t.Fatal(err)
		}
		if row != nil {
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !found {
		t.Error("watcher did not pick up externally written file within 2s")
	}

	if row != nil {
		if row.Title != "External Edit" {
			t.Errorf("expected title 'External Edit', got %q", row.Title)
		}
		if row.Status != "draft" {
			t.Errorf("expected status 'draft', got %q", row.Status)
		}
	}
}

// TestExternalEditUpdateExisting verifies that modifying an existing file
// on disk is detected by the watcher and the index is updated.
func TestExternalEditUpdateExisting(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/watched.md",
			content: makeArtifact("Watched Idea", "idea", "draft", "watched", "", "Original content."),
		},
	}
	env := newTestEnv(t, seeds)

	// Verify initial state.
	row, _ := env.proj.Idx.Get("lifecycle/ideas/watched.md")
	if row == nil {
		t.Fatal("seed artifact not indexed")
	}
	if row.Title != "Watched Idea" {
		t.Fatalf("unexpected initial title: %q", row.Title)
	}

	// Overwrite the file on disk with a new title.
	updated := makeArtifact("Watched Idea Updated", "idea", "draft", "watched", "", "Modified content.")
	absPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "watched.md")
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to pick it up.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, _ = env.proj.Idx.Get("lifecycle/ideas/watched.md")
		if row != nil && row.Title == "Watched Idea Updated" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if row == nil || row.Title != "Watched Idea Updated" {
		t.Errorf("expected updated title 'Watched Idea Updated', got %v", row)
	}
}

// TestExternalDeleteRemovesFromIndex verifies that deleting a file on disk
// causes the watcher to remove it from the index.
func TestExternalDeleteRemovesFromIndex(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/ephemeral.md",
			content: makeArtifact("Ephemeral", "idea", "draft", "ephemeral", "", "Will be deleted."),
		},
	}
	env := newTestEnv(t, seeds)

	// Verify it's indexed.
	row, _ := env.proj.Idx.Get("lifecycle/ideas/ephemeral.md")
	if row == nil {
		t.Fatal("seed artifact not indexed")
	}

	// Delete the file on disk.
	absPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "ephemeral.md")
	if err := os.Remove(absPath); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to remove it from the index.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, _ = env.proj.Idx.Get("lifecycle/ideas/ephemeral.md")
		if row == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if row != nil {
		t.Error("expected deleted file to be removed from index")
	}
}
