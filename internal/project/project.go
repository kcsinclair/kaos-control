// Package project holds per-project runtime state.
package project

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/ideachat"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/lock"
	"github.com/kaos-control/kaos-control/internal/watcher"
	"github.com/kaos-control/kaos-control/internal/workflow"
)

// Project is the runtime services container for one registered project.
type Project struct {
	Entry          *config.ProjectEntry
	Cfg            *config.Project
	Idx            *index.Index
	Git            *kgit.Repo // nil if the project directory is not a git repo
	Hub            *hub.Hub
	Watcher        *watcher.Watcher
	Workflow       *workflow.Engine
	Locks          *lock.Manager
	Agents         *agent.Manager  // nil if no agents configured
	IdeaChatStore  *ideachat.Store // per-project conversational idea-capture sessions

	// watcherDone is closed when the watcher goroutine exits.
	// Close() waits on this before closing the index DB.
	watcherDone <-chan struct{}
}

// OpenOptions configures optional parameters for Open.
type OpenOptions struct {
	MaxConcurrentAgents int
}

// Open loads the project config, opens the SQLite index, scans the lifecycle tree,
// and initialises the git repo wrapper and event hub.
// dbDir is the app-level data directory; per-project DBs live at dbDir/<name>/index.db.
func Open(entry *config.ProjectEntry, dbDir string, opts OpenOptions) (*Project, error) {
	cfg, err := config.LoadProject(entry.Path)
	if err != nil {
		return nil, fmt.Errorf("project %q: loading config: %w", entry.Name, err)
	}

	// Open git first so the index can use it for created-date backfill during scan.
	var gitRepo *kgit.Repo
	if kgit.IsRepo(entry.Path) {
		gitRepo, err = kgit.Open(entry.Path)
		if err != nil {
			slog.Warn("project: failed to open git repo", "name", entry.Name, "err", err)
		}
	} else {
		slog.Info("project: not a git repo, write operations will not commit", "name", entry.Name)
	}

	dbPath := filepath.Join(dbDir, entry.Name, "index.db")
	idxOpts := []index.Option{index.WithIgnore(cfg.Ignore)}
	if gitRepo != nil {
		idxOpts = append(idxOpts, index.WithGit(gitRepo))
	}
	idx, err := index.Open(dbPath, entry.Path, cfg.Stages, idxOpts...)
	if err != nil {
		return nil, fmt.Errorf("project %q: opening index: %w", entry.Name, err)
	}

	h := hub.New()

	w, err := watcher.New(entry.Path, idx, h, cfg.Ignore...)
	if err != nil {
		slog.Warn("project: failed to create watcher", "name", entry.Name, "err", err)
		w = nil
	}

	wf := workflow.New(cfg.Transitions)

	locks := lock.New(idx, h)

	maxConcurrent := opts.MaxConcurrentAgents
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}

	var agentMgr *agent.Manager
	if len(cfg.Agents) > 0 {
		runsLogDir := filepath.Join(dbDir, entry.Name, "runs")
		agentMgr = agent.New(cfg.Agents, maxConcurrent, idx, gitRepo, h, locks, entry.Path, runsLogDir)
	}

	return &Project{
		Entry:         entry,
		Cfg:           cfg,
		Idx:           idx,
		Git:           gitRepo,
		Hub:           h,
		Watcher:       w,
		Workflow:      wf,
		Locks:         locks,
		Agents:        agentMgr,
		IdeaChatStore: ideachat.NewStore(),
	}, nil
}

// StartWatcher launches the fsnotify watcher goroutine.
// It returns immediately; the watcher runs until ctx is cancelled.
// Close() will wait for the goroutine to fully exit before closing the index,
// preventing "sql: database is closed" errors from in-flight debounce callbacks.
func (p *Project) StartWatcher(ctx context.Context) {
	if p.Watcher == nil {
		return
	}
	done := make(chan struct{})
	p.watcherDone = done
	go func() {
		defer close(done)
		if err := p.Watcher.Start(ctx); err != nil {
			slog.Error("watcher stopped with error", "project", p.Entry.Name, "err", err)
		}
	}()
}

// StartLockReaper launches the lock reaper goroutine.
func (p *Project) StartLockReaper(ctx context.Context) {
	p.Locks.StartReaper(ctx)
}

// StartSessionReaper launches the idea-chat session reaper goroutine.
// The reaper exits when ctx is cancelled.
func (p *Project) StartSessionReaper(ctx context.Context) {
	p.IdeaChatStore.StartReaper(ctx)
}

// Close releases resources held by the project.
// It waits for the watcher goroutine to fully stop before closing the index
// so that in-flight debounce callbacks cannot touch the DB after it is closed.
func (p *Project) Close() error {
	if p.watcherDone != nil {
		select {
		case <-p.watcherDone:
		case <-time.After(5 * time.Second):
			slog.Warn("project: timed out waiting for watcher to stop", "name", p.Entry.Name)
		}
	}
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
