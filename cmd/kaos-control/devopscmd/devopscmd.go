// SPDX-License-Identifier: AGPL-3.0-or-later

// Package devopscmd implements the `kaos-control devops` subcommand group.
// It is an HTTP client that talks to the running server and resolves the
// caller's identity from flags, env vars, or the OS username mapped via
// project config.
package devopscmd

import (
	"fmt"
	"os"
)

// Exit codes (NF5).
const (
	exitOK                 = 0
	exitOpFailed           = 1
	exitIdentityUnresolved = 3
	exitForbidden          = 4
)

const devopsUsage = `Usage: kaos-control devops <subcommand> [flags]

Subcommands:
  list     List artifacts in the project index
  status   Show project health (status counts, active runs, locks)
  run      Trigger a devops pipeline task

Identity flags (applied to all subcommands):
  --token <token>     Bearer API token (overrides KAOS_CONTROL_TOKEN env var)
  --as <email>        Assert identity as the given email (requires loopback + privilege)
  --project <name>    Select project by name (default: inferred from cwd)
  --json              Emit machine-readable JSON on stdout

Exit codes:
  0  Success
  1  Operation failed (server error, unknown task, etc.)
  3  Identity unresolved (unmapped Linux user, no token)
  4  Forbidden (insufficient role)

Run 'kaos-control devops <subcommand> --help' for subcommand-specific flags.
`

// Run is the entrypoint for the `kaos-control devops` subcommand.
// args is os.Args[2:] (everything after "devops").
func Run(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-help" || args[0] == "-h" {
		fmt.Print(devopsUsage)
		return exitOK
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list":
		return runList(rest)
	case "status":
		return runStatus(rest)
	case "run":
		return runRun(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown devops subcommand %q\n\n%s", sub, devopsUsage)
		return exitOpFailed
	}
}
