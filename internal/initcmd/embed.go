// SPDX-License-Identifier: AGPL-3.0-or-later

package initcmd

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

// renderTemplate reads the named template from the embedded FS, executes it
// with data, and returns the rendered bytes.
func renderTemplate(name string, data TemplateData) ([]byte, error) {
	raw, err := templateFS.ReadFile("templates/" + name)
	if err != nil {
		return nil, fmt.Errorf("reading embedded template %q: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parsing template %q: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template %q: %w", name, err)
	}

	return buf.Bytes(), nil
}
