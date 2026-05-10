// SPDX-License-Identifier: AGPL-3.0-or-later

package authcmd

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
)

func runCreateToken(store *auth.Store, args []string) int {
	fs := flag.NewFlagSet("create-token", flag.ContinueOnError)
	var (
		email   string
		expires string
	)
	fs.StringVar(&email, "email", "", "user email to issue the token for (required)")
	fs.StringVar(&expires, "expires", "", "token lifetime as a Go duration, e.g. 720h (omit for no expiry)")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if email == "" {
		fmt.Fprintln(os.Stderr, "error: --email is required")
		fs.Usage()
		return 1
	}

	var expiresAt *time.Time
	if expires != "" {
		d, err := time.ParseDuration(expires)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing --expires %q: %v\n", expires, err)
			return 1
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	token, err := store.CreateToken(email, expiresAt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating token: %v\n", err)
		return 1
	}

	fmt.Println(token)
	fmt.Fprintln(os.Stderr, "WARNING: this token will not be shown again. Store it securely.")
	return 0
}
