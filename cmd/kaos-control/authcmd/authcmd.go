// SPDX-License-Identifier: AGPL-3.0-or-later

// Package authcmd implements the `kaos-control auth` subcommand family.
// All commands open the auth DB directly and do not require the HTTP server.
package authcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
)

const authUsage = `Usage: kaos-control auth <subcommand> [flags]

Subcommands:
  create-user     Register a new user account
  list-users      List all registered users
  delete-user     Delete a user and their sessions/tokens
  reset-password  Reset a user's password
  create-token    Create a bearer API token for a user

Run 'kaos-control auth <subcommand> --help' for subcommand-specific flags.
`

// Run is the entrypoint for the `kaos-control auth` subcommand.
// args is os.Args[2:] (everything after "auth").
func Run(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-help" || args[0] == "-h" {
		fmt.Print(authUsage)
		return 0
	}

	sub := args[0]
	rest := args[1:]

	store, cleanup, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening auth database: %v\n", err)
		return 1
	}
	defer cleanup()

	switch sub {
	case "create-user":
		return runCreateUser(store, rest)
	case "list-users":
		return runListUsers(store, rest)
	case "delete-user":
		return runDeleteUser(store, rest)
	case "reset-password":
		return runResetPassword(store, rest)
	case "create-token":
		return runCreateToken(store, rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown auth subcommand %q\n\n%s", sub, authUsage)
		return 1
	}
}

// openStore resolves the config path, loads app config, and opens the auth DB.
func openStore() (*auth.Store, func(), error) {
	cfgPath := defaultConfigPath()

	appCfg, err := config.LoadApp(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	dbPath := filepath.Join(appCfg.DataDir, "auth.db")
	store, err := auth.Open(dbPath, appCfg.Auth.SessionTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("opening auth db at %s: %w", dbPath, err)
	}

	cleanup := func() { _ = store.Close() }
	return store, cleanup, nil
}

func defaultConfigPath() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "kaos-control", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kaos-control", "config.yaml")
}
