// SPDX-License-Identifier: AGPL-3.0-or-later

// Package lock manages per-lineage editor/agent locks with SQLite persistence.
package lock

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
)

// ErrLocked is returned when a lineage is already locked.
var ErrLocked = errors.New("lineage already locked")

// Manager is the per-project lineage lock service.
type Manager struct {
	mu  sync.Mutex
	idx *index.Index
	hub *hub.Hub
}

// New creates a lock manager backed by idx. Existing locks in the DB are
// treated as active (crash recovery: they will be reaped if stale).
func New(idx *index.Index, h *hub.Hub) *Manager {
	return &Manager{idx: idx, hub: h}
}

// Acquire tries to acquire an exclusive lock on lineage for the given holder
// (user email or run_id) and kind ("editor" or "agent").
// Returns ErrLocked if already held.
func (m *Manager) Acquire(lineage, holder, kind string) (*index.LockRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.idx.AcquireLock(lineage, holder, kind); err != nil {
		if errors.Is(err, index.ErrLocked) {
			return nil, ErrLocked
		}
		return nil, err
	}
	row, err := m.idx.GetLock(lineage)
	if err != nil {
		return nil, err
	}
	m.hub.Broadcast(hub.Event{
		Type:    "lock.acquired",
		Payload: map[string]any{"lineage": lineage, "holder": holder, "kind": kind},
	})
	return row, nil
}

// Release removes the lock for lineage.
func (m *Manager) Release(lineage string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.idx.ReleaseLock(lineage); err != nil {
		return err
	}
	m.hub.Broadcast(hub.Event{
		Type:    "lock.released",
		Payload: map[string]any{"lineage": lineage},
	})
	return nil
}

// Heartbeat refreshes the TTL for the given lineage lock.
func (m *Manager) Heartbeat(lineage string) error {
	return m.idx.HeartbeatLock(lineage)
}

// Get returns the current lock for lineage, or nil if unlocked.
func (m *Manager) Get(lineage string) (*index.LockRow, error) {
	return m.idx.GetLock(lineage)
}

// staleTestThreshold is how long a test artifact may stay in-qa before a
// test.stale WebSocket event is broadcast.
const staleTestThreshold = 60 * time.Minute

// StartReaper launches a goroutine that forcibly releases stale locks every 60 s.
// Locks are considered stale when last_heartbeat is older than 5 minutes.
// The same tick also checks for test artifacts that have been in in-qa for more
// than 60 minutes and broadcasts a test.stale WebSocket event for each.
func (m *Manager) StartReaper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.reap()
				m.checkStaleTestArtifacts()
			}
		}
	}()
}

// checkStaleTestArtifacts queries for test artifacts in in-qa status and
// broadcasts a test.stale event for any that have been there for over 60
// minutes. Called once per reaper tick (every 60 s).
func (m *Manager) checkStaleTestArtifacts() {
	rows, _, err := m.idx.List(index.Filter{Type: "test", Status: "in-qa", Unlimited: true})
	if err != nil {
		slog.Warn("lock reaper: querying stale test artifacts", "err", err)
		return
	}
	now := time.Now()
	for _, row := range rows {
		age := now.Sub(row.Mtime)
		if age < staleTestThreshold {
			continue
		}
		slog.Warn("test artifact has been in in-qa for over 60 minutes",
			"path", row.Path, "age", age.Round(time.Second).String())
		m.hub.Broadcast(hub.Event{
			Type: "test.stale",
			Payload: map[string]any{
				"path":    row.Path,
				"lineage": row.Lineage,
				"age_s":   int64(age.Seconds()),
				"age":     age.Round(time.Second).String(),
			},
		})
	}
}

func (m *Manager) reap() {
	m.mu.Lock()
	defer m.mu.Unlock()
	reaped, err := m.idx.ReapLocks(5 * time.Minute)
	if err != nil {
		slog.Warn("lock reaper error", "err", err)
		return
	}
	for _, lineage := range reaped {
		slog.Warn("lock reaper: forcibly released stale lock", "lineage", lineage)
		m.hub.Broadcast(hub.Event{
			Type:    "lock.released",
			Payload: map[string]any{"lineage": lineage, "reason": "heartbeat_timeout"},
		})
	}
}
