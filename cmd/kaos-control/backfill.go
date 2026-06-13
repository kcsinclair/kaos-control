// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
)

const backfillUsage = `Usage: kaos-control backfill agent-run-metrics --project <id> [flags]

Walks every agent_runs row that is missing metrics OR missing a model (and has
a terminal status), reads the per-run log file, parses the type:result line, and
writes the extracted model + cost/token metrics. Safe to re-run.

Flags:
  --project <id>    Project name (required)
  --config <path>   Path to app config.yaml (default: ~/.kaos-control/config.yaml)
  --dry-run         Parse and count but do not write to the database
`

// runBackfill is the entry point for `kaos-control backfill <subcommand>`.
func runBackfill(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("backfill requires a subcommand; try: backfill agent-run-metrics --help")
	}
	switch args[0] {
	case "agent-run-metrics":
		return runBackfillAgentRunMetrics(args[1:])
	default:
		return fmt.Errorf("unknown backfill subcommand %q", args[0])
	}
}

func runBackfillAgentRunMetrics(args []string) error {
	fs := flag.NewFlagSet("backfill agent-run-metrics", flag.ContinueOnError)
	projectName := fs.String("project", "", "project name (required)")
	cfgPath := fs.String("config", defaultConfigPath(), "path to app config.yaml")
	dryRun := fs.Bool("dry-run", false, "parse and count but do not write")
	fs.Usage = func() { fmt.Fprint(os.Stderr, backfillUsage) }
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *projectName == "" {
		fs.Usage()
		return fmt.Errorf("--project is required")
	}

	appCfg, err := config.LoadApp(*cfgPath)
	if err != nil {
		return fmt.Errorf("loading app config: %w", err)
	}

	entries, err := config.LoadProjectRegistry(appCfg.ProjectsDir)
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

	dbPath := filepath.Join(appCfg.DataDir, entry.Name, "index.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("opening db at %s: %w", dbPath, err)
	}
	defer db.Close()

	// Log files live next to the index: <dataDir>/<project>/runs/<run_id>.log
	runsLogDir := filepath.Join(appCfg.DataDir, entry.Name, "runs")

	// Process runs that are missing metrics, OR that have metrics but no model
	// (rows backfilled before model extraction was added — re-parsing the log to
	// fill in the model column is cheap and idempotent).
	rows, err := db.Query(
		`SELECT run_id FROM agent_runs
		 WHERE (metrics_available=0 OR model IS NULL OR model='')
		   AND status IN ('done','failed','killed','killed-timeout')`,
	)
	if err != nil {
		return fmt.Errorf("querying runs: %w", err)
	}
	defer rows.Close()

	var runIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scanning run_id: %w", err)
		}
		runIDs = append(runIDs, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating rows: %w", err)
	}

	var backfilled, skipped, errs int
	for _, runID := range runIDs {
		logPath := filepath.Join(runsLogDir, runID+".log")
		data, readErr := os.ReadFile(logPath)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				fmt.Printf("  skip     %s (no log file)\n", runID)
				skipped++
				continue
			}
			fmt.Fprintf(os.Stderr, "  error    %s: read log: %v\n", runID, readErr)
			errs++
			continue
		}

		parsed, parseErr := agent.ParseResultLine(string(data))
		if parseErr != nil {
			fmt.Printf("  skip     %s (no result line)\n", runID)
			skipped++
			continue
		}

		if *dryRun {
			fmt.Printf("  would backfill %s (model=%s cost=%.6f)\n", runID, parsed.Model, parsed.TotalCostUSD)
			backfilled++
			continue
		}

		// COALESCE(NULLIF(?, ''), model) leaves any existing model untouched
		// when the log carries no modelUsage (parsed.Model == "").
		_, writeErr := db.Exec(
			`UPDATE agent_runs
			 SET total_cost_usd=?, duration_api_ms=?, num_turns=?,
			     input_tokens=?, cache_creation_tokens=?, cache_read_tokens=?,
			     output_tokens=?, model=COALESCE(NULLIF(?, ''), model),
			     metrics_available=1
			 WHERE run_id=?`,
			parsed.TotalCostUSD, parsed.DurationApiMs, parsed.NumTurns,
			parsed.Usage.InputTokens, parsed.Usage.CacheCreationInputTokens,
			parsed.Usage.CacheReadInputTokens, parsed.Usage.OutputTokens,
			parsed.Model,
			runID,
		)
		if writeErr != nil {
			fmt.Fprintf(os.Stderr, "  error    %s: write metrics: %v\n", runID, writeErr)
			errs++
			continue
		}
		fmt.Printf("  backfilled %s\n", runID)
		backfilled++
	}

	verb := "backfilled"
	if *dryRun {
		verb = "would backfill"
	}
	fmt.Printf("\nScanned %d runs: %s %d / skipped %d / errors %d\n",
		len(runIDs), verb, backfilled, skipped, errs)
	if errs > 0 {
		return fmt.Errorf("%d run(s) failed", errs)
	}
	return nil
}
