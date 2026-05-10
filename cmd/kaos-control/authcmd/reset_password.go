// SPDX-License-Identifier: AGPL-3.0-or-later

package authcmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/kaos-control/kaos-control/internal/auth"
)

func runResetPassword(store *auth.Store, args []string) int {
	fs := flag.NewFlagSet("reset-password", flag.ContinueOnError)
	var (
		email         string
		passwordStdin bool
	)
	fs.StringVar(&email, "email", "", "user email address (required)")
	fs.BoolVar(&passwordStdin, "password-stdin", false, "read new password from stdin instead of prompting")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if email == "" {
		fmt.Fprintln(os.Stderr, "error: --email is required")
		fs.Usage()
		return 1
	}

	password, err := readPassword(passwordStdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading password: %v\n", err)
		return 1
	}
	if password == "" {
		fmt.Fprintln(os.Stderr, "error: password must not be empty")
		return 1
	}

	if err := store.ResetPassword(email, password); err != nil {
		fmt.Fprintf(os.Stderr, "error resetting password: %v\n", err)
		return 1
	}
	fmt.Printf("Password for %q updated successfully.\n", email)
	return 0
}
