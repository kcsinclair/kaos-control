// Package project holds per-project runtime state.
package project

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/kaos-control/kaos-control/internal/config"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/watcher"
)

// Project is the runtime services container for one registered project.
type Project struct {
	Entry   *config.ProjectEntry
	Cfg     *config.Project
	Idx     *index.Index
	Git     *kgit.Repo // nil if the project directory is not a git repo
	Hub     *hub.Hub
	Watcher *watcher.Watcher
}

// Open loads the project config, opens the SQLite index, scans the lifecycle tree,
// and initialises the git repo wrapper and event hub.
// dbDir is the app-level data directory; per-project DBs live at dbDir/<name>/index.db.
func Open(entry *config.ProjectEntry, dbDir string) (*Project, error) {
	cfg, err := config.LoadProject(entry.Path)
	if err != nil {
		return nil, fmt.Errorf("project %q: loading config: %w", entry.Name, err)
	}

	dbPath := filepath.Join(dbDir, entry.Name, "index.db")
	idx, err := index.Open(dbPath, entry.Path, cfg.Stages)
	if err != nil {
		return nil, fmt.Errorf("project %q: opening index: %w", entry.Name, err)
	}

	h := hub.New()

	w, err := watcher.New(entry.Path, idx, h)
	if err != nil {
		// Non-fatal: log and continue without file watching.
		slog.Warn("project: failed to create watcher", "name", entry.Name, "err", err)
		w = nil
	}

	var gitRepo *kgit.Repo
	if kgit.IsRepo(entry.Path) {
		gitRepo, err = kgit.Open(entry.Path)
		if err != nil {
			slog.Warn("project: failed to open git repo", "name", entry.Name, "err", err)
		}
	} else {
		slog.Info("project: not a git repo, write operations will not commit", "name", entry.Name)
	}

	return &Project{Entry: entry, Cfg: cfg, Idx: idx, Git: gitRepo, Hub: h, Watcher: w}, nil
}

// StartWatcher launches the fsnotify watcher goroutine.
// It returns immediately; the watcher runs until ctx is cancelled.
func (p *Project) StartWatcher(ctx context.Context) {
	if p.Watcher == nil {
		return
	}
	go func() {
		if err := p.Watcher.Start(ctx); err != nil {
			slog.Error("watcher stopped with error", "project", p.Entry.Name, "err", err)
		}
	}()
}

// Close releases resources held by the project.
func (p *Project) Close() error {
	return p.Idx.Close()
}

// LifecycleDir returns the absolute path to the lifecycle/ directory.
func (p *Project) LifecycleDir() string {
	return filepath.Join(p.Entry.Path, "lifecycle")
}

// BranchForLineage returns the branch name for a lineage using the project's template.
func (p *Project) BranchForLineage(lineage, slug string) string {
	return kgit.BranchNameFor(p.Cfg.Git.BranchTemplate, slug, lineage)
}
