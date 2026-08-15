// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

// TestOpen_RetrofitsArchitectureScaffold verifies that Open reconciles
// lifecycle/architecture/ for a pre-existing project that predates it,
// seeding the shipped catalog and the empty, tracked decisions/ and
// standards/ directories without requiring a fresh `init`.
func TestOpen_RetrofitsArchitectureScaffold(t *testing.T) {
	root := t.TempDir()
	dbDir := t.TempDir()

	entry := &config.ProjectEntry{Name: "test-project", Path: root}

	p, err := Open(entry, dbDir, OpenOptions{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer p.Close()

	if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture/README.md")); err != nil {
		t.Errorf("expected lifecycle/architecture/README.md to be retrofitted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture/architectures/local-web.md")); err != nil {
		t.Errorf("expected a seeded architecture catalog entry: %v", err)
	}
	for _, empty := range []string{"decisions", "standards"} {
		if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture", empty, ".gitkeep")); err != nil {
			t.Errorf("expected tracked empty %s/: %v", empty, err)
		}
	}
}
