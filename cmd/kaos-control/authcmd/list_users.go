// SPDX-License-Identifier: AGPL-3.0-or-later

package authcmd

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kaos-control/kaos-control/internal/auth"
)

func runListUsers(store *auth.Store, args []string) int {
	fs := flag.NewFlagSet("list-users", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	users, err := store.ListUsers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing users: %v\n", err)
		return 1
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "EMAIL\tDISPLAY NAME\tADMIN\tCREATED AT")
	for _, u := range users {
		admin := "no"
		if u.Admin {
			admin = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			u.Email, u.DisplayName, admin, u.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	_ = w.Flush()
	return 0
}
