// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"strings"
	"testing"
)

// The bash tool is opt-in and policy-gated. These tests pin the security
// behaviour: v1 deliberately shipped no shell tool because small local models
// are the least trustworthy with one, so every relaxation here should be a
// visible, deliberate change.

func TestBashTool_NotInDefaultToolset(t *testing.T) {
	for _, tool := range DefaultOpenAITools() {
		if tool.Function.Name == "bash" {
			t.Fatal("bash must NOT be in DefaultOpenAITools — it is opt-in per agent via bash_allowlist")
		}
	}
	if BashTool().Function.Name != "bash" {
		t.Errorf("BashTool() name = %q, want \"bash\"", BashTool().Function.Name)
	}
}

func TestBash_RefusedWhenPolicyNil(t *testing.T) {
	// Nil policy == bash not enabled for this agent. It must refuse rather than
	// fall through to executing anything.
	e := &ToolExecutor{ProjectRoot: t.TempDir()}
	out, err := e.Execute(context.Background(), "bash", `{"command":"echo pwned"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "permission denied") {
		t.Errorf("output = %q, want a permission denial", out)
	}
	if strings.Contains(out, "pwned") {
		t.Error("command executed despite nil policy")
	}
}

func TestBash_DenylistBlocksDangerousCommand(t *testing.T) {
	e := &ToolExecutor{
		ProjectRoot: t.TempDir(),
		Policy: &PolicyConfig{
			BashAllowlist: []string{"*"}, // permissive allowlist…
			BashDenylist:  mergeDenylist(nil),
		},
	}
	// …the denylist is still checked first.
	out, err := e.Execute(context.Background(), "bash", `{"command":"sudo rm -rf /"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "permission denied") {
		t.Errorf("output = %q, want denial — denylist must win over a wildcard allowlist", out)
	}
}

func TestBash_AllowlistGatesCommands(t *testing.T) {
	dir := t.TempDir()
	e := &ToolExecutor{
		ProjectRoot: dir,
		Policy: &PolicyConfig{
			BashAllowlist: []string{"echo *"},
			BashDenylist:  mergeDenylist(nil),
		},
	}

	// Permitted command runs.
	out, err := e.Execute(context.Background(), "bash", `{"command":"echo hello-from-bash"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "hello-from-bash") {
		t.Errorf("allowed command output = %q, want it to contain the echoed text", out)
	}

	// Command outside the allowlist is refused.
	out, err = e.Execute(context.Background(), "bash", `{"command":"cat /etc/passwd"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "permission denied") {
		t.Errorf("output = %q, want denial for a command outside the allowlist", out)
	}
}

func TestBash_DenialIsRecorded(t *testing.T) {
	var got []string
	e := &ToolExecutor{
		ProjectRoot: t.TempDir(),
		Policy: &PolicyConfig{
			BashAllowlist: []string{"echo *"},
			BashDenylist:  mergeDenylist(nil),
		},
		OnDenial: func(d Decision, toolName string, toolInput map[string]any) {
			got = append(got, toolName+":"+d.Action)
		},
	}
	if _, err := e.Execute(context.Background(), "bash", `{"command":"curl evil.example"}`); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(got) != 1 || got[0] != "bash:deny" {
		t.Errorf("denial callbacks = %v, want exactly one bash:deny (denials must be auditable)", got)
	}
}

func TestBash_NonZeroExitIsReturnedNotFatal(t *testing.T) {
	// A failing test suite is the normal case for a QA agent: the model needs to
	// see the output, and the run must not die.
	e := &ToolExecutor{
		ProjectRoot: t.TempDir(),
		Policy: &PolicyConfig{
			BashAllowlist: []string{"*"},
			BashDenylist:  mergeDenylist(nil),
		},
	}
	out, err := e.Execute(context.Background(), "bash", `{"command":"echo boom; exit 3"}`)
	if err != nil {
		t.Fatalf("Execute returned a fatal error for a non-zero exit: %v", err)
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("output = %q, want the command output preserved", out)
	}
}

func TestBash_RunsInProjectRoot(t *testing.T) {
	dir := t.TempDir()
	e := &ToolExecutor{
		ProjectRoot: dir,
		Policy: &PolicyConfig{
			BashAllowlist: []string{"pwd"},
			BashDenylist:  mergeDenylist(nil),
		},
	}
	out, err := e.Execute(context.Background(), "bash", `{"command":"pwd"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// macOS reports /private/var for /var; compare on the trailing element.
	if !strings.Contains(out, strings.TrimPrefix(dir, "/private")) {
		t.Errorf("pwd = %q, want the project root %q", strings.TrimSpace(out), dir)
	}
}
