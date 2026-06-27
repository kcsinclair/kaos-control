// SPDX-License-Identifier: AGPL-3.0-or-later

package devopscmd

import (
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
		typeArg    = fs.String("type", "", "filter by artifact type")
		statusArg  = fs.String("status", "", "filter by status")
		lineageArg = fs.String("lineage", "", "filter by lineage slug")
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

	path := "/api/p/" + entry.Name + "/artifacts"
	sep := "?"
	if *typeArg != "" {
		path += sep + "type=" + *typeArg
		sep = "&"
	}
	if *statusArg != "" {
		path += sep + "status=" + *statusArg
		sep = "&"
	}
	if *lineageArg != "" {
		path += sep + "lineage=" + *lineageArg
	}

	body, code := c.get(path)
	if code != exitOK {
		return code
	}

	if *jsonOut {
		// Extract the "artifacts" array and emit raw.
		fmt.Println(extractJSONField(body, "artifacts"))
		return exitOK
	}

	// Human-readable table.
	artifacts := parseArtifactList(body)
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TYPE\tSTATUS\tLINEAGE\tTITLE")
	for _, a := range artifacts {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", a.Type, a.Status, a.Lineage, a.Title)
	}
	tw.Flush()
	return exitOK
}
