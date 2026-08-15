// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

// promotedGoVueFixture writes a minimal project root with a promoted
// go-vue.md tech-stack and the standard-agent config fixture (which
// includes a qa agent on driver: gemini-cli).
func promotedGoVueFixture(t *testing.T) string {
	t.Helper()
	root := writeConfigFixture(t)

	raw, err := catalogfs.FS.ReadFile("tech-stacks/go-vue.md")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "lifecycle/architecture/go-vue.md")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestGenerate_PromotedStack_WritesAllFilesAndPatchesConfig(t *testing.T) {
	root := promotedGoVueFixture(t)

	res, err := Generate(root, GenerateOptions{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	names := make(map[string]FileWrite, len(res.Files))
	for _, f := range res.Files {
		names[f.Path] = f
	}
	for _, want := range []string{agentsFile, claudeFile, geminiFile} {
		fw, ok := names[want]
		if !ok {
			t.Fatalf("missing FileWrite for %s in %+v", want, res.Files)
		}
		if !fw.Created {
			t.Errorf("%s: expected Created=true, got %+v", want, fw)
		}
	}
	if len(res.Skipped) != 0 {
		t.Errorf("expected nothing skipped (gemini-cli driver present), got %v", res.Skipped)
	}

	agentsBody, err := os.ReadFile(filepath.Join(root, agentsFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(agentsBody), genStart) {
		t.Error("AGENTS.md missing managed-region start marker")
	}

	claudeBody, err := os.ReadFile(filepath.Join(root, claudeFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(claudeBody) != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md: got %q", claudeBody)
	}

	geminiBody, err := os.ReadFile(filepath.Join(root, geminiFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(geminiBody) != "@AGENTS.md\n" {
		t.Errorf("GEMINI.md: got %q", geminiBody)
	}

	cfgRaw, err := os.ReadFile(filepath.Join(root, "lifecycle/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgRaw), "go build ./...") {
		t.Error("lifecycle/config.yaml was not patched with the stack's build command")
	}
}

func TestGenerate_Idempotent(t *testing.T) {
	root := promotedGoVueFixture(t)

	if _, err := Generate(root, GenerateOptions{}); err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	res, err := Generate(root, GenerateOptions{})
	if err != nil {
		t.Fatalf("second Generate: %v", err)
	}
	for _, f := range res.Files {
		if f.Created || f.Changed {
			t.Errorf("expected no-op on second run, got %+v", f)
		}
		if !f.Skipped {
			t.Errorf("expected Skipped=true on second run for %s, got %+v", f.Path, f)
		}
	}
}

func TestGenerate_NoGeminiDriver_SkipsGeminiMd(t *testing.T) {
	root := promotedGoVueFixture(t)

	res, err := Generate(root, GenerateOptions{Drivers: []string{"claude-code-cli"}})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	for _, f := range res.Files {
		if f.Path == geminiFile {
			t.Fatalf("did not expect a FileWrite for GEMINI.md, got %+v", f)
		}
	}
	if len(res.Skipped) != 1 || res.Skipped[0] != geminiFile {
		t.Errorf("expected GEMINI.md in Skipped, got %v", res.Skipped)
	}
	if _, err := os.Stat(filepath.Join(root, geminiFile)); !os.IsNotExist(err) {
		t.Error("GEMINI.md should not have been written")
	}
}

func TestGenerate_NoPromotedStack_FallsBackToGeneric(t *testing.T) {
	root := writeConfigFixture(t) // no promoted stack under lifecycle/architecture/

	res, err := Generate(root, GenerateOptions{Drivers: []string{"claude-code-cli"}})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(res.DisabledAgents) != 0 {
		t.Errorf("expected no disabled agents in generic fallback, got %v", res.DisabledAgents)
	}

	agentsBody, err := os.ReadFile(filepath.Join(root, agentsFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agentsBody), "Once the Architecture Wizard has run") {
		t.Errorf("expected generic pre-wizard wording, got:\n%s", agentsBody)
	}

	// lifecycle/config.yaml must be untouched in the generic fallback path.
	cfgBefore := configFixture
	cfgAfter, err := os.ReadFile(filepath.Join(root, "lifecycle/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(cfgAfter) != cfgBefore {
		t.Error("lifecycle/config.yaml should not be patched when no stack is promoted")
	}
}

func TestGenericAgents(t *testing.T) {
	out, err := GenericAgents("my-project", "")
	if err != nil {
		t.Fatalf("GenericAgents: %v", err)
	}
	if !strings.Contains(string(out), "my-project") {
		t.Errorf("expected project name in output:\n%s", out)
	}
	if !strings.HasPrefix(string(out), genStart) {
		t.Error("output missing managed-region marker")
	}
}
