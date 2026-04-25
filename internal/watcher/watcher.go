// Package watcher wraps fsnotify to drive incremental artifact re-indexing.
package watcher

import (
	"context"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
)

// Watcher watches lifecycle/ and drives incremental re-indexing + WebSocket events.
type Watcher struct {
	fsw          *fsnotify.Watcher
	lifecycleDir string
	projectRoot  string
	idx          *index.Index
	hub          *hub.Hub
}

// New creates a Watcher but does not start it.
func New(projectRoot string, idx *index.Index, h *hub.Hub) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		fsw:          fsw,
		lifecycleDir: filepath.Join(projectRoot, "lifecycle"),
		projectRoot:  projectRoot,
		idx:          idx,
		hub:          h,
	}, nil
}

// Start begins watching the lifecycle/ tree and blocks until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	if err := w.addDirRecursive(w.lifecycleDir); err != nil {
		return err
	}

	// Debounce: map from path to active timer.
	timers := map[string]*time.Timer{}
	var mu sync.Mutex

	fire := func(path string) {
		mu.Lock()
		defer mu.Unlock()
		if t, ok := timers[path]; ok {
			t.Reset(150 * time.Millisecond)
			return
		}
		timers[path] = time.AfterFunc(150*time.Millisecond, func() {
			mu.Lock()
			delete(timers, path)
			mu.Unlock()
			w.handleChange(path)
		})
	}

	defer w.fsw.Close()
	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}
			if w.shouldProcess(evt.Name) {
				fire(evt.Name)
			}
			// When a new directory is created inside lifecycle/, watch it too.
			if evt.Has(fsnotify.Create) {
				_ = w.fsw.Add(evt.Name)
			}
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return nil
			}
			slog.Warn("watcher error", "err", err)
		}
	}
}

func (w *Watcher) handleChange(absPath string) {
	// Resolve both root and file through EvalSymlinks so firmlinks don't
	// produce `../..` paths that escape the project root.
	resolvedRoot, err := filepath.EvalSymlinks(w.projectRoot)
	if err != nil {
		resolvedRoot = filepath.Clean(w.projectRoot)
	}
	resolvedFile, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		resolvedFile = filepath.Clean(absPath)
	}
	relPath, err := filepath.Rel(resolvedRoot, resolvedFile)
	if err != nil {
		return
	}
	relPath = filepath.ToSlash(relPath)
	if relPath == ".." || strings.HasPrefix(relPath, "../") || filepath.IsAbs(relPath) {
		// File is outside the project; ignore.
		return
	}

	if err := w.idx.IndexFile(absPath); err != nil {
		// File was deleted or unreadable — remove from index.
		if removeErr := w.idx.DeletePath(relPath); removeErr != nil {
			slog.Warn("watcher: delete from index failed", "path", relPath, "err", removeErr)
		}
	}

	w.hub.Broadcast(hub.Event{
		Type:    "file.changed",
		Payload: map[string]string{"path": relPath},
	})
}

func (w *Watcher) addDirRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		return w.fsw.Add(path)
	})
}

func (w *Watcher) shouldProcess(path string) bool {
	if !strings.HasPrefix(path, w.lifecycleDir) {
		return false
	}
	if !strings.HasSuffix(path, ".md") {
		return false
	}
	base := filepath.Base(path)
	for _, sfx := range []string{"~", ".swp", ".DS_Store"} {
		if strings.HasSuffix(base, sfx) {
			return false
		}
	}
	for _, pfx := range []string{".", "#"} {
		if strings.HasPrefix(base, pfx) {
			return false
		}
	}
	return true
}
