// SPDX-License-Identifier: AGPL-3.0-or-later

// Package releasescmd implements the `kaos-control releases` subcommand family.
package releasescmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/release"
)

const rehydrateUsage = `Usage: kaos-control releases rehydrate --project <id> [flags]

Reads lifecycle/releases/*.md files and upserts them into the project's
SQLite index. Invalid files are skipped with a warning. The result is
printed as JSON on stdout.

Flags:
  --project <id>    Project name (required)
  --config <path>   Path to app config.yaml (default: ~/.kaos-control/config.yaml)
`

// Run is the entry point for `kaos-control releases <subcommand>`.
func Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("releases requires a subcommand; try: releases rehydrate --help")
	}
	switch args[0] {
	case "rehydrate":
		return runRehydrate(args[1:])
	default:
		return fmt.Errorf("unknown releases subcommand %q", args[0])
	}
}

func runRehydrate(args []string) error {
	fs := flag.NewFlagSet("releases rehydrate", flag.ContinueOnError)
	projectName := fs.String("project", "", "project name (required)")
	cfgPath := fs.String("config", defaultConfigPath(), "path to app config.yaml")
	fs.Usage = func() { fmt.Fprint(os.Stderr, rehydrateUsage) }

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *projectName == "" {
		fs.Usage()
		return fmt.Errorf("--project is required")
	}

	// Load app config to find the data directory.
	appCfg, err := config.LoadApp(*cfgPath)
	if err != nil {
		return fmt.Errorf("loading app config: %w", err)
	}

	// Find the project entry.
	registryDir := filepath.Dir(*cfgPath)
	entries, err := config.LoadProjectRegistry(filepath.Join(registryDir, "projects"))
	if err != nil {
		return fmt.Errorf("loading project registry: %w", err)
	}
	var entry *config.ProjectEntry
	for _, e := range entries {
		if e.Name == *projectName {
			entry = e
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("project %q not found in registry", *projectName)
	}

	// Open the project's SQLite DB.
	dbPath := filepath.Join(appCfg.DataDir, entry.Name, "index.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("opening db at %s: %w", dbPath, err)
	}
	defer db.Close()

	store := release.NewStore(db)
	result, err := release.Rehydrate(context.Background(), store, entry.Name, entry.Path)
	if err != nil {
		return fmt.Errorf("rehydrate: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kaos-control", "config.yaml")
}
