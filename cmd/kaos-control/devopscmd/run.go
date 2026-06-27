// SPDX-License-Identifier: AGPL-3.0-or-later

package devopscmd

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func runRun(args []string) int {
	fs := flag.NewFlagSet("devops run", flag.ContinueOnError)
	var (
		token      = fs.String("token", "", "bearer API token (overrides KAOS_CONTROL_TOKEN)")
		asEmail    = fs.String("as", "", "assert identity as this email")
		projectArg = fs.String("project", "", "project name (default: infer from cwd)")
		jsonOut    = fs.Bool("json", false, "emit JSON output")
		follow     = fs.Bool("follow", false, "stream run log to completion")
	)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return exitOK
		}
		return exitOpFailed
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: kaos-control devops run <task> [--follow] [--json]")
		return exitOpFailed
	}
	task := fs.Arg(0)

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

	// Trigger the pipeline run.
	body, code := c.post(base+"/devops/pipelines/"+task+"/run", nil)
	if code != exitOK {
		return code
	}

	var result struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil || result.RunID == "" {
		fmt.Fprintf(os.Stderr, "unexpected response from server: %s\n", body)
		return exitOpFailed
	}

	if *jsonOut {
		fmt.Printf(`{"run_id":%q}`+"\n", result.RunID)
	} else {
		fmt.Println(result.RunID)
	}

	if !*follow {
		return exitOK
	}

	// Stream the NDJSON run log.
	logPath := base + "/devops/runs/" + result.RunID
	return streamRunLog(c, logPath, *jsonOut)
}

// streamRunLog polls/streams the NDJSON run log for the given path and exits
// with exitOK on a successful terminal status or exitOpFailed otherwise.
func streamRunLog(c *client, path string, jsonOut bool) int {
	logBody, code := c.get(path)
	if code != exitOK {
		return code
	}

	scanner := bufio.NewScanner(strings.NewReader(logBody))
	var lastStatus string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if jsonOut {
			fmt.Println(line)
		} else {
			// Pretty-print the log line content.
			var event map[string]json.RawMessage
			if err := json.Unmarshal([]byte(line), &event); err == nil {
				if msg, ok := event["message"]; ok {
					fmt.Println(strings.Trim(string(msg), `"`))
				} else {
					fmt.Println(line)
				}
			} else {
				fmt.Println(line)
			}
		}
		// Track the terminal status.
		var event struct {
			Status string `json:"status"`
		}
		if json.Unmarshal([]byte(line), &event) == nil && event.Status != "" {
			lastStatus = event.Status
		}
	}

	if lastStatus == "success" || lastStatus == "done" || lastStatus == "completed" {
		return exitOK
	}
	return exitOpFailed
}
