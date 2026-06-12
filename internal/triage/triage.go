// SPDX-License-Identifier: AGPL-3.0-or-later

// Package triage implements automatic triage of raw idea artifacts.
// It watches for status:raw / type:idea artifacts under lifecycle/ideas/,
// calls the LLM via ideachat.Generate, rewrites the artifact body and
// frontmatter, and records each attempt as an agent run.
package triage

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kaos-control/kaos-control/internal/config"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/lock"
	"github.com/kaos-control/kaos-control/internal/workflow"
)

// failureCooldown is how long a failed run's relPath stays in the in-flight map
// after the goroutine completes. During this window the entry acts as a zombie:
// any new Trigger call coalesces onto the already-finished run instead of
// starting a fresh attempt, enforcing the no-retry-on-failure policy.
const failureCooldown = 5 * time.Second

// TriggerSource identifies what initiated a triage run.
type TriggerSource string

const (
	TriggerWatcher TriggerSource = "watcher"
	TriggerStartup TriggerSource = "startup"
	TriggerAPI     TriggerSource = "api"
)

// ErrIneligible is returned by Trigger when the artifact does not meet
// the criteria for triage.
type ErrIneligible struct {
	Reason string // not_in_ideas_dir | wrong_type | wrong_status | not_indexed
}

func (e ErrIneligible) Error() string { return "not eligible for triage: " + e.Reason }

// ErrLocked is returned by Trigger when the lineage lock is already held.
var ErrLocked = errors.New("lineage is locked")

// ErrBusy is returned by Trigger when the context is cancelled while
// waiting for a concurrency slot.
var ErrBusy = errors.New("triage manager at capacity")

// IndexStore is the minimal index interface required by the triage package.
// Satisfied by *index.Index.
type IndexStore interface {
	Get(relPath string) (*index.ArtifactRow, error)
	List(f index.Filter) ([]*index.ArtifactRow, int, error)
	Labels() ([]string, error)
	IndexFile(absPath string) error
	InsertAgentRun(r *index.AgentRunRow) error
	UpdateAgentRun(r *index.AgentRunRow) error
}

// LockManager is the minimal lock interface required by the triage package.
// Satisfied by *lock.Manager.
type LockManager interface {
	Acquire(lineage, holder, kind string) (*index.LockRow, error)
	Release(lineage string) error
}

// Deps holds external dependencies for a triage Manager.
type Deps struct {
	Idx         IndexStore
	Locks       LockManager
	Workflow    *workflow.Engine
	Hub         *hub.Hub
	Agents      []config.AgentConfig
	ProjectRoot string
	Git         *kgit.Repo // optional; nil skips git commits
}

// Options configures a triage Manager.
type Options struct {
	MaxConcurrent int    // concurrency cap; default 2
	AgentName     string // agent entry name; default "idea-triage"

	// executeHook overrides the built-in execute function when non-nil.
	// Used only in tests (same package); never set in production code.
	executeHook func(ctx context.Context, runID, relPath, lineage string, trigger TriggerSource) error
}

func (o Options) withDefaults() Options {
	if o.MaxConcurrent <= 0 {
		o.MaxConcurrent = 2
	}
	if o.AgentName == "" {
		o.AgentName = "idea-triage"
	}
	return o
}

// Manager orchestrates automatic triage of raw idea artifacts.
type Manager struct {
	deps Deps
	opts Options

	mu       sync.Mutex
	inFlight map[string]string       // relPath → runID (active or zombie after failure)
	cooling  map[string]*time.Timer  // relPath → timer that clears the zombie inFlight entry
	sem      chan struct{}            // concurrency cap (size = opts.MaxConcurrent)
	wg       sync.WaitGroup          // tracks in-flight goroutines
}

// New creates a Manager. Callers must call Stop when done to release resources.
func New(deps Deps, opts Options) *Manager {
	o := opts.withDefaults()
	return &Manager{
		deps:     deps,
		opts:     o,
		inFlight: make(map[string]string),
		cooling:  make(map[string]*time.Timer),
		sem:      make(chan struct{}, o.MaxConcurrent),
	}
}

