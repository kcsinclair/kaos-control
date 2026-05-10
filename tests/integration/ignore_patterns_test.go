// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIgnorePatterns_StartupScan verifies that a file matching the default ignore
// pattern (README.md) is absent from the SQLite index after the startup scan,
// while a legitimate artifact placed alongside it is indexed normally.
//
// Covers test plan Milestone 3.
func TestIgnorePatterns_StartupScan(t *testing.T) {
	env := newTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/README.md",
			content: "# Ideas stage\n\nThis file is intentionally excluded from the artifact index.\n",
		},
		{
			relPath: "lifecycle/ideas/login.md",
			content: makeArtifact("Login Feature", "idea", "draft", "login", "", "User login via email/password."),
		},
	})

	// README.md must not appear in the index.
	row, err := env.proj.Idx.Get("lifecycle/ideas/README.md")
	if err != nil {
		t.Fatalf("Get README.md: %v", err)
	}
	if row != nil {
		t.Errorf("README.md should not be indexed after startup scan, got row: %+v", row)
	}

	// The legitimate artifact must be indexed.
	loginRow, err := env.proj.Idx.Get("lifecycle/ideas/login.md")
	if err != nil {
		t.Fatalf("Get login.md: %v", err)
	}
	if loginRow == nil {
		t.Error("lifecycle/ideas/login.md should be indexed but was not found")
	}
}

// TestIgnorePatterns_WatcherSkipsIgnored verifies live-watcher behaviour:
//   - Writing a README.md to a watched directory does not produce an index row or
//     a file.changed hub event.
//   - Writing a legitimate artifact to the same directory does produce both an
//     index row and a file.changed event.
//
// Covers test plan Milestone 4.
func TestIgnorePatterns_WatcherSkipsIgnored(t *testing.T) {
	env := newTestEnv(t, nil)

	// Register a hub channel before any writes so no events are missed.
	hubCh := make(chan []byte, 64)
	env.proj.Hub.Register((chan<- []byte)(hubCh))
	defer env.proj.Hub.Unregister((chan<- []byte)(hubCh))

	ideasDir := filepath.Join(env.projectRoot, "lifecycle", "ideas")

	// ── Write README.md ──────────────────────────────────────────────────────
	// shouldProcess returns false for ignored files so handleChange is never
	// called; no index row or file.changed event should appear.
	readmePath := filepath.Join(ideasDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Ideas\n"), 0o644); err != nil {
		t.Fatalf("writing README.md: %v", err)
	}

	// Wait beyond the 150 ms debounce window.
	time.Sleep(400 * time.Millisecond)

	row, err := env.proj.Idx.Get("lifecycle/ideas/README.md")
	if err != nil {
		t.Fatalf("index Get README.md: %v", err)
	}
	if row != nil {
		t.Errorf("README.md should not be indexed after watcher event, got row: %+v", row)
	}

	// Drain whatever is in the channel and confirm no file.changed for README.md.
	for {
		select {
		case raw := <-hubCh:
			var evt struct {
				Type    string            `json:"type"`
				Payload map[string]string `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "file.changed" && evt.Payload["path"] == "lifecycle/ideas/README.md" {
				t.Errorf("received unexpected file.changed event for README.md")
			}
		default:
			goto doneReadme
		}
	}
doneReadme:

	// ── Write a legitimate artifact ───────────────────────────────────────────
	// The watcher should index it and broadcast file.changed.
	newPath := filepath.Join(ideasDir, "new-feature.md")
	content := makeArtifact("New Feature", "idea", "draft", "new-feature", "", "A brand new feature idea.")
	if err := os.WriteFile(newPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing new-feature.md: %v", err)
	}

	// Wait for the file.changed event (up to 2 s to absorb any scheduler jitter).
	timeout := time.After(2 * time.Second)
	var gotEvent bool
waitLoop:
	for !gotEvent {
		select {
		case raw := <-hubCh:
			var evt struct {
				Type    string            `json:"type"`
				Payload map[string]string `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "file.changed" && evt.Payload["path"] == "lifecycle/ideas/new-feature.md" {
				gotEvent = true
			}
		case <-timeout:
			break waitLoop
		}
	}
	if !gotEvent {
		t.Error("expected file.changed event for new-feature.md but timed out")
	}

	// Index must now contain the artifact.
	featureRow, err := env.proj.Idx.Get("lifecycle/ideas/new-feature.md")
	if err != nil {
		t.Fatalf("index Get new-feature.md: %v", err)
	}
	if featureRow == nil {
		t.Error("lifecycle/ideas/new-feature.md should be indexed after watcher event but was not found")
	}
}

// TestIgnorePatterns_IndexFileRejectsIgnored verifies that calling IndexFile
// directly with a path matching an ignore pattern returns a non-nil error and
// does not insert any row into the artifacts table.
//
// Covers test plan Milestone 5.
func TestIgnorePatterns_IndexFileRejectsIgnored(t *testing.T) {
	env := newTestEnv(t, nil)

	// Create the file on disk so IndexFile can stat it.
	ideasDir := filepath.Join(env.projectRoot, "lifecycle", "ideas")
	readmePath := filepath.Join(ideasDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Ideas stage\n"), 0o644); err != nil {
		t.Fatalf("writing README.md: %v", err)
	}

	// IndexFile must return a non-nil error for an ignored file.
	if err := env.proj.Idx.IndexFile(readmePath); err == nil {
		t.Error("IndexFile should return an error for an ignored file, got nil")
	}

	// No artifact row must exist in the index for this path.
	row, err := env.proj.Idx.Get("lifecycle/ideas/README.md")
	if err != nil {
		t.Fatalf("Get README.md: %v", err)
	}
	if row != nil {
		t.Errorf("IndexFile must not insert a row for an ignored file, got: %+v", row)
	}
}

// TestIgnorePatterns_APIExcludesIgnored verifies that both the artifact list
// endpoint and the graph endpoint omit files matching the ignore pattern.
//
// Covers test plan Milestone 6.
func TestIgnorePatterns_APIExcludesIgnored(t *testing.T) {
	env := newTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/README.md",
			content: "# Ideas stage overview\n\nNot an artifact.\n",
		},
		{
			relPath: "lifecycle/ideas/login.md",
			content: makeArtifact("Login Feature", "idea", "draft", "login", "", "User login via email/password."),
		},
	})

	// ── GET /api/p/testproject/artifacts ──────────────────────────────────────
	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	var foundLogin bool
	for _, item := range items {
		a, _ := item.(map[string]any)
		path, _ := a["path"].(string)
		if path == "lifecycle/ideas/README.md" {
			t.Errorf("/artifacts response contains ignored file README.md")
		}
		if path == "lifecycle/ideas/login.md" {
			foundLogin = true
		}
	}
	if !foundLogin {
		t.Error("/artifacts response is missing the legitimate artifact login.md")
	}

	// ── GET /api/p/testproject/graph ──────────────────────────────────────────
	resp2 := env.doRequest("GET", "/api/p/testproject/graph", nil)
	graph := readJSON(t, resp2)

	nodes, _ := graph["nodes"].([]any)
	var foundLoginNode bool
	for _, n := range nodes {
		node, _ := n.(map[string]any)
		id, _ := node["id"].(string)
		if id == "lifecycle/ideas/README.md" {
			t.Errorf("/graph response contains ignored file README.md as a node")
		}
		if id == "lifecycle/ideas/login.md" {
			foundLoginNode = true
		}
	}
	if !foundLoginNode {
		t.Error("/graph response is missing login.md node")
	}
}
