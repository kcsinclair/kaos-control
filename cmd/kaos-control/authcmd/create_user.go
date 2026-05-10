// SPDX-License-Identifier: AGPL-3.0-or-later

package authcmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kaos-control/kaos-control/internal/auth"
	"golang.org/x/term"
)

func runCreateUser(store *auth.Store, args []string) int {
	fs := flag.NewFlagSet("create-user", flag.ContinueOnError)
	var (
		email         string
		name          string
		admin         bool
		passwordStdin bool
	)
	fs.StringVar(&email, "email", "", "user email address (required)")
	fs.StringVar(&name, "name", "", "display name (defaults to email)")
	fs.BoolVar(&admin, "admin", false, "grant admin flag")
	fs.BoolVar(&passwordStdin, "password-stdin", false, "read password from stdin instead of prompting")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if email == "" {
		fmt.Fprintln(os.Stderr, "error: --email is required")
		fs.Usage()
		return 1
	}
	if name == "" {
		name = email
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

	if err := store.CreateUser(email, name, password, admin); err != nil {
		fmt.Fprintf(os.Stderr, "error creating user: %v\n", err)
		return 1
	}
	fmt.Printf("User %q created successfully.\n", email)
	return 0
}

// readPassword reads a password from stdin (stripped of trailing newline) or
// prompts interactively using the terminal when passwordStdin is false.
func readPassword(fromStdin bool) (string, error) {
	if fromStdin {
		buf := make([]byte, 4096)
		n, err := os.Stdin.Read(buf)
		if err != nil && n == 0 {
			return "", err
		}
		pw := strings.TrimRight(string(buf[:n]), "\r\n")
		return pw, nil
	}

	fmt.Fprint(os.Stderr, "Password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after hidden input
	if err != nil {
		return "", err
	}
	return string(pw), nil
}
