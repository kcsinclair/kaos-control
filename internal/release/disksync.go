// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// ExpectedEvents is a concurrent-safe set of absolute file paths that were
// written by the API. The watcher checks this before emitting WS events so
// API-driven writes do not produce spurious release.changed broadcasts.
type ExpectedEvents struct {
	mu    sync.Mutex
	paths map[string]struct{}
}

// NewExpectedEvents creates an empty ExpectedEvents registry.
func NewExpectedEvents() *ExpectedEvents {
	return &ExpectedEvents{paths: make(map[string]struct{})}
}

// Expect marks absPath as an expected file-system event.
func (e *ExpectedEvents) Expect(absPath string) {
	e.mu.Lock()
	e.paths[absPath] = struct{}{}
	e.mu.Unlock()
}

// Consume checks whether absPath is expected and, if so, removes it and
// returns true. Returns false if the path is unknown.
func (e *ExpectedEvents) Consume(absPath string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.paths[absPath]; ok {
		delete(e.paths, absPath)
		return true
	}
	return false
}

// DiskSync writes, renames, and deletes release markdown files atomically.
// It records each operation in ExpectedEvents so the watcher can suppress
// the resulting fsnotify event.
type DiskSync struct {
	expected *ExpectedEvents
}

// NewDiskSync creates a DiskSync backed by expected.
func NewDiskSync(expected *ExpectedEvents) *DiskSync {
	return &DiskSync{expected: expected}
}

// relPath returns the project-relative path for a release slug.
func relPath(slug string) string {
	return fmt.Sprintf("lifecycle/releases/%s.md", slug)
}

// Write atomically writes the release markdown file for r.
// It uses a *.tmp file + rename to guarantee that the watcher never observes
// a partial write. The target absolute path is registered in ExpectedEvents
// before the write so the watcher treats the resulting event as a no-op.
// Returns the project-relative path on success.
func (d *DiskSync) Write(projectRoot string, r *Release) (string, error) {
	rel := relPath(r.Slug)
	absPath, err := sandbox.Resolve(projectRoot, rel)
	if err != nil {
		return "", fmt.Errorf("resolving release path %q: %w", rel, err)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("creating releases dir: %w", err)
	}

	f := &File{
		Title:     r.Name,
		Slug:      r.Slug,
		Status:    r.Status,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
		UpdatedAt: r.UpdatedAt,
	}
	data, err := f.Marshal()
	if err != nil {
		return "", fmt.Errorf("marshalling release file: %w", err)
	}

	tmpPath := absPath + ".tmp"
	d.expected.Expect(absPath)
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		d.expected.Consume(absPath) // undo
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		d.expected.Consume(absPath)
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("atomic rename: %w", err)
	}
	return rel, nil
}

// Rename writes the release to its new slug path and removes the old slug
// file. Both paths are registered in ExpectedEvents. Returns the new
// project-relative path on success.
func (d *DiskSync) Rename(projectRoot, oldSlug, newSlug string, r *Release) (string, error) {
	// Write the new file first; on failure the old file is left intact.
	newRel, err := d.Write(projectRoot, r)
	if err != nil {
		return "", err
	}

	// Remove the old file only after the new one is safely on disk.
	oldRel := relPath(oldSlug)
	oldAbs, err := sandbox.Resolve(projectRoot, oldRel)
	if err != nil {
		return newRel, nil // new file exists; best-effort on old removal
	}
	d.expected.Expect(oldAbs)
	if err := os.Remove(oldAbs); err != nil && !os.IsNotExist(err) {
		d.expected.Consume(oldAbs)
		// Non-fatal: the new file was written successfully.
	}
	return newRel, nil
}

// Delete removes the release markdown file for slug from disk.
// The path is registered in ExpectedEvents before removal.
func (d *DiskSync) Delete(projectRoot, slug string) error {
	rel := relPath(slug)
	absPath, err := sandbox.Resolve(projectRoot, rel)
	if err != nil {
		return fmt.Errorf("resolving release path %q: %w", rel, err)
	}
	d.expected.Expect(absPath)
	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		d.expected.Consume(absPath)
		return fmt.Errorf("removing release file: %w", err)
	}
	return nil
}
