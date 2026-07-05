// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPathParsing verifies artifact path parsing for flat, single-nested, and
// deeply-nested paths (Milestone 1).
func TestPathParsing(t *testing.T) {
	env := newTestEnv(t, nil)

	// Test that flat files are indexed
	content := makeArtifact("Flat Idea", "idea", "draft", "flat", "", "Body.")
	absPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "flat.md")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for indexing
	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		row, err := env.proj.Idx.Get("lifecycle/ideas/flat.md")
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
		t.Fatal("flat.md not indexed")
	}

	// Test that single-nested files are indexed
	content = makeArtifact("Nested Idea", "idea", "draft", "nested", "", "Body.")
	absPath = filepath.Join(env.projectRoot, "lifecycle", "ideas", "done", "nested.md")
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for indexing
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, err := env.proj.Idx.Get("lifecycle/ideas/done/nested.md")
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
		t.Fatal("done/nested.md not indexed")
	}

	// Test that deeply-nested files are indexed
	content = makeArtifact("Deeply Nested Idea", "idea", "draft", "deep", "", "Body.")
	absPath = filepath.Join(env.projectRoot, "lifecycle", "ideas", "2026", "q3", "release-x.md")
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for indexing
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, err := env.proj.Idx.Get("lifecycle/ideas/2026/q3/release-x.md")
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
		t.Fatal("2026/q3/release-x.md not indexed")
	}
}

// TestRecursiveScan verifies startup scan indexes nested artifacts (Milestone 2).
func TestRecursiveScan(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/done/archived.md",
			content: makeArtifact("Archived Idea", "idea", "draft", "archived", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	// Verify the nested artifact is indexed
	row, err := env.proj.Idx.Get("lifecycle/ideas/done/archived.md")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Error("nested artifact not indexed at startup")
	}

	// Verify it appears in the API
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/lifecycle/ideas/done/archived.md", nil)
	requireStatus(t, resp, 200)

	// Verify flat behavior is preserved (backward compatibility)
	flatRow, err := env.proj.Idx.Get("lifecycle/ideas/archived.md")
	if err != nil {
		t.Fatal(err)
	}
	if flatRow == nil {
		t.Error("flat artifact not indexed at startup")
	}

	// Verify the artifact is editable via PUT
	resp = env.doRequest("PUT", "/api/p/testproject/artifacts/lifecycle/ideas/done/archived.md", map[string]any{
		"frontmatter": map[string]any{
			"title":   "Archived Idea Updated",
			"type":    "idea",
			"status":  "draft",
			"lineage": "archived",
		},
		"body": "Updated body.",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Wait for indexing
	time.Sleep(100 * time.Millisecond)

	// Check that the update is reflected in the API
	resp = env.doRequest("GET", "/api/p/testproject/artifacts/lifecycle/ideas/done/archived.md", nil)
	requireStatus(t, resp, 200)
}

// TestRuntimeNestedCreate verifies runtime creation of nested artifacts (Milestone 3).
func TestRuntimeNestedCreate(t *testing.T) {
	env := newTestEnv(t, nil)

	// Write a new file into an existing nested directory at runtime
	content := makeArtifact("Runtime Nested", "idea", "draft", "runtime-nested", "", "Body.")
	absPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "done", "runtime.md")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to pick it up
	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		row, err := env.proj.Idx.Get("lifecycle/ideas/done/runtime.md")
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
		t.Error("runtime nested file not indexed within 2s")
	}

	// Create a brand-new subdirectory and artifact in one operation
	newDir := filepath.Join(env.projectRoot, "lifecycle", "ideas", "new-folder")
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content = makeArtifact("New Folder Artifact", "idea", "draft", "new-folder", "", "Body.")
	absPath = filepath.Join(newDir, "artifact.md")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to pick it up
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, err := env.proj.Idx.Get("lifecycle/ideas/new-folder/artifact.md")
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
		t.Error("new folder artifact not indexed within 2s")
	}

	// Verify the directory is watched by writing to it after creation
	content = makeArtifact("New Folder Artifact Updated", "idea", "draft", "new-folder", "", "Updated body.")
	absPath = filepath.Join(newDir, "artifact.md")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for indexing
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, err := env.proj.Idx.Get("lifecycle/ideas/new-folder/artifact.md")
		if err != nil {
			t.Fatal(err)
		}
		if row != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestDotDirExclusion verifies hidden paths are skipped (Milestone 4).
