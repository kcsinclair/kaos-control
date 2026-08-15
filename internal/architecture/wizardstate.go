// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// WizardState is the in-progress guided/browse wizard session for one user
// in one project (OQ-3): the path taken, the answers so far, any
// provisional selection, and the current step. It is scratch state — never
// written under lifecycle/architecture/ (NFR-1) and never indexed.
type WizardState struct {
	Path               string // "browse" | "guided"
	Answers            []Answer
	ChosenArchitecture string
	ChosenTechStack    string
	Step               string
	UpdatedUnix        int64
}

// wizardStateUserIDRe restricts the characters SaveWizardState/LoadWizardState
// will use verbatim in a filename; anything else is replaced with "_" as a
// defence-in-depth measure against a crafted userID escaping the state dir.
var wizardStateUserIDRe = regexp.MustCompile(`[^A-Za-z0-9._@-]`)

// SaveWizardState persists st for userID under
// projectRuntimeDir/wizard-state/<userID>.json — outside lifecycle/, so no
// write happens under lifecycle/architecture/ before confirm (NFR-1).
func SaveWizardState(projectRuntimeDir, userID string, st WizardState) error {
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshalling wizard state: %w", err)
	}
	return writeAtomic(wizardStatePath(projectRuntimeDir, userID), data)
}

// LoadWizardState reads back the state saved by SaveWizardState for userID.
// Returns (zero value, false, nil) when no state has been saved.
func LoadWizardState(projectRuntimeDir, userID string) (WizardState, bool, error) {
	data, err := os.ReadFile(wizardStatePath(projectRuntimeDir, userID))
	if err != nil {
		if os.IsNotExist(err) {
			return WizardState{}, false, nil
		}
		return WizardState{}, false, err
	}
	var st WizardState
	if err := json.Unmarshal(data, &st); err != nil {
		return WizardState{}, false, fmt.Errorf("parsing wizard state: %w", err)
	}
	return st, true, nil
}

// ClearWizardState removes any saved state for userID. Clearing an
// already-absent state is a no-op, not an error.
func ClearWizardState(projectRuntimeDir, userID string) error {
	err := os.Remove(wizardStatePath(projectRuntimeDir, userID))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func wizardStatePath(projectRuntimeDir, userID string) string {
	safeID := wizardStateUserIDRe.ReplaceAllString(userID, "_")
	return filepath.Join(projectRuntimeDir, "wizard-state", safeID+".json")
}
