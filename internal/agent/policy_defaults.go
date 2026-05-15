// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

// DefaultBashDenylist is the built-in set of glob patterns that are always
// denied for Bash tool calls, regardless of the per-agent bash_denylist (FR12).
// It is merged with the per-agent denylist at run start; duplicates are harmless.
var DefaultBashDenylist = []string{
	"rm -rf /",
	"rm -rf /*",
	"sudo *",
	"curl *|*sh",
	"wget *|*sh",
	"curl *| *sh",
	"wget *| *sh",
	"chmod 777 /*",
	"chown * /*",
}
