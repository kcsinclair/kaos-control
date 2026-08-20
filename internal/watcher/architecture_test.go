// SPDX-License-Identifier: AGPL-3.0-or-later

package watcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
)

// TestWatcher_ArchitectureSubdirsEmitFileChanged pins FR-12 at the watcher
// layer: lifecycle/architecture/ is not a lineage stage (never listed in
// config.Project.Stages), so a disk change under its standards/, decisions/,
// or archive/ subdirectory must still reach the index and broadcast
// file.changed (the same event the frontend re-fetches on, alongside the
// API-write-originated artifact.indexed) through the same unscoped,
// recursive lifecycle/ watch every other artifact relies on — no dedicated
// architecture-watching code is required.
func TestWatcher_ArchitectureSubdirsEmitFileChanged(t *testing.T) {
	root := t.TempDir()
	archDir := filepath.Join(root, "lifecycle", "architecture")
	for _, sub := range []string{"standards", "decisions", "archive"} {
		if err := os.MkdirAll(filepath.Join(archDir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}

	idx, err := index.Open(filepath.Join(root, "idx.db"), root, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	h := hub.New()
	ch := make(chan []byte, 32)
	h.Register(ch)

	w, err := New(root, idx, h)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	startWatcher(t, w)

	standardPath := filepath.Join(archDir, "standards", "secrets.md")
	content := "---\ntitle: Secrets Handling\ntype: doc\nstatus: approved\n---\n\nBody.\n"
	if err := os.WriteFile(standardPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotFileChanged bool
	deadline := time.After(500 * time.Millisecond)
COLLECT:
	for {
		select {
		case data := <-ch:
			var evt struct {
				Type    string            `json:"type"`
				Payload map[string]string `json:"payload"`
			}
			if json.Unmarshal(data, &evt) == nil && evt.Type == "file.changed" &&
				evt.Payload["path"] == "lifecycle/architecture/standards/secrets.md" {
				gotFileChanged = true
				break COLLECT
			}
		case <-deadline:
			break COLLECT
		}
	}
	if !gotFileChanged {
		t.Fatal("expected file.changed for the new standard; got none within 500 ms")
	}

	row, err := idx.Get("lifecycle/architecture/standards/secrets.md")
	if err != nil {
		t.Fatalf("idx.Get: %v", err)
	}
	if row == nil {
		t.Fatal("expected the standard to be indexed")
	}
}
