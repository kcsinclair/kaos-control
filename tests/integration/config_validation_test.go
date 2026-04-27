//go:build integration

package integration

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/config"
)

// TestAgentActiveStatusIsKnown loads the real lifecycle/config.yaml and asserts
// that every agent whose active_status is non-empty uses a status value that
// exists in artifact.KnownStatuses.  This prevents configuration drift where
// the YAML is updated with a new status name that the artifact parser does not
// recognise.
//
// Covers test plan Milestone 5.
func TestAgentActiveStatusIsKnown(t *testing.T) {
	// Locate the repository root relative to this test file's location so the
	// test works regardless of the working directory when invoked.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile: .../kaos-control/tests/integration/config_validation_test.go
	// repoRoot: .../kaos-control/
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	cfg, err := config.LoadProject(repoRoot)
	if err != nil {
		t.Fatalf("loading lifecycle/config.yaml: %v", err)
	}

	if len(cfg.Agents) == 0 {
		t.Skip("no agents configured in lifecycle/config.yaml — nothing to validate")
	}

	for _, ag := range cfg.Agents {
		if ag.ActiveStatus == "" {
			continue
		}
		if !artifact.KnownStatuses[ag.ActiveStatus] {
			t.Errorf("agent %q has active_status %q which is not in artifact.KnownStatuses",
				ag.Name, ag.ActiveStatus)
		}
	}
}
