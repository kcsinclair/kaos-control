// SPDX-License-Identifier: AGPL-3.0-or-later

package devops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

func wailsProfile(t *testing.T) architecture.StackProfile {
	t.Helper()
	raw, err := catalogfs.FS.ReadFile("tech-stacks/wails.md")
	if err != nil {
		t.Fatal(err)
	}
	p, err := architecture.ParseStackProfile(raw)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBootstrapPipelines_Wails(t *testing.T) {
	root := t.TempDir()

	created, err := BootstrapPipelines(root, wailsProfile(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 3 {
		t.Fatalf("expected build/lint/test pipelines, got %v", created)
	}

	// The generated files must parse via the normal discovery path.
	pipelines, warnings := Discover(filepath.Join(root, "lifecycle", "devops"))
	if len(warnings) != 0 {
		t.Errorf("unexpected discovery warnings: %v", warnings)
	}
	byType := map[string]Pipeline{}
	for _, p := range pipelines {
		byType[p.Type] = p
	}

	// build: backend `go build` + frontend `pnpm build` = 2 steps.
	if got := len(byType["build"].Steps); got != 2 {
		t.Errorf("build pipeline: want 2 steps, got %d: %+v", got, byType["build"].Steps)
	}
	// test: backend-developer and test-developer both run `go test ./...`, so
	// the duplicate collapses — backend `go test` + frontend `pnpm test` = 2.
	if got := len(byType["test"].Steps); got != 2 {
		t.Errorf("test pipeline: want 2 deduped steps, got %d: %+v", got, byType["test"].Steps)
	}

	var haveGoBuild bool
	for _, s := range byType["build"].Steps {
		if s.Command == "go build ./..." {
			haveGoBuild = true
		}
	}
	if !haveGoBuild {
		t.Error("build pipeline missing the `go build ./...` step")
	}
}

func TestBootstrapPipelines_SkipIfExists(t *testing.T) {
	root := t.TempDir()
	devDir := filepath.Join(root, "lifecycle", "devops")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := []byte("name: My Build\ntype: build\nsteps:\n  - {name: custom, command: make}\n")
	if err := os.WriteFile(filepath.Join(devDir, "build.yaml"), custom, 0o644); err != nil {
		t.Fatal(err)
	}

	created, err := BootstrapPipelines(root, wailsProfile(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range created {
		if c == "lifecycle/devops/build.yaml" {
			t.Error("build.yaml was regenerated despite already existing")
		}
	}
	got, _ := os.ReadFile(filepath.Join(devDir, "build.yaml"))
	if string(got) != string(custom) {
		t.Error("an existing build.yaml was overwritten")
	}
}
