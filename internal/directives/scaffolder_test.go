// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffolder_Available_OffersAgentDirectivesStep(t *testing.T) {
	steps, ok := Scaffolder{}.Available(t.TempDir(), "modular-monolith", "go-vue")
	if !ok {
		t.Fatal("expected agent-directives scaffolding to be available")
	}
	if len(steps) != 1 || steps[0].Key != scaffoldStepKey {
		t.Fatalf("expected a single %q step, got %+v", scaffoldStepKey, steps)
	}
	if steps[0].Title == "" || steps[0].Description == "" {
		t.Error("scaffold step is missing a title/description")
	}
}

func TestScaffolder_Run_GeneratesDirectiveFiles(t *testing.T) {
	root := promotedGoVueFixture(t)

	res, err := Scaffolder{}.Run(root, "modular-monolith", "go-vue", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	applied := strings.Join(res.Applied, ",")
	for _, want := range []string{"AGENTS.md", "CLAUDE.md"} {
		if !strings.Contains(applied, want) {
			t.Errorf("expected %s in applied files, got %v", want, res.Applied)
		}
		if _, err := os.Stat(filepath.Join(root, want)); err != nil {
			t.Errorf("expected %s written to disk: %v", want, err)
		}
	}
}
