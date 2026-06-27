// SPDX-License-Identifier: AGPL-3.0-or-later

package devopscmd

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func runStatus(args []string) int {
	fs := flag.NewFlagSet("devops status", flag.ContinueOnError)
	var (
		token      = fs.String("token", "", "bearer API token (overrides KAOS_CONTROL_TOKEN)")
		asEmail    = fs.String("as", "", "assert identity as this email")
		projectArg = fs.String("project", "", "project name (default: infer from cwd)")
		jsonOut    = fs.Bool("json", false, "emit JSON output")
	)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return exitOK
		}
		return exitOpFailed
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: kaos-control devops status <job-name> [--json]")
		return exitOpFailed
	}
	slug := fs.Arg(0)

	flags := commonFlags{
		token:   *token,
		asEmail: *asEmail,
		project: *projectArg,
		json:    *jsonOut,
	}

	appCfg, code := loadAppConfig()
	if code != exitOK {
		return code
	}

	entry, proj, code := selectProject(flags, appCfg)
	if code != exitOK {
		return code
	}

	identity, code := resolveIdentity(flags, proj)
	if code != exitOK {
		return code
	}

	c := newClient(appCfg, identity)
	base := "/api/p/" + entry.Name

	// Fetch the most recent run for this pipeline.
	runsBody, code := c.get(base + "/devops/pipelines/" + slug + "/runs?limit=1")
	if code != exitOK {
		return code
	}

	var runsWrapper struct {
		Runs []struct {
			RunID      string `json:"run_id"`
			Status     string `json:"status"`
			StartedAt  string `json:"started_at"`
			EndedAt    string `json:"ended_at"`
			DurationMs int64  `json:"duration_ms"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(runsBody), &runsWrapper); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing runs response: %v\n", err)
		return exitOpFailed
	}

	if len(runsWrapper.Runs) == 0 {
		if *jsonOut {
			fmt.Printf(`{"pipeline":%q,"runs":[]}`+"\n", slug)
		} else {
			fmt.Printf("pipeline: %s\nno runs found\n", slug)
		}
		return exitOK
	}

	run := runsWrapper.Runs[0]

	// Fetch the run log for step-level output summary; non-fatal if unavailable.
	logBody, logCode := c.get(base + "/devops/pipelines/" + slug + "/runs/" + run.RunID + "/log")
	steps := make([]stepSummary, 0)
	if logCode == exitOK {
		steps = parseStepSummaries(logBody)
	}

	if *jsonOut {
		out := map[string]any{
			"pipeline": slug,
			"run": map[string]any{
				"run_id":      run.RunID,
				"status":      run.Status,
				"started_at":  run.StartedAt,
				"ended_at":    run.EndedAt,
				"duration_ms": run.DurationMs,
			},
			"steps": steps,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return exitOK
	}

	fmt.Printf("Pipeline: %s\n", slug)
	fmt.Printf("Run ID:   %s\n", run.RunID)
	fmt.Printf("Status:   %s\n", run.Status)
	fmt.Printf("Started:  %s\n", run.StartedAt)
	fmt.Printf("Ended:    %s\n", run.EndedAt)
	fmt.Printf("Duration: %s\n", formatDuration(run.DurationMs))

	if len(steps) > 0 {
		fmt.Println()
		fmt.Println("Steps:")
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  NAME\tSTATUS\tDURATION\tEXIT CODE")
		for _, s := range steps {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%d\n", s.Name, s.Status, formatDuration(int64(s.DurationMs)), s.ExitCode)
		}
		tw.Flush()
	}
	return exitOK
}

type stepSummary struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	ExitCode   int     `json:"exit_code"`
	DurationMs float64 `json:"duration_ms"`
}

// parseStepSummaries extracts step completion events from an NDJSON run log.
func parseStepSummaries(logBody string) []stepSummary {
	steps := make([]stepSummary, 0)
	scanner := bufio.NewScanner(strings.NewReader(logBody))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var event struct {
			Type            string  `json:"type"`
			Step            string  `json:"step"`
			Status          string  `json:"status"`
			ExitCode        int     `json:"exit_code"`
			DurationSeconds float64 `json:"duration_seconds"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type == "pipeline.step.completed" {
			steps = append(steps, stepSummary{
				Name:       event.Step,
				Status:     event.Status,
				ExitCode:   event.ExitCode,
				DurationMs: event.DurationSeconds * 1000,
			})
		}
	}
	return steps
}

// formatDuration formats milliseconds into a human-readable string.
func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	s := float64(ms) / 1000
	if s < 60 {
		return fmt.Sprintf("%.1fs", s)
	}
	m := int(s / 60)
	s -= float64(m * 60)
	return fmt.Sprintf("%dm%.0fs", m, s)
}
