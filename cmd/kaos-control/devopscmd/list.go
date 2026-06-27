// SPDX-License-Identifier: AGPL-3.0-or-later

package devopscmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

func runList(args []string) int {
	fs := flag.NewFlagSet("devops list", flag.ContinueOnError)
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
	body, code := c.get("/api/p/" + entry.Name + "/devops/pipelines")
	if code != exitOK {
		return code
	}

	if *jsonOut {
		fmt.Println(extractJSONField(body, "pipelines"))
		return exitOK
	}

	var wrapper struct {
		Pipelines []struct {
			Slug      string `json:"slug"`
			Name      string `json:"name"`
			Type      string `json:"type"`
			StepCount int    `json:"step_count"`
		} `json:"pipelines"`
	}
	if err := json.Unmarshal([]byte(body), &wrapper); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing response: %v\n", err)
		return exitOpFailed
	}

	if len(wrapper.Pipelines) == 0 {
		fmt.Println("no pipelines defined under lifecycle/devops/")
		return exitOK
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SLUG\tNAME\tTYPE\tSTEPS")
	for _, p := range wrapper.Pipelines {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\n", p.Slug, p.Name, p.Type, p.StepCount)
	}
	tw.Flush()
	return exitOK
}
