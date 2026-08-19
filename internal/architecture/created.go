// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"os"
	"path/filepath"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// createdFor returns the created: value (RFC3339) to stamp on the artifact at
// absPath: the value already on disk if the file carries one — so re-writing an
// existing artifact stays idempotent — otherwise the current time. This lets
// the ADR/summary/promotion primitives give every artifact they author a
// creation date without breaking their idempotent-rewrite contract.
func createdFor(absPath string) string {
	if raw, err := os.ReadFile(absPath); err == nil {
		if c := artifact.Parse(raw, filepath.Base(absPath), time.Time{}).FM.Created; c != "" {
			return c
		}
	}
	return time.Now().Format(time.RFC3339)
}