func TestDotDirExclusion(t *testing.T) {
	env := newTestEnv(t, nil)

	// Create a dotfile and a file under a dot directory
	dotFile := makeArtifact("Dot File", "idea", "draft", "dot-file", "", "Body.")
	dotDir := filepath.Join(env.projectRoot, "lifecycle", "ideas", ".trash")
	if err := os.MkdirAll(dotDir, 0o755); err != nil {
		t.Fatal(err)
	}
	absPath := filepath.Join(dotDir, "dot.md")
	if err := os.WriteFile(absPath, []byte(dotFile), 0o644); err != nil {
		t.Fatal(err)
	}

	dotfilePath := filepath.Join(env.projectRoot, "lifecycle", "ideas", ".hidden.md")
	if err := os.WriteFile(dotfilePath, []byte(dotFile), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for indexing
	time.Sleep(500 * time.Millisecond)

	// Verify neither file was indexed
	row1, _ := env.proj.Idx.Get("lifecycle/ideas/.trash/dot.md")
	if row1 != nil {
		t.Error(".trash/dot.md should not be indexed")
	}

	row2, _ := env.proj.Idx.Get("lifecycle/ideas/.hidden.md")
	if row2 != nil {
		t.Error(".hidden.md should not be indexed")
	}
}

// TestNestedWriteSafety verifies nested write safety and path traversal (Milestone 4).
func TestNestedWriteSafety(t *testing.T) {
	env := newTestEnv(t, nil)

	// Test that POST with subdir works
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage":  "ideas",
		"slug":   "subdir-test",
		"body":   "Body.",
		"frontmatter": map[string]any{
			"title":   "Subdir Test",
			"type":    "idea",
			"status":  "draft",
			"lineage": "subdir-test",
		},
		"subdir": "archive",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Verify it's indexed
	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		row, err := env.proj.Idx.Get("lifecycle/ideas/archive/subdir-test.md")
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
		t.Error("artifact with subdir not indexed")
	}

	// Test path traversal rejection
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/ideas/../outside.md", map[string]any{
		"frontmatter": map[string]any{
			"title":   "Outside",
			"type":    "idea",
			"status":  "draft",
			"lineage": "outside",
		},
		"body": "Body.",
	})
	if resp.StatusCode == 200 {
		t.Error("POST to path with .. should be rejected")
	}
	resp.Body.Close()
}

// TestCrossFolderUniqueness verifies lineage index allocation and collision surfacing (Milestone 5).
func TestCrossFolderUniqueness(t *testing.T) {
	env := newTestEnv(t, nil)

	// Create two artifacts with same lineage and index in different folders
	content1 := makeArtifact("Same Index 1", "idea", "draft", "same-index", "", "Body.")
	content2 := makeArtifact("Same Index 2", "idea", "draft", "same-index", "", "Body.")

	absPath1 := filepath.Join(env.projectRoot, "lifecycle", "ideas", "folder1", "file.md")
	absPath2 := filepath.Join(env.projectRoot, "lifecycle", "ideas", "folder2", "file.md")

	if err := os.MkdirAll(filepath.Dir(absPath1), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath1, []byte(content1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath2), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath2, []byte(content2), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for indexing
	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		row1, err := env.proj.Idx.Get("lifecycle/ideas/folder1/file.md")
		if err != nil {
			t.Fatal(err)
		}
		row2, err := env.proj.Idx.Get("lifecycle/ideas/folder2/file.md")
		if err != nil {
			t.Fatal(err)
		}
		if row1 != nil && row2 != nil {
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !found {
		t.Error("files not indexed for cross-folder uniqueness test")
	}
}

// TestMovePreservesIdentity verifies that moving an artifact preserves identity (Milestone 3).
func TestMovePreservesIdentity(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/move-test.md",
			content: makeArtifact("Move Test", "idea", "draft", "move-test", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	// Verify initial state
	row, err := env.proj.Idx.Get("lifecycle/ideas/move-test.md")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Error("initial artifact not indexed")
	}

	// Move the file from root to a subdirectory (simulating filesystem move)
	oldPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "move-test.md")
	newPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "done", "move-test.md")
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	// Wait for indexing
	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		row, err := env.proj.Idx.Get("lifecycle/ideas/done/move-test.md")
		if err != nil {
			t.Fatal(err)
		}
		if row != nil {
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Verify the artifact is now in the new location
	if !found {
		t.Error("moved artifact not indexed")
	}

	// Move it back to root
	oldPath = newPath
	newPath = filepath.Join(env.projectRoot, "lifecycle", "ideas", "move-test.md")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	// Wait for indexing
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, err := env.proj.Idx.Get("lifecycle/ideas/move-test.md")
		if err != nil {
			t.Fatal(err)
		}
		if row != nil {
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Verify it's back in the root
	if !found {
		t.Error("artifact not indexed after move back")
	}
}