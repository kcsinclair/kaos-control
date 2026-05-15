// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// hookSettings is the structure written to the Claude Code settings.json that
// wires the PreToolUse hook to the kaos-control hook-helper binary (FR6).
type hookSettings struct {
	Hooks map[string][]hookEntry `json:"hooks"`
}

type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// WriteHookSettings writes a Claude Code settings.json to dir that configures
// a PreToolUse hook pointing at the kaos-control hook-helper binary.
//
// The returned path is the absolute path to the file. The cleanup function
// removes it and is safe to call more than once (NFR4).
func WriteHookSettings(dir, binary, serverAddr, runID string) (path string, cleanup func(), err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", nil, fmt.Errorf("creating hook settings dir: %w", err)
	}

	cmd := fmt.Sprintf("%s hook-helper --server %s --run-id %s", binary, serverAddr, runID)
	settings := hookSettings{
		Hooks: map[string][]hookEntry{
			"PreToolUse": {
				{Type: "command", Command: cmd},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("marshalling hook settings: %w", err)
	}

	// Use the run ID in the filename to avoid collisions across concurrent runs.
	path = filepath.Join(dir, "hook-settings-"+runID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", nil, fmt.Errorf("writing hook settings: %w", err)
	}

	var once sync.Once
	cleanup = func() {
		once.Do(func() { _ = os.Remove(path) })
	}
	return path, cleanup, nil
}
