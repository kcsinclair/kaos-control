// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/AGENTS.md.tmpl
var templatesFS embed.FS

// genStart and genEnd delimit the machine-managed region of a generated
// directive file. A refresh replaces only the content between them,
// preserving any user prose written above or below (OQ-6).
const (
	genStart = "<!-- kaos-control:generated:start -->"
	genEnd   = "<!-- kaos-control:generated:end -->"
)

var agentsTmpl = template.Must(template.New("AGENTS.md.tmpl").Funcs(template.FuncMap{
	"join": func(items []string) string { return strings.Join(items, ", ") },
}).ParseFS(templatesFS, "templates/AGENTS.md.tmpl"))

// RenderAgents executes the AGENTS.md template against m and wraps the
// result in the managed-region markers. The output is deterministic: the
// same model always renders byte-identical output (NFR-2).
func RenderAgents(m DirectiveModel) ([]byte, error) {
	var body bytes.Buffer
	if err := agentsTmpl.Execute(&body, m); err != nil {
		return nil, fmt.Errorf("rendering AGENTS.md template: %w", err)
	}

	var out bytes.Buffer
	out.WriteString(genStart)
	out.WriteString("\n")
	out.Write(bytes.TrimRight(body.Bytes(), "\n"))
	out.WriteString("\n")
	out.WriteString(genEnd)
	out.WriteString("\n")
	return out.Bytes(), nil
}

// RenderPointer returns the body of a pointer file (CLAUDE.md, GEMINI.md)
// that imports the canonical AGENTS.md via the `@AGENTS.md` convention
// (OQ-7: gemini-cli and Claude Code both follow it). It is a literal
// one-liner and carries no managed region — pointer files never drift.
func RenderPointer(canonical string) []byte {
	return []byte("@" + canonical + "\n")
}
