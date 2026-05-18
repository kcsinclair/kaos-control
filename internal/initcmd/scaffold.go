// SPDX-License-Identifier: AGPL-3.0-or-later

package initcmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// lifecycleDirs lists every directory that init must create, relative to the
// project root. Order matches FR-2 in the requirements.
var lifecycleDirs = []string{
	"lifecycle/ideas",
	"lifecycle/requirements",
	"lifecycle/backend-plans",
	"lifecycle/frontend-plans",
	"lifecycle/test-plans",
	"lifecycle/tests",
	"lifecycle/prototypes",
	"lifecycle/releases",
	"lifecycle/defects",
	"lifecycle/docs",
	"lifecycle/devops",
	"tests",
	"devops",
}

// scaffoldDirs creates all lifecycle directories under root, placing a
// .gitkeep file inside each one so that empty directories are tracked by git.
// It returns one Result per directory; Created is false when the directory (and
// its .gitkeep) already existed before the call.
func scaffoldDirs(root string) ([]Result, error) {
	var results []Result

	for _, dir := range lifecycleDirs {
		absDir := filepath.Join(root, filepath.FromSlash(dir))
		gitkeep := filepath.Join(absDir, ".gitkeep")

		// Record whether both the directory and .gitkeep already exist so we
		// can accurately report created vs skipped.
		_, dirStatErr := os.Stat(absDir)
		dirExists := dirStatErr == nil

		_, gpStatErr := os.Stat(gitkeep)
		gitkeepExists := gpStatErr == nil

		if err := os.MkdirAll(absDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating directory %q: %w", absDir, err)
		}

		if !gitkeepExists {
			if err := os.WriteFile(gitkeep, []byte{}, 0o644); err != nil {
				return nil, fmt.Errorf("writing .gitkeep in %q: %w", absDir, err)
			}
		}

		results = append(results, Result{
			Path:    filepath.Join(filepath.FromSlash(dir), ".gitkeep"),
			Created: !dirExists || !gitkeepExists,
		})
	}

	return results, nil
}
