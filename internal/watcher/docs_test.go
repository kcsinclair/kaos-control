// SPDX-License-Identifier: AGPL-3.0-or-later

package watcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kaos-control/kaos-control/internal/hub"
)

// newDocsWatcher builds a minimal Watcher with the given project root for
// testing the docs-specific methods. index and callbacks are left nil.
func newDocsWatcher(t *testing.T, projectRoot string) *Watcher {
	t.Helper()
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("fsnotify.NewWatcher: %v", err)
	}
	t.Cleanup(func() { fsw.Close() })
	return &Watcher{
		fsw:          fsw,
		lifecycleDir: filepath.Join(projectRoot, "lifecycle"),
		docsDir:      filepath.Join(projectRoot, "docs"),
		projectRoot:  projectRoot,
		hub:          hub.New(),
	}
}

// collectDocChanged waits up to maxWait for doc.changed events on ch.
func collectDocChanged(ch <-chan []byte, maxWait time.Duration) []string {
	var paths []string
	deadline := time.After(maxWait)
	for {
		select {
		case data := <-ch:
			var evt struct {
				Type    string            `json:"type"`
				Payload map[string]string `json:"payload"`
			}
			if json.Unmarshal(data, &evt) == nil && evt.Type == "doc.changed" {
				paths = append(paths, evt.Payload["path"])
			}
		case <-deadline:
			return paths
		}
	}
}

func TestHandleDocChange_Create(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	w := newDocsWatcher(t, root)
	ch := make(chan []byte, 8)
	w.hub.Register(ch)

	// Create a file and call handleDocChange.
	absPath := filepath.Join(docsDir, "architecture.md")
	if err := os.WriteFile(absPath, []byte("# Arch"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.handleDocChange(absPath)

	paths := collectDocChanged(ch, 100*time.Millisecond)
	if len(paths) != 1 || paths[0] != "architecture.md" {
		t.Errorf("doc.changed paths: got %v, want [architecture.md]", paths)
	}
}

func TestHandleDocChange_SubDir(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "docs", "subsystems")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	w := newDocsWatcher(t, root)
	ch := make(chan []byte, 8)
	w.hub.Register(ch)

	absPath := filepath.Join(subDir, "agents.md")
	if err := os.WriteFile(absPath, []byte("# Agents"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.handleDocChange(absPath)

	paths := collectDocChanged(ch, 100*time.Millisecond)
	if len(paths) != 1 || paths[0] != "subsystems/agents.md" {
		t.Errorf("doc.changed paths: got %v, want [subsystems/agents.md]", paths)
	}
}

func TestHandleDocChange_DeletedFile(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	w := newDocsWatcher(t, root)
	ch := make(chan []byte, 8)
	w.hub.Register(ch)

	// File does not exist — simulates deletion event.
	absPath := filepath.Join(docsDir, "deleted.md")
	w.handleDocChange(absPath)

	paths := collectDocChanged(ch, 100*time.Millisecond)
	if len(paths) != 1 || paths[0] != "deleted.md" {
		t.Errorf("doc.changed paths for delete: got %v, want [deleted.md]", paths)
	}
}

func TestIsDocsFile_LifecyclePath_NotDocs(t *testing.T) {
	root := t.TempDir()
	w := newDocsWatcher(t, root)

	lifecyclePath := filepath.Join(root, "lifecycle", "ideas", "login.md")
	if w.isDocsFile(lifecyclePath) {
		t.Error("lifecycle path should not be identified as a docs file")
	}
}

func TestIsDocsFile_DocsPath(t *testing.T) {
	root := t.TempDir()
	w := newDocsWatcher(t, root)

	docsPath := filepath.Join(root, "docs", "architecture.md")
	if !w.isDocsFile(docsPath) {
		t.Error("docs path should be identified as a docs file")
	}
}
