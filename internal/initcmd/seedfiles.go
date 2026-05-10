// SPDX-License-Identifier: AGPL-3.0-or-later

package initcmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// seedFileSpec describes one seed file: the embedded template to render, the
// target path relative to the project root, and the ForceFlags field that
// permits overwriting it.
type seedFileSpec struct {
	tmpl    string
	relPath string
	force   func(ForceFlags) bool
}

// seedFileSpecs lists every seed file emitted by init, in the order they are
// written (and reported in the summary).
var seedFileSpecs = []seedFileSpec{
	{
		tmpl:    "config.yaml.tmpl",
		relPath: "lifecycle/config.yaml",
		force:   func(f ForceFlags) bool { return f.Config },
	},
	{
		tmpl:    "CLAUDE.md.tmpl",
		relPath: "CLAUDE.md",
		force:   func(f ForceFlags) bool { return f.ClaudeMd },
	},
	{
		tmpl:    "settings.json.tmpl",
		relPath: ".claude/settings.json",
		force:   func(f ForceFlags) bool { return f.Settings },
	},
	{
		tmpl:    "gitignore.tmpl",
		relPath: ".gitignore",
		force:   func(f ForceFlags) bool { return f.Gitignore },
	},
}

// writeSeedFiles renders and writes each seed file under root. If a file
// already exists and the corresponding force flag is false, it is skipped and a
// message is printed to stderr. Parent directories are created as needed.
// File permissions are 0644.
func writeSeedFiles(root string, data TemplateData, ff ForceFlags) ([]Result, error) {
	var results []Result

	for _, sf := range seedFileSpecs {
		absPath := filepath.Join(root, filepath.FromSlash(sf.relPath))

		_, statErr := os.Stat(absPath)
		exists := statErr == nil

		if exists && !sf.force(ff) {
			fmt.Fprintf(os.Stderr, "skipped: %s (already exists; use --force to overwrite)\n", sf.relPath)
			results = append(results, Result{Path: sf.relPath, Created: false})
			continue
		}

		content, err := renderTemplate(sf.tmpl, data)
		if err != nil {
			return nil, fmt.Errorf("rendering %s: %w", sf.tmpl, err)
		}

		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return nil, fmt.Errorf("creating parent directory for %s: %w", sf.relPath, err)
		}

		if err := os.WriteFile(absPath, content, 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", sf.relPath, err)
		}

		results = append(results, Result{Path: sf.relPath, Created: true})
	}

	return results, nil
}
