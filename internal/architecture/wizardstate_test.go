// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
)

func TestWizardState_SaveLoadRoundTrips(t *testing.T) {
	runtimeDir := t.TempDir()
	st := architecture.WizardState{
		Path:               "guided",
		Answers:            []architecture.Answer{{QuestionID: "offline", Value: "no"}},
		ChosenArchitecture: "lifecycle/architecture/architectures/modular-monolith.md",
		ChosenTechStack:    "lifecycle/architecture/tech-stacks/go-vue.md",
		Step:               "stack-choice",
		UpdatedUnix:        1700000000,
	}

	if err := architecture.SaveWizardState(runtimeDir, "user@example.com", st); err != nil {
		t.Fatalf("SaveWizardState: %v", err)
	}

	got, found, err := architecture.LoadWizardState(runtimeDir, "user@example.com")
	if err != nil {
		t.Fatalf("LoadWizardState: %v", err)
	}
	if !found {
		t.Fatal("LoadWizardState: found = false, want true")
	}
	if !reflect.DeepEqual(got, st) {
		t.Errorf("LoadWizardState round-trip mismatch:\ngot:  %+v\nwant: %+v", got, st)
	}
}

func TestWizardState_LoadMissing_ReturnsNotFound(t *testing.T) {
	runtimeDir := t.TempDir()
	_, found, err := architecture.LoadWizardState(runtimeDir, "nobody@example.com")
	if err != nil {
		t.Fatalf("LoadWizardState: %v", err)
	}
	if found {
		t.Error("found = true, want false for a user with no saved state")
	}
}

func TestWizardState_Clear_RemovesState(t *testing.T) {
	runtimeDir := t.TempDir()
	st := architecture.WizardState{Path: "browse", Step: "browse-catalog"}
	if err := architecture.SaveWizardState(runtimeDir, "user@example.com", st); err != nil {
		t.Fatalf("SaveWizardState: %v", err)
	}

	if err := architecture.ClearWizardState(runtimeDir, "user@example.com"); err != nil {
		t.Fatalf("ClearWizardState: %v", err)
	}

	_, found, err := architecture.LoadWizardState(runtimeDir, "user@example.com")
	if err != nil {
		t.Fatalf("LoadWizardState after clear: %v", err)
	}
	if found {
		t.Error("found = true after ClearWizardState, want false")
	}

	// Clearing an already-absent state must not error.
	if err := architecture.ClearWizardState(runtimeDir, "user@example.com"); err != nil {
		t.Errorf("ClearWizardState on already-cleared state: %v", err)
	}
}

// TestWizardState_NeverWritesUnderLifecycleArchitecture guards NFR-1: saving
// mid-flow state must land entirely outside lifecycle/architecture/, even
// when the project root and the runtime dir happen to share a parent.
func TestWizardState_NeverWritesUnderLifecycleArchitecture(t *testing.T) {
	base := t.TempDir()
	projectRoot := filepath.Join(base, "project")
	runtimeDir := filepath.Join(base, "runtime", "myproject")
	if err := os.MkdirAll(filepath.Join(projectRoot, "lifecycle", "architecture"), 0o755); err != nil {
		t.Fatal(err)
	}

	st := architecture.WizardState{Path: "guided", Step: "questions", Answers: []architecture.Answer{{QuestionID: "offline", Value: "yes"}}}
	if err := architecture.SaveWizardState(runtimeDir, "user@example.com", st); err != nil {
		t.Fatalf("SaveWizardState: %v", err)
	}

	var lifecycleFiles []string
	err := filepath.Walk(filepath.Join(projectRoot, "lifecycle"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			lifecycleFiles = append(lifecycleFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking lifecycle tree: %v", err)
	}
	if len(lifecycleFiles) != 0 {
		t.Errorf("expected zero files under lifecycle/, got %v", lifecycleFiles)
	}

	statePath := filepath.Join(runtimeDir, "wizard-state", "user@example.com.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected state file at %q: %v", statePath, err)
	}
	if strings.Contains(statePath, filepath.Join("lifecycle", "architecture")) {
		t.Errorf("state path %q falls under lifecycle/architecture/", statePath)
	}
}
