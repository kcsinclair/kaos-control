// SPDX-License-Identifier: AGPL-3.0-or-later

package watcher

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/release"
)

// ReleaseStore is the subset of release.Store used by the release handler.
type ReleaseStore interface {
	UpsertBySlug(r *release.Release) error
	DeleteBySlug(projectID, slug string) error
}

// ReleaseHandler handles fsnotify events for files in lifecycle/releases/.
// It is wired into the Watcher via SetReleaseCallback.
type ReleaseHandler struct {
	store     ReleaseStore
	projectID string
	expected  *release.ExpectedEvents
	h         *hub.Hub
}

// NewReleaseHandler creates a ReleaseHandler.
func NewReleaseHandler(store ReleaseStore, projectID string, expected *release.ExpectedEvents, h *hub.Hub) *ReleaseHandler {
	return &ReleaseHandler{
		store:     store,
		projectID: projectID,
		expected:  expected,
		h:         h,
	}
}

// Handle is the callback passed to Watcher.SetReleaseCallback.
// absPath is the absolute path to the changed release markdown file.
func (rh *ReleaseHandler) Handle(absPath string) {
	// Suppress events triggered by our own API writes.
	if rh.expected.Consume(absPath) {
		return
	}

	slug := strings.TrimSuffix(filepath.Base(absPath), ".md")

	// Determine if file exists (CREATE/WRITE) or was deleted (REMOVE).
	raw, err := os.ReadFile(absPath)
	if err != nil {
		// File gone — delete from DB.
		if err2 := rh.store.DeleteBySlug(rh.projectID, slug); err2 != nil {
			slog.Warn("release handler: DeleteBySlug failed", "slug", slug, "err", err2)
			return
		}
		rh.h.Broadcast(hub.Event{
			Type: "release.changed",
			Payload: map[string]any{
				"project": rh.projectID,
				"action":  "deleted",
				"slug":    slug,
			},
		})
		return
	}

	// File exists — parse and upsert.
	f, err := release.Parse(filepath.Base(absPath), raw)
	if err != nil {
		slog.Warn("release handler: skipping invalid release file",
			"path", absPath, "err", err)
		return
	}

	r := &release.Release{
		ProjectID: rh.projectID,
		Name:      f.Title,
		Slug:      slug,
		Status:    f.Status,
		StartDate: f.StartDate,
		EndDate:   f.EndDate,
		UpdatedAt: f.UpdatedAt,
	}
	if err := rh.store.UpsertBySlug(r); err != nil {
		slog.Warn("release handler: UpsertBySlug failed", "slug", slug, "err", err)
		return
	}

	rh.h.Broadcast(hub.Event{
		Type: "release.changed",
		Payload: map[string]any{
			"project": rh.projectID,
			"release": r,
		},
	})
}
