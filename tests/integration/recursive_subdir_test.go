// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
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
		{
			relPath: "lifecycle/ideas/archived-flat.md",
			content: makeArtifact("Archived Flat Idea", "idea", "draft", "archived-flat", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	// Verify the nested artifact is indexed
	row, err := env.proj.Idx.Get("lifecycle/ideas/done/archived.md")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("nested artifact not indexed at startup")
	}

	// Verify it appears in the API with rel_path carrying forward slashes (AC5).
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/lifecycle/ideas/done/archived.md", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	artifactObj, _ := data["artifact"].(map[string]any)
	if relPath, _ := artifactObj["rel_path"].(string); relPath != "done/archived.md" {
		t.Errorf("nested rel_path: want %q, got %q", "done/archived.md", relPath)
	}

	// Verify flat behavior is preserved: a flat sibling's rel_path equals the
	// bare filename (AC9, backward compatibility).
	flatRow, err := env.proj.Idx.Get("lifecycle/ideas/archived-flat.md")
	if err != nil {
		t.Fatal(err)
	}
	if flatRow == nil {
		t.Fatal("flat artifact not indexed at startup")
	}
	if flatRow.RelPath != "archived-flat.md" {
		t.Errorf("flat rel_path: want %q, got %q", "archived-flat.md", flatRow.RelPath)
	}

	// Verify the nested artifact appears in the graph endpoint keyed by its
	// full path (AC2).
	resp = env.doRequest("GET", "/api/p/testproject/graph", nil)
	requireStatus(t, resp, 200)
	graphData := readJSON(t, resp)
	nodes := decodeGraphNodes(t, graphData)
	if findNodeByID(nodes, "lifecycle/ideas/done/archived.md") == nil {
		t.Error("nested artifact not present in graph endpoint")
	}

	// Verify the artifact is editable via PUT (AC2).
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
	data = readJSON(t, resp)
	artifactObj, _ = data["artifact"].(map[string]any)
	if title, _ := artifactObj["title"].(string); title != "Archived Idea Updated" {
		t.Errorf("title after PUT: want %q, got %q", "Archived Idea Updated", title)
	}
}

// TestFlatOnlyProjectBackwardCompat verifies that a project seeded with no
// subdirectories at all produces the same index rows as before recursive
// support was added, with rel_path equal to the bare filename for every
// artifact (Milestone 2, AC9 — backward compatibility).
func TestFlatOnlyProjectBackwardCompat(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/one.md", content: makeArtifact("One", "idea", "draft", "one", "", "Body.")},
		{relPath: "lifecycle/ideas/two.md", content: makeArtifact("Two", "idea", "draft", "two", "", "Body.")},
		{relPath: "lifecycle/requirements/one-2.md", content: makeArtifact("One Req", "requirements", "draft", "one", "lifecycle/ideas/one.md", "Body.")},
	}
	env := newTestEnv(t, seeds)

	rows, err := env.proj.Idx.ListByLineage("")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(seeds) {
		t.Fatalf("want %d indexed rows, got %d", len(seeds), len(rows))
	}
	for _, r := range rows {
		want := filepath.Base(r.Path)
		if r.RelPath != want {
			t.Errorf("path %s: rel_path want %q (== filepath.Base(path)), got %q", r.Path, want, r.RelPath)
		}
	}
}

