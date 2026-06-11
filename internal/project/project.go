// SPDX-License-Identifier: AGPL-3.0-or-later

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
	"github.com/kaos-control/kaos-control/internal/devops"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/ideachat"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/lock"
	"github.com/kaos-control/kaos-control/internal/release"
	"github.com/kaos-control/kaos-control/internal/scheduler"
	"github.com/kaos-control/kaos-control/internal/triage"
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
	Agents         *agent.Manager       // nil if no agents configured
	IdeaChatStore  *ideachat.Store      // per-project conversational idea-capture sessions
	DevopsRunner   *devops.Runner       // manages active pipeline runs
	DevopsLogs     *devops.LogStore     // persists run logs to ~/.kaos-control/devops/<project>/
	Scheduler      *scheduler.Scheduler // nil until StartScheduler is called
	SchedulerStore *scheduler.Store     // always set after Open()
	TriageMgr      *triage.Manager      // auto-triage of raw idea artifacts
	ReleaseSync    *release.DiskSync    // disk sync for release markdown files

	// watcherDone is closed when the watcher goroutine exits.
	// Close() waits on this before closing the index DB.
	watcherDone <-chan struct{}
}

// OpenOptions configures optional parameters for Open.
type OpenOptions struct {
	MaxConcurrentAgents        int
	MaxConcurrentSchedulerJobs int
	SchedulerRunRetentionDays  int
	OllamaInstances            []config.OllamaInstance // app-level Ollama servers for OllamaDriver
	AgentCfg                   config.AppAgentConfig   // precheck timeout + bypass-permissions flag

	// DevopsLogDir is the base directory for pipeline run logs.
	// Logs are stored at DevopsLogDir/<project-name>/<run_id>.log.
	// If empty, defaults to filepath.Dir(dbDir), placing logs at
	// <appHome>/devops/<project> (e.g. ~/.kaos-control/devops/<project> when
	// dbDir = ~/.kaos-control/data).
	DevopsLogDir string

	// HookServerAddr is the address the hook-helper binary should POST
	// permission requests to (e.g. "127.0.0.1:9600"). Required for
	// claude-mediated agents; ignored by other drivers.
	HookServerAddr string
	// HookBinaryPath is the absolute path to the kaos-control binary used as
	// the hook-helper command. If empty, os.Executable() is used at run time.
	HookBinaryPath string
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

	// Create hub and workflow engine before opening the index so the startup
	// Scan can auto-transition any artifacts with stale status.
	h := hub.New()
	wf := workflow.New(cfg.Transitions)

	dbPath := filepath.Join(dbDir, entry.Name, "index.db")
	idxOpts := []index.Option{
		index.WithIgnore(cfg.Ignore),
		index.WithHub(h),
		index.WithWorkflow(wf),
	}
	if gitRepo != nil {
		idxOpts = append(idxOpts, index.WithGit(gitRepo))
	}
	idx, err := index.Open(dbPath, entry.Path, cfg.Stages, idxOpts...)
	if err != nil {
		return nil, fmt.Errorf("project %q: opening index: %w", entry.Name, err)
	}

	// Prune stale events on startup according to retention config.
	_ = idx.PruneEvents(cfg.Feed.RetentionDays, cfg.Feed.MaxEvents)

	w, err := watcher.New(entry.Path, idx, h, cfg.Ignore...)
	if err != nil {
		slog.Warn("project: failed to create watcher", "name", entry.Name, "err", err)
		w = nil
	}

	// Wire the git-status broadcast callback so the watcher can signal
	// external git state changes (branch checkout, git add) via WebSocket.
	if w != nil && gitRepo != nil {
		w.SetGitStatusCallback(func() {
			if summary, err := gitRepo.Status(); err == nil {
				h.Broadcast(hub.Event{Type: "git.status", Payload: summary})
			}
		})
	}

	locks := lock.New(idx, h)

	// Build the triage manager.
	triageDeps := triage.Deps{
		Idx:         idx,
		Locks:       locks,
		Workflow:    wf,
		Hub:         h,
		Agents:      cfg.Agents,
		ProjectRoot: entry.Path,
		Git:         gitRepo,
	}
	triageMgr := triage.New(triageDeps, triage.Options{})

	// Wire the triage callback to the watcher so raw ideas are triaged on create/modify.
	if w != nil {
		w.SetTriageCallback(func(relPath string) {
			if _, err := triageMgr.Trigger(context.Background(), relPath, triage.TriggerWatcher); err != nil {
				slog.Warn("triage: watcher trigger failed", "path", relPath, "err", err)
			}
		})
	}

	// Startup re-scan: enqueue any raw ideas that were present when the server
	// started (the watcher only covers live changes). Runs in a goroutine so
	// Open returns before potentially-slow LLM calls begin.
	go triage.RescanRaw(context.Background(), triageMgr, idx)

	maxConcurrent := opts.MaxConcurrentAgents
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}

	var agentMgr *agent.Manager
	if len(cfg.Agents) > 0 {
		runsLogDir := filepath.Join(dbDir, entry.Name, "runs")
		agentMgr = agent.New(cfg.Agents, maxConcurrent, idx, gitRepo, h, locks, wf, entry.Path, runsLogDir, opts.OllamaInstances, opts.AgentCfg)
		if opts.HookServerAddr != "" {
			agentMgr.ConfigureHookDriver(opts.HookServerAddr, opts.HookBinaryPath)
		}
	}

	// Determine the base directory for devops run logs.
	// Default to the parent of dbDir so that, with the standard layout where
	// dbDir = ~/.kaos-control/data, logs land at ~/.kaos-control/devops/<project>
	// as specified in the backend plan.
	devopsLogDir := opts.DevopsLogDir
	if devopsLogDir == "" {
		devopsLogDir = filepath.Dir(dbDir)
	}
	logStore := devops.NewLogStore(devopsLogDir)
	devopsRunner := devops.NewRunner()
	devopsRunner.SetEventHook(func(runID, eventType string, payload any) {
		logStore.WriteEvent(entry.Name, runID, eventType, payload)
	})

	// Scheduler store is always created so the HTTP API can serve job CRUD
	// even when no scheduler goroutine is running.
	schedulerStore := scheduler.NewStore(idx.DB())

	// Prune old scheduler runs according to retention config.
	retentionDays := opts.SchedulerRunRetentionDays
	if retentionDays <= 0 {
		retentionDays = 90
	}
	if err := schedulerStore.PruneOldRuns(retentionDays); err != nil {
		slog.Warn("project: pruning old scheduler runs", "name", entry.Name, "err", err)
	}

	// Build the scheduler engine.
	maxSchedulerWorkers := opts.MaxConcurrentSchedulerJobs
	if maxSchedulerWorkers <= 0 {
		maxSchedulerWorkers = 2
	}
	schedulerLogDir := filepath.Join(dbDir, entry.Name, "scheduler-runs")
	sched := scheduler.New(schedulerStore, agentMgr, h, entry.Path, schedulerLogDir, maxSchedulerWorkers)

	releaseExpected := release.NewExpectedEvents()
	releaseSync := release.NewDiskSync(releaseExpected)

	return &Project{
		Entry:          entry,
		Cfg:            cfg,
		Idx:            idx,
		Git:            gitRepo,
		Hub:            h,
		Watcher:        w,
		Workflow:       wf,
		Locks:          locks,
		Agents:         agentMgr,
		IdeaChatStore:  ideachat.NewStore(),
		DevopsRunner:   devopsRunner,
		DevopsLogs:     logStore,
		Scheduler:      sched,
		SchedulerStore: schedulerStore,
		TriageMgr:      triageMgr,
		ReleaseSync:    releaseSync,
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

// StartScheduler launches the scheduler goroutines. It is a no-op if the
// scheduler was not created during Open.
func (p *Project) StartScheduler(ctx context.Context) {
	if p.Scheduler == nil {
		return
	}
	p.Scheduler.Start(ctx)
}

// Close releases resources held by the project.
// It waits for the watcher goroutine to fully stop before closing the index
// so that in-flight debounce callbacks cannot touch the DB after it is closed.
func (p *Project) Close() error {
	// Stop the scheduler first so its goroutines can no longer touch the DB.
	if p.Scheduler != nil {
		p.Scheduler.Stop()
	}
	// Stop the triage manager so in-flight runs complete before the DB closes.
	if p.TriageMgr != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p.TriageMgr.Stop(stopCtx)
	}
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
