// SPDX-License-Identifier: AGPL-3.0-or-later

package authcmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/kaos-control/kaos-control/internal/auth"
)

func runDeleteUser(store *auth.Store, args []string) int {
	fs := flag.NewFlagSet("delete-user", flag.ContinueOnError)
	var email string
	fs.StringVar(&email, "email", "", "email of the user to delete (required)")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if email == "" {
		fmt.Fprintln(os.Stderr, "error: --email is required")
		fs.Usage()
		return 1
	}

	if err := store.DeleteUser(email); err != nil {
		fmt.Fprintf(os.Stderr, "error deleting user: %v\n", err)
		return 1
	}
	fmt.Printf("User %q deleted (sessions and tokens revoked).\n", email)
	return 0
}
