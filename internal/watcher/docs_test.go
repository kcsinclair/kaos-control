// SPDX-License-Identifier: AGPL-3.0-or-later

package watcher

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
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

// --- Start()-based integration tests (real fsnotify, 150 ms debounce) ---

// startWatcher launches Start in a goroutine and returns a cancel func.
// The watcher's fsnotify setup completes within the 100 ms sleep that follows.
func startWatcher(t *testing.T, w *Watcher) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	go w.Start(ctx) //nolint:errcheck
	// Give fsnotify time to add directory watches before we write files.
	time.Sleep(100 * time.Millisecond)
	t.Cleanup(cancel)
	return cancel
}

func TestWatcher_DocCreateEmitsDocChanged(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// lifecycle/ not created; no lifecycle events fire, so nil index is safe.
	if err := os.MkdirAll(filepath.Join(root, "lifecycle"), 0o755); err != nil {
		t.Fatal(err)
	}

	h := hub.New()
	ch := make(chan []byte, 16)
	h.Register(ch)

	w, err := New(root, nil, h)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	startWatcher(t, w)

	absPath := filepath.Join(docsDir, "a.md")
	if err := os.WriteFile(absPath, []byte("# A"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths := collectDocChanged(ch, 500*time.Millisecond)
	found := false
	for _, p := range paths {
		if p == "a.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("doc.changed: expected at least one event for %q, got paths: %v", "a.md", paths)
	}
}

func TestWatcher_DocModifyEmitsDocChanged(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lifecycle"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-create the file before starting the watcher.
	absPath := filepath.Join(docsDir, "a.md")
	if err := os.WriteFile(absPath, []byte("# A original"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := hub.New()
	ch := make(chan []byte, 16)
	h.Register(ch)

	w, err := New(root, nil, h)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	startWatcher(t, w)

	// Modify the file.
	if err := os.WriteFile(absPath, []byte("# A modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths := collectDocChanged(ch, 500*time.Millisecond)
	if len(paths) == 0 {
		t.Error("doc.changed: expected at least one event for modified file; got none")
	}
}

func TestWatcher_DocDeleteEmitsDocChanged(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lifecycle"), 0o755); err != nil {
		t.Fatal(err)
	}

	absPath := filepath.Join(docsDir, "a.md")
	if err := os.WriteFile(absPath, []byte("# A"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := hub.New()
	ch := make(chan []byte, 16)
	h.Register(ch)

	w, err := New(root, nil, h)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	startWatcher(t, w)

	if err := os.Remove(absPath); err != nil {
		t.Fatal(err)
	}

	paths := collectDocChanged(ch, 500*time.Millisecond)
	if len(paths) == 0 {
		t.Error("doc.changed: expected at least one event for deleted file; got none")
	}
	// No index mutation happens for docs files (verified by the nil index not panicking).
}

func TestWatcher_LifecycleStillEmitsFileChanged(t *testing.T) {
	root := t.TempDir()
	ideasDir := filepath.Join(root, "lifecycle", "ideas")
	if err := os.MkdirAll(ideasDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Lifecycle events invoke the indexer, so a real index is required.
	idx, err := index.Open(
		filepath.Join(root, "idx.db"),
		root,
		[]config.Stage{{Name: "ideas", Dir: "ideas"}},
	)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	h := hub.New()
	ch := make(chan []byte, 32)
	h.Register(ch)

	w, err := New(root, idx, h)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	startWatcher(t, w)

	// Write a lifecycle artifact.
	lifecycleFile := filepath.Join(ideasDir, "x.md")
	content := "---\ntitle: Watcher Test\ntype: idea\nstatus: draft\nlineage: watcher-test\n---\n\nBody.\n"
	if err := os.WriteFile(lifecycleFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotFileChanged, gotDocChanged bool
	deadline := time.After(500 * time.Millisecond)
COLLECT:
	for {
		select {
		case data := <-ch:
			var evt struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(data, &evt) == nil {
				switch evt.Type {
				case "file.changed":
					gotFileChanged = true
				case "doc.changed":
					gotDocChanged = true
				}
			}
		case <-deadline:
			break COLLECT
		}
	}

	if !gotFileChanged {
		t.Error("expected file.changed event for lifecycle file; got none within 500 ms")
	}
	if gotDocChanged {
		t.Error("unexpected doc.changed event emitted for a lifecycle file")
	}
}

func TestWatcher_MissingDocsDirIsNonFatal(t *testing.T) {
	root := t.TempDir()
	// Create lifecycle/ but NOT docs/.
	if err := os.MkdirAll(filepath.Join(root, "lifecycle"), 0o755); err != nil {
		t.Fatal(err)
	}

	w, err := New(root, nil, hub.New())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	startErr := make(chan error, 1)
	go func() {
		startErr <- w.Start(ctx)
	}()

	// The watcher must still be running after a short wait — no early exit.
	select {
	case err := <-startErr:
		t.Fatalf("watcher exited prematurely (missing docs/ should be non-fatal): %v", err)
	case <-time.After(200 * time.Millisecond):
		// Good — watcher running normally despite missing docs/.
	}
}
