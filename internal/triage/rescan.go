// SPDX-License-Identifier: AGPL-3.0-or-later

package triage

import (
	"context"
	"log/slog"

	"github.com/kaos-control/kaos-control/internal/index"
)

// RescanRaw queries the index for all raw idea artifacts and enqueues each
// through mgr so they are triaged on startup. Errors per artifact are logged
// at warn level and otherwise swallowed so a single bad artifact does not
// block the rest.
func RescanRaw(ctx context.Context, mgr *Manager, idx IndexStore) {
	rows, _, err := idx.List(index.Filter{Type: "idea", Status: "raw", Unlimited: true})
	if err != nil {
		slog.Warn("triage rescan: list raw ideas failed", "err", err)
		return
	}
	for _, row := range rows {
		if _, err := mgr.Trigger(ctx, row.Path, TriggerStartup); err != nil {
			slog.Warn("triage rescan: trigger failed", "path", row.Path, "err", err)
		}
	}
}