// Trigger attempts to start (or coalesce onto) a triage run for relPath.
//
// Return values:
//   - (runID, nil) on success or coalescing onto an existing in-flight run.
//   - ("", ErrIneligible{}) when the artifact is not eligible.
//   - ("", ErrLocked) when the lineage lock is already held.
//   - ("", ErrBusy) when the context expires while waiting for a semaphore slot.
func (m *Manager) Trigger(ctx context.Context, relPath string, trigger TriggerSource) (string, error) {
	// Fast path: already in flight — return the existing run ID.
	m.mu.Lock()
	if runID, ok := m.inFlight[relPath]; ok {
		m.mu.Unlock()
		return runID, nil
	}
	m.mu.Unlock()

	// Eligibility check (cheap; before consuming a concurrency slot).
	row, reason, err := eligible(ctx, m.deps.Idx, relPath)
	if err != nil {
		return "", err
	}
	if row == nil {
		return "", ErrIneligible{Reason: reason}
	}
	lineage := row.Lineage

	// Acquire a concurrency slot (blocks until available or ctx is done).
	select {
	case m.sem <- struct{}{}:
	case <-ctx.Done():
		return "", ErrBusy
	}

	// Acquire the lineage write-lock before any mutation.
	runID := newRunID()
	if _, lockErr := m.deps.Locks.Acquire(lineage, runID, "agent"); lockErr != nil {
		<-m.sem
		if isLockConflict(lockErr) {
			// The lock may be held by an in-flight triage run for this same
			// path. If so, coalesce onto that run rather than returning an error.
			m.mu.Lock()
			existingID, ok := m.inFlight[relPath]
			m.mu.Unlock()
			if ok {
				return existingID, nil
			}
			return "", ErrLocked
		}
		return "", lockErr
	}

	// Register as in-flight; re-check under the mutex to close the narrow race
	// between the initial fast-path check and here.
	m.mu.Lock()
	if existing, ok := m.inFlight[relPath]; ok {
		m.mu.Unlock()
		_ = m.deps.Locks.Release(lineage)
		<-m.sem
		return existing, nil
	}
	m.inFlight[relPath] = runID
	m.mu.Unlock()

	slog.Info("triage started",
		"path", relPath,
		"lineage", lineage,
		"run_id", runID,
		"trigger", string(trigger),
	)

	// Launch the run goroutine.
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		var runErr error
		defer func() {
			<-m.sem
			_ = m.deps.Locks.Release(lineage)
			m.mu.Lock()
			if runErr != nil {
				// Failed run: keep the inFlight entry as a zombie for failureCooldown.
				// Any Trigger call during this window coalesces onto the already-finished
				// run instead of starting a new attempt (no-retry policy).
				timer := time.AfterFunc(failureCooldown, func() {
					m.mu.Lock()
					delete(m.inFlight, relPath)
					delete(m.cooling, relPath)
					m.mu.Unlock()
				})
				m.cooling[relPath] = timer
			} else {
				delete(m.inFlight, relPath)
			}
			m.mu.Unlock()
		}()
		// Recover from panics so the deferred cleanup above always runs.
		defer func() {
			if r := recover(); r != nil {
				slog.Error("triage goroutine panicked",
					"path", relPath,
					"lineage", lineage,
					"run_id", runID,
					"panic", r,
				)
			}
		}()

		if m.opts.executeHook != nil {
			runErr = m.opts.executeHook(ctx, runID, relPath, lineage, trigger)
		} else {
			runErr = m.execute(ctx, runID, relPath, lineage, trigger)
		}
	}()

	return runID, nil
}

// Stop waits for all in-flight triage runs to complete, then cancels any pending
// failure-cooldown timers. It returns when all goroutines have exited or ctx is cancelled.
func (m *Manager) Stop(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
	// Cancel pending failure-cooldown timers and purge zombie in-flight entries.
	m.mu.Lock()
	for _, timer := range m.cooling {
		timer.Stop()
	}
	m.inFlight = make(map[string]string)
	m.cooling = make(map[string]*time.Timer)
	m.mu.Unlock()
}

// isLockConflict reports whether err indicates a lineage is already locked.
func isLockConflict(err error) bool {
	return errors.Is(err, lock.ErrLocked)
}

// newRunID generates a new unique run identifier.
func newRunID() string {
	return uuid.New().String()
}
