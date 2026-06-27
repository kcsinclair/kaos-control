// SPDX-License-Identifier: AGPL-3.0-or-later

package devopscmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaos-control/kaos-control/internal/config"
)

// selectProject resolves the target project from --project or cwd inference.
// Returns the project entry, the loaded project config, and an exit code.
func selectProject(flags commonFlags, appCfg *config.App) (*config.ProjectEntry, *config.Project, int) {
	entries, err := config.LoadProjectRegistry(appCfg.ProjectsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading project registry: %v\n", err)
		return nil, nil, exitOpFailed
	}

	var entry *config.ProjectEntry
	if flags.project != "" {
		for _, e := range entries {
			if e.Name == flags.project {
				entry = e
				break
			}
		}
		if entry == nil {
			fmt.Fprintf(os.Stderr, "project %q not found in registry\n", flags.project)
			return nil, nil, exitOpFailed
		}
	} else {
		// Infer from cwd — require unambiguous match.
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not determine working directory: %v\n", err)
			return nil, nil, exitOpFailed
		}
		cwd = filepath.Clean(cwd)
		var matches []*config.ProjectEntry
		for _, e := range entries {
			root := filepath.Clean(e.Path)
			sep := string(filepath.Separator)
			if cwd == root || strings.HasPrefix(cwd+sep, root+sep) {
				matches = append(matches, e)
			}
		}
		switch len(matches) {
		case 0:
			fmt.Fprintf(os.Stderr, "no registered project contains the current directory; use --project to specify one\n")
			return nil, nil, exitOpFailed
		case 1:
			entry = matches[0]
		default:
			names := make([]string, len(matches))
			for i, m := range matches {
				names[i] = m.Name
			}
			fmt.Fprintf(os.Stderr, "ambiguous project: cwd matches %v; use --project to specify one\n", names)
			return nil, nil, exitOpFailed
		}
	}

	proj, err := config.LoadProject(entry.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading project config for %q: %v\n", entry.Name, err)
		return nil, nil, exitOpFailed
	}
	return entry, proj, exitOK
}
