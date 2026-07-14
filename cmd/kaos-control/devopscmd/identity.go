// SPDX-License-Identifier: AGPL-3.0-or-later

package devopscmd

import (
	"fmt"
	"os"
	"os/user"

	"github.com/kaos-control/kaos-control/internal/config"
)

// authMode describes how the HTTP client should authenticate requests.
type authMode struct {
	// bearer is set when the caller provided a token (--token / KAOS_CONTROL_TOKEN).
	bearer string
	// localEmail is set when the identity is resolved via --as or Linux-user mapping.
	// The client sends this as X-Kaos-Local-User.
	localEmail string
}

// commonFlags holds flags shared across all devops subcommands.
type commonFlags struct {
	token   string
	asEmail string
	project string
	json    bool
}

// resolveIdentity determines authentication from the given flags and the project
// config, in F6 precedence order:
//  1. --token / KAOS_CONTROL_TOKEN → bearer mode
//  2. --as <email> → local-identity mode (server enforces authz)
//  3. os/user.Current().Username mapped via project config → local-identity mode
//
// Returns (authMode, exitCode) where exitCode is exitOK on success or
// exitIdentityUnresolved when no identity can be resolved.
func resolveIdentity(flags commonFlags, proj *config.Project) (authMode, int) {
	// Precedence 1: explicit token.
	token := flags.token
	if token == "" {
		token = os.Getenv("KAOS_CONTROL_TOKEN")
	}
	if token != "" {
		return authMode{bearer: token}, exitOK
	}

	// Precedence 2: --as <email> (honoured as local-identity; server decides authz).
	if flags.asEmail != "" {
		return authMode{localEmail: flags.asEmail}, exitOK
	}

	// Precedence 3: Linux username mapped via project config.
	if proj != nil {
		u, err := user.Current()
		if err == nil && u.Username != "" {
			if email, ok := proj.EmailForLinuxUser(u.Username); ok {
				return authMode{localEmail: email}, exitOK
			}
			fmt.Fprintf(os.Stderr, "identity not resolved: linux user %q has no mapping and no --token/KAOS_CONTROL_TOKEN supplied\n", u.Username)
			return authMode{}, exitIdentityUnresolved
		}
	}

	fmt.Fprintln(os.Stderr, "identity not resolved: no --token/KAOS_CONTROL_TOKEN and could not determine Linux username")
	return authMode{}, exitIdentityUnresolved
}
