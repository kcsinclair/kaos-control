package devops

import "regexp"

// ansiEscapeRE matches ANSI/VT100 escape sequences:
//   - CSI sequences:  ESC [ ... m / K / J / H / A-G / s / u / etc.
//   - OSC sequences:  ESC ] ... BEL or ST
//   - Simple two-byte: ESC followed by a single non-[ character
var ansiEscapeRE = regexp.MustCompile(
	"\x1b(?:" +
		`\[[0-9;:<=>?]*[ -/]*[@-~]` + // CSI
		`|\][^\x07\x1b]*(?:\x07|\x1b\\)` + // OSC
		`|[^[\]]` + // other two-byte sequences
		")",
)

// StripANSI removes ANSI escape sequences from s and returns the plain text.
func StripANSI(s string) string {
	return ansiEscapeRE.ReplaceAllString(s, "")
}
