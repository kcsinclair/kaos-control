// SPDX-License-Identifier: AGPL-3.0-or-later

// Package migratecmd implements the `kaos-control migrate-directives`
// subcommand: upgrading a project from the legacy single-CLAUDE.md layout
// to the AGENTS.md-primary directive set (see internal/directives.Migrate).
package migratecmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kaos-control/kaos-control/internal/directives"
)

// Run is the entrypoint for `kaos-control migrate-directives [-force] [path]`.
// If path is omitted the current working directory is used.
func Run(args []string) error {
	fs := flag.NewFlagSet("migrate-directives", flag.ContinueOnError)
	force := fs.Bool("force", false, "overwrite an existing user-edited AGENTS.md")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	targetPath := "."
	if fs.NArg() > 0 {
		targetPath = fs.Arg(0)
	}
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", targetPath, err)
	}

	res, err := directives.Migrate(absPath, directives.MigrateOptions{Force: *force})
	if err != nil {
		return err
	}

	printReport(absPath, res)

	for _, f := range res.Files {
		if f.Diff != "" {
			fmt.Fprintln(os.Stderr, "\npending diff — rerun with -force to overwrite:")
			fmt.Fprintln(os.Stderr, f.Diff)
			return fmt.Errorf("%s has pending user edits; rerun with -force to overwrite", f.Path)
		}
	}
	return nil
}

// printReport prints the same created/updated/skipped summary style as
// `kaos-control init`.
func printReport(root string, res directives.GenerateResult) {
	fmt.Printf("Migrated directives for %s\n", root)
	if len(res.Files) == 0 && len(res.Skipped) == 0 {
		fmt.Println("  nothing to do (already migrated, or no CLAUDE.md found)")
		return
	}
	for _, f := range res.Files {
		switch {
		case f.Diff != "":
			fmt.Printf("  pending  %s (user-edited; needs -force)\n", f.Path)
		case f.Created:
			fmt.Printf("  created  %s\n", f.Path)
		case f.Changed:
			fmt.Printf("  updated  %s\n", f.Path)
		case f.Skipped:
			fmt.Printf("  skipped  %s (already up to date)\n", f.Path)
		}
	}
	for _, s := range res.Skipped {
		fmt.Printf("  skipped  %s (no gemini driver configured)\n", s)
	}
}