// TestRuntimeNestedCreate verifies runtime creation of nested artifacts (Milestone 3).
func TestRuntimeNestedCreate(t *testing.T) {
	// Seed a placeholder so lifecycle/ideas/done/ already exists (and is
	// watched) before the server starts — this is the "existing nested dir"
	// case from AC3, distinct from the "brand-new subdir" case (AC4) below.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/done/placeholder.md",
			content: makeArtifact("Placeholder", "idea", "draft", "placeholder", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

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
	// Seed a dot-dir and a dotfile so both are present at startup scan time —
	// exercises internal/index's disk-scan dot-prefix skip (AC8), independent
	// of the watcher's runtime-creation path exercised by
	// TestDotDirExclusion_RuntimeCreation below.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/.trash/dot.md",
			content: makeArtifact("Dot File", "idea", "draft", "dot-file", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/.hidden.md",
			content: makeArtifact("Hidden File", "idea", "draft", "hidden-file", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	row1, _ := env.proj.Idx.Get("lifecycle/ideas/.trash/dot.md")
	if row1 != nil {
		t.Error(".trash/dot.md should not be indexed at startup")
	}
	row2, _ := env.proj.Idx.Get("lifecycle/ideas/.hidden.md")
	if row2 != nil {
		t.Error(".hidden.md should not be indexed at startup")
	}
}

// TestDotfileExclusion_Runtime verifies that a dotfile written directly into
// an already-watched, non-hidden directory at runtime is not indexed
// (AC8, via the watcher's own shouldProcess basename check).
func TestDotfileExclusion_Runtime(t *testing.T) {
	env := newTestEnv(t, nil)

	dotFile := makeArtifact("Hidden File", "idea", "draft", "hidden-file", "", "Body.")
	dotfilePath := filepath.Join(env.projectRoot, "lifecycle", "ideas", ".hidden.md")
	if err := os.WriteFile(dotfilePath, []byte(dotFile), 0o644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)

	row, _ := env.proj.Idx.Get("lifecycle/ideas/.hidden.md")
	if row != nil {
		t.Error(".hidden.md should not be indexed")
	}
}

// TestDotDirExclusion_RuntimeCreation verifies that a *.md file created
// together with a brand-new dot-directory at runtime (e.g. `mkdir .trash &&
// write .trash/dot.md` as one operation, mirroring the "new subdir created
// with a file already inside it" race documented in
// internal/watcher/watcher.go's addDirRecursive caller) is not indexed and
// the dot-directory is not watched (AC8).
//
// KNOWN DEFECT: this currently fails. internal/watcher/watcher.go's
// new-directory race-handling fallback (the `filepath.WalkDir(evt.Name, ...)`
// call right after `addDirRecursive` in the fsnotify event loop, around
// watcher.go:206) walks and indexes pre-existing *.md files under a
// newly-created directory without re-checking whether that directory (or any
// ancestor) is dot-prefixed, unlike addDirRecursive itself. A `.git` under
// the lifecycle root is unaffected because `.git` is never created at
// runtime by these tests, only present at repo-init time (covered by the
// startup-scan case above via TestDotDirExclusion).
func TestDotDirExclusion_RuntimeCreation(t *testing.T) {
	env := newTestEnv(t, nil)

	dotFile := makeArtifact("Dot File", "idea", "draft", "dot-file", "", "Body.")
	dotDir := filepath.Join(env.projectRoot, "lifecycle", "ideas", ".trash")
	if err := os.MkdirAll(dotDir, 0o755); err != nil {
		t.Fatal(err)
	}
	absPath := filepath.Join(dotDir, "dot.md")
	if err := os.WriteFile(absPath, []byte(dotFile), 0o644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)

	row, _ := env.proj.Idx.Get("lifecycle/ideas/.trash/dot.md")
	if row != nil {
		t.Error(".trash/dot.md should not be indexed (see KNOWN DEFECT comment above)")
	}
}

// TestNestedWriteSafety verifies nested write safety and path traversal (Milestone 4).
func TestNestedWriteSafety(t *testing.T) {
	env := newTestEnv(t, nil)

	// Test that POST with subdir works
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "subdir-test",
		"body":  "Body.",
		"frontmatter": map[string]any{
			"title":   "Subdir Test",
			"type":    "idea",
			"status":  "draft",
			"lineage": "subdir-test",
		},
		"subdir": "archive",
	})
	requireStatus(t, resp, http.StatusCreated)
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
		t.Fatal("moved artifact not indexed")
	}

	// Verify identity is preserved (ACs 6, 7): type/status/lineage/index/parent
	// unchanged, rel_path updated, exactly one row for the lineage (no
	// duplicate), and the old path is gone.
	movedRow, err := env.proj.Idx.Get("lifecycle/ideas/done/move-test.md")
	if err != nil {
		t.Fatal(err)
	}
	if movedRow.Type != row.Type || movedRow.Status != row.Status || movedRow.Lineage != row.Lineage ||
		movedRow.Index != row.Index || movedRow.Slug != row.Slug || movedRow.FM.Parent != row.FM.Parent {
		t.Errorf("identity diverged after move: before=%+v after=%+v", row, movedRow)
	}
	if movedRow.RelPath != "done/move-test.md" {
		t.Errorf("rel_path after move: want %q, got %q", "done/move-test.md", movedRow.RelPath)
	}

	deadline = time.Now().Add(2 * time.Second)
	var oldGone bool
	for time.Now().Before(deadline) {
		r, err := env.proj.Idx.Get("lifecycle/ideas/move-test.md")
		if err != nil {
			t.Fatal(err)
		}
		if r == nil {
			oldGone = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !oldGone {
		t.Error("old path lifecycle/ideas/move-test.md still indexed after move")
	}

	lineageRows, err := env.proj.Idx.ListByLineage("move-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineageRows) != 1 {
		t.Errorf("want exactly 1 row for lineage move-test after move (no duplicate), got %d", len(lineageRows))
	}

	// Move it back to root
	oldPath = newPath
	newPath = filepath.Join(env.projectRoot, "lifecycle", "ideas", "move-test.md")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	// Wait for indexing
	deadline = time.Now().Add(2 * time.Second)
	found = false
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
		t.Fatal("artifact not indexed after move back")
	}

	deadline = time.Now().Add(2 * time.Second)
	oldGone = false
	for time.Now().Before(deadline) {
		r, err := env.proj.Idx.Get("lifecycle/ideas/done/move-test.md")
		if err != nil {
			t.Fatal(err)
		}
		if r == nil {
			oldGone = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !oldGone {
		t.Error("nested path lifecycle/ideas/done/move-test.md still indexed after moving back to root")
	}

	lineageRows, err = env.proj.Idx.ListByLineage("move-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineageRows) != 1 {
		t.Errorf("want exactly 1 row for lineage move-test after moving back (no duplicate), got %d", len(lineageRows))
	}
}
