// SPDX-License-Identifier: AGPL-3.0-or-later

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
	"github.com/kaos-control/kaos-control/internal/config"
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
	ignore       []string // glob patterns; base name matched via config.ShouldIgnore

	// gitStatusFn is called (with 150 ms debounce) when .git/HEAD or
	// .git/index change. May be nil when git is not available.
	gitStatusFn func()

	// triageFn is called after a successful re-index when the freshly
	// indexed artifact has type "idea" and status "raw". May be nil.
	triageFn func(relPath string)

	// releaseFn is called (debounced) for .md files under lifecycle/releases/.
	// May be nil; when set, lifecycle/releases/ events are dispatched here
	// instead of the standard artifact indexer.
	releaseFn func(absPath string)
}

// New creates a Watcher but does not start it. The optional ignore variadic
// receives the project's ignore-pattern list (config.Project.Ignore).
func New(projectRoot string, idx *index.Index, h *hub.Hub, ignore ...string) (*Watcher, error) {
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
		ignore:       ignore,
	}, nil
}

// SetGitStatusCallback registers a callback that is invoked (debounced, 150 ms)
// whenever .git/HEAD or .git/index changes. It must be called before Start.
func (w *Watcher) SetGitStatusCallback(fn func()) {
	w.gitStatusFn = fn
}

// SetTriageCallback registers a callback that is invoked in a goroutine after
// a successful re-index when the newly indexed artifact has type "idea" and
// status "raw". The triage Manager handles concurrency and deduplication
// internally, so the callback may fire more than once for the same path.
// It must be called before Start.
func (w *Watcher) SetTriageCallback(fn func(relPath string)) {
	w.triageFn = fn
}

// SetReleaseCallback registers a callback that is invoked (debounced) for
// every .md file event under lifecycle/releases/. When set, such events are
// dispatched to fn instead of the standard artifact indexer. fn receives the
// absolute path of the changed file. It must be called before Start.
func (w *Watcher) SetReleaseCallback(fn func(absPath string)) {
	w.releaseFn = fn
}

// Start begins watching the lifecycle/ tree and blocks until ctx is cancelled.
// It does not return until all in-flight handleChange callbacks have completed,
// ensuring it is safe to close the index immediately after Start returns.
func (w *Watcher) Start(ctx context.Context) error {
	if err := w.addDirRecursive(w.lifecycleDir); err != nil {
		return err
	}

	// Watch individual .git files so we detect external branch checkouts and
	// index updates. Errors are non-fatal (project may not be a git repo).
	gitHEAD := filepath.Join(w.projectRoot, ".git", "HEAD")
	gitIndex := filepath.Join(w.projectRoot, ".git", "index")
	_ = w.fsw.Add(gitHEAD)
	_ = w.fsw.Add(gitIndex)

	// Debounce: map from path to active timer.
	timers := map[string]*time.Timer{}
	var mu sync.Mutex
	// wg tracks in-flight handleChange calls launched by time.AfterFunc.
	var wg sync.WaitGroup

	fire := func(path string, fn func()) {
		mu.Lock()
		defer mu.Unlock()
		if t, ok := timers[path]; ok {
			t.Reset(150 * time.Millisecond)
			return
		}
		wg.Add(1)
		timers[path] = time.AfterFunc(150*time.Millisecond, func() {
			defer wg.Done()
			mu.Lock()
			delete(timers, path)
			mu.Unlock()
			fn()
		})
	}

	// On exit: stop any pending timers, wait for in-flight handlers, then
	// close the fsnotify watcher.  This guarantees the index is not touched
	// after Start returns, so callers can safely close the DB immediately.
	defer func() {
		mu.Lock()
		for path, t := range timers {
			if t.Stop() {
				// Timer hadn't fired yet; decrement the Add we pre-incremented.
				wg.Done()
			}
			delete(timers, path)
		}
		mu.Unlock()
		wg.Wait()
		w.fsw.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}
			// Git-state files: debounce and call the git status callback.
			if w.gitStatusFn != nil && (evt.Name == gitHEAD || evt.Name == gitIndex) {
				fire(evt.Name, w.gitStatusFn)
				continue
			}
			if w.shouldProcess(evt.Name) {
				if w.releaseFn != nil && w.isReleaseFile(evt.Name) {
					path := evt.Name // capture for closure
					fn := w.releaseFn
					fire(evt.Name, func() { fn(path) })
				} else {
					fire(evt.Name, func() { w.handleChange(evt.Name) })
				}
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
		// File no longer exists (deleted). EvalSymlinks would partially resolve
		// the root (e.g. /var → /private/var on macOS) but leave the file path
		// raw, causing filepath.Rel to produce "../.." paths.  Use Clean for
		// both so the prefixes are consistent.
		resolvedFile = filepath.Clean(absPath)
		resolvedRoot = filepath.Clean(w.projectRoot)
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

	// Check if the path is new before indexing (for defect_raised detection).
	existingRow, _ := w.idx.Get(relPath)
	isNew := existingRow == nil

	if err := w.idx.IndexFile(absPath); err != nil {
		// File was deleted or unreadable — remove from index.
		if removeErr := w.idx.DeletePath(relPath); removeErr != nil {
			slog.Warn("watcher: delete from index failed", "path", relPath, "err", removeErr)
		}
	} else {
		if isNew {
			// Detect newly created defect artifacts.
			if row, err := w.idx.Get(relPath); err == nil && row != nil && row.Type == "defect" {
				artifactPath := relPath
				summary := "Defect raised: " + row.FM.Title
				feedEvent := &index.EventRow{
					EventType:    "defect_raised",
					Timestamp:    time.Now().Unix(),
					Actor:        "system",
					ArtifactPath: &artifactPath,
					Summary:      summary,
				}
				if err := w.idx.InsertEvent(feedEvent); err == nil {
					w.hub.Broadcast(hub.Event{Type: "feed.new", Payload: feedEvent})
				}
			}
		}
		// Fire the triage callback for raw idea artifacts (created or modified).
		if w.triageFn != nil {
			if row, err := w.idx.Get(relPath); err == nil && row != nil &&
				row.Type == "idea" && row.Status == "raw" {
				fn := w.triageFn
				go fn(relPath)
			}
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

func (w *Watcher) isReleaseFile(path string) bool {
	releasesDir := filepath.Join(w.lifecycleDir, "releases")
	return strings.HasPrefix(path, releasesDir+string(filepath.Separator)) ||
		path == releasesDir
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
	return !config.ShouldIgnore(path, w.ignore)
}
