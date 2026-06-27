// SPDX-License-Identifier: AGPL-3.0-or-later

package devopscmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
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

	// Compose status from three endpoints (mirrors the HTTP status view).
	distBody, code := c.get(base + "/dashboard/status-distribution")
	if code != exitOK {
		return code
	}
	runsBody, code := c.get(base + "/agents/runs?limit=5")
	if code != exitOK {
		return code
	}
	locksBody, code := c.get(base + "/locks")
	if code != exitOK {
		return code
	}

	if *jsonOut {
		// Emit one combined object.
		combined := map[string]json.RawMessage{
			"status_distribution": json.RawMessage(distBody),
			"agent_runs":          json.RawMessage(runsBody),
			"locks":               json.RawMessage(locksBody),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(combined)
		return exitOK
	}

	// Human-readable summary.
	fmt.Printf("Project: %s\n\n", entry.Name)
	fmt.Println("=== Status Distribution ===")
	fmt.Println(distBody)
	fmt.Println("=== Recent Agent Runs ===")
	fmt.Println(runsBody)
	fmt.Println("=== Locks ===")
	fmt.Println(locksBody)
	return exitOK
}
