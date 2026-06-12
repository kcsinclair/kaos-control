// SPDX-License-Identifier: AGPL-3.0-or-later

package watcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/release"
)

// memStore is an in-memory ReleaseStore for testing.
type memStore struct {
	mu      sync.Mutex
	bySlug  map[string]*release.Release
	deleted []string
}

func newMemStore() *memStore {
	return &memStore{bySlug: make(map[string]*release.Release)}
}

func (m *memStore) UpsertBySlug(r *release.Release) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *r
	m.bySlug[r.Slug] = &cp
	return nil
}

func (m *memStore) DeleteBySlug(projectID, slug string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.bySlug, slug)
	m.deleted = append(m.deleted, slug)
	return nil
}

func validReleaseMD(title, status string) []byte {
	return []byte("---\ntitle: " + title + "\ntype: release\nstatus: " + status +
		"\nupdated_at: 2026-01-01T00:00:00Z\n---\n")
}

// drainEvents reads up to maxWait duration worth of events from the hub channel.
func drainEvents(ch <-chan []byte, maxWait time.Duration) []map[string]any {
	var out []map[string]any
	deadline := time.After(maxWait)
	for {
		select {
		case data := <-ch:
			var evt map[string]any
			if json.Unmarshal(data, &evt) == nil {
				out = append(out, evt)
			}
		case <-deadline:
			return out
		}
	}
}

func TestReleaseHandler_WriteFile_UpsertAndBroadcast(t *testing.T) {
	dir := t.TempDir()
	absPath := filepath.Join(dir, "q1-2026.md")
	if err := os.WriteFile(absPath, validReleaseMD("Q1 2026", "planned"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := newMemStore()
	expected := release.NewExpectedEvents()
	h := hub.New()
	ch := make(chan []byte, 16)
	h.Register(ch)

	rh := NewReleaseHandler(store, "proj", expected, h)
	rh.Handle(absPath)

	// Row should be upserted.
	store.mu.Lock()
	r, ok := store.bySlug["q1-2026"]
	store.mu.Unlock()
	if !ok {
		t.Fatal("expected release to be upserted in store")
	}
	if r.Name != "Q1 2026" {
		t.Errorf("Name = %q, want %q", r.Name, "Q1 2026")
	}

	// WS event should be broadcast.
	evts := drainEvents(ch, 50*time.Millisecond)
	if len(evts) == 0 {
		t.Error("expected release.changed WS event to be broadcast")
	}
	if evts[0]["type"] != "release.changed" {
		t.Errorf("event type = %q, want release.changed", evts[0]["type"])
	}
}

func TestReleaseHandler_APIOriginated_NoWSEvent(t *testing.T) {
	dir := t.TempDir()
	absPath := filepath.Join(dir, "q1-2026.md")
	if err := os.WriteFile(absPath, validReleaseMD("Q1 2026", "planned"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := newMemStore()
	expected := release.NewExpectedEvents()
	h := hub.New()
	ch := make(chan []byte, 16)
	h.Register(ch)

	// Mark this path as API-originated. In production DiskSync.Expect records
	// the symlink-resolved path (via sandbox.Resolve), and Handle resolves the
	// fsnotify path the same way before Consume — so the keys match. Mirror that
	// here by resolving before Expect; otherwise on macOS the temp dir's
	// /var → /private/var symlink makes the keys diverge and the event is not
	// suppressed.
	resolvedPath := absPath
	if r, err := filepath.EvalSymlinks(absPath); err == nil {
		resolvedPath = r
	}
	expected.Expect(resolvedPath)

	rh := NewReleaseHandler(store, "proj", expected, h)
	rh.Handle(absPath)

	// Store should NOT be updated (Consume was called, handler returns early).
	store.mu.Lock()
	_, ok := store.bySlug["q1-2026"]
	store.mu.Unlock()
	if ok {
		t.Error("expected no upsert for API-originated event")
	}

	// No WS event.
	evts := drainEvents(ch, 30*time.Millisecond)
	if len(evts) != 0 {
		t.Errorf("expected no WS events for API-originated write, got %d", len(evts))
	}
}

func TestReleaseHandler_DeleteFile_DeletesRowAndBroadcasts(t *testing.T) {
	dir := t.TempDir()
	absPath := filepath.Join(dir, "q1-2026.md")
	// File does NOT exist — simulates deletion.

	store := newMemStore()
	store.bySlug["q1-2026"] = &release.Release{Slug: "q1-2026"}
	expected := release.NewExpectedEvents()
	h := hub.New()
	ch := make(chan []byte, 16)
	h.Register(ch)

	rh := NewReleaseHandler(store, "proj", expected, h)
	rh.Handle(absPath)

	store.mu.Lock()
	_, still := store.bySlug["q1-2026"]
	store.mu.Unlock()
	if still {
		t.Error("expected release to be deleted from store")
	}

	evts := drainEvents(ch, 50*time.Millisecond)
	if len(evts) == 0 {
		t.Error("expected WS event after delete")
	}
	if evts[0]["type"] != "release.changed" {
		t.Errorf("event type = %q, want release.changed", evts[0]["type"])
	}
}

func TestReleaseHandler_InvalidFile_Skipped(t *testing.T) {
	dir := t.TempDir()
	absPath := filepath.Join(dir, "bad.md")
	bad := []byte("---\ntitle: Bad\ntype: release\nstatus: planned\n" +
		"start_date: 2026-06-01\nend_date: 2026-01-01\nupdated_at: 2026-01-01T00:00:00Z\n---\n")
	if err := os.WriteFile(absPath, bad, 0o644); err != nil {
		t.Fatal(err)
	}

	store := newMemStore()
	expected := release.NewExpectedEvents()
	h := hub.New()
	ch := make(chan []byte, 16)
	h.Register(ch)

	rh := NewReleaseHandler(store, "proj", expected, h)
	rh.Handle(absPath)

	store.mu.Lock()
	count := len(store.bySlug)
	store.mu.Unlock()
	if count != 0 {
		t.Errorf("expected no rows in store after invalid file, got %d", count)
	}

	evts := drainEvents(ch, 30*time.Millisecond)
	if len(evts) != 0 {
		t.Errorf("expected no WS events for invalid file, got %d", len(evts))
	}
}
