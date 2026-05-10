// SPDX-License-Identifier: AGPL-3.0-or-later

package initcmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// TemplateData is passed to all seed-file templates.
type TemplateData struct {
	ProjectName string
	Language    string
}

// ForceFlags controls which existing seed files may be overwritten.
type ForceFlags struct {
	Config    bool
	ClaudeMd  bool
	Settings  bool
	Gitignore bool
}

// Result records the outcome of creating or skipping one file or directory.
type Result struct {
	Path    string // relative path from the project root
	Created bool   // true = created/written, false = skipped
}

// Run is the entrypoint for the `kaos-control init` subcommand.
func Run(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)

	var (
		force          bool
		forceConfig    bool
		forceClaudeMd  bool
		forceSettings  bool
		forceGitignore bool
		projectName    string
		language       string
	)

	fs.BoolVar(&force, "force", false, "overwrite all existing seed files")
	fs.BoolVar(&forceConfig, "force-config", false, "overwrite lifecycle/config.yaml if it exists")
	fs.BoolVar(&forceClaudeMd, "force-claude-md", false, "overwrite CLAUDE.md if it exists")
	fs.BoolVar(&forceSettings, "force-settings", false, "overwrite .claude/settings.json if it exists")
	fs.BoolVar(&forceGitignore, "force-gitignore", false, "overwrite .gitignore if it exists")
	fs.StringVar(&projectName, "project-name", "", "project name interpolated into CLAUDE.md (defaults to directory name)")
	fs.StringVar(&language, "language", "", "primary language hint for CLAUDE.md")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Positional argument: path (defaults to current directory).
	targetPath := "."
	if fs.NArg() > 0 {
		targetPath = fs.Arg(0)
	}

	// Resolve to an absolute path and create it if absent.
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", targetPath, err)
	}
	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", absPath, err)
	}

	// Default project name to the directory basename.
	if projectName == "" {
		projectName = filepath.Base(absPath)
	}

	// --force implies all granular force flags.
	if force {
		forceConfig = true
		forceClaudeMd = true
		forceSettings = true
		forceGitignore = true
	}

	ff := ForceFlags{
		Config:    forceConfig,
		ClaudeMd:  forceClaudeMd,
		Settings:  forceSettings,
		Gitignore: forceGitignore,
	}

	data := TemplateData{
		ProjectName: projectName,
		Language:    language,
	}

	// Scaffold lifecycle directories.
	dirResults, err := scaffoldDirs(absPath)
	if err != nil {
		return fmt.Errorf("scaffolding directories: %w", err)
	}

	// Write seed files.
	fileResults, err := writeSeedFiles(absPath, data, ff)
	if err != nil {
		return fmt.Errorf("writing seed files: %w", err)
	}

	// Print summary (FR-7).
	fmt.Printf("Initialized kaos-control project at %s\n", absPath)
	for _, r := range dirResults {
		if r.Created {
			fmt.Printf("  created  %s\n", r.Path)
		} else {
			fmt.Printf("  skipped  %s (already exists)\n", r.Path)
		}
	}
	for _, r := range fileResults {
		if r.Created {
			fmt.Printf("  created  %s\n", r.Path)
		} else {
			fmt.Printf("  skipped  %s (already exists)\n", r.Path)
		}
	}

	return nil
}
