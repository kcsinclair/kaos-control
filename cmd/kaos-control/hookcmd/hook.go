// SPDX-License-Identifier: AGPL-3.0-or-later

// Package hookcmd implements the `kaos-control hook-helper` subcommand.
//
// Claude Code spawns this binary on every PreToolUse event (FR4). It reads
// the tool-call JSON from stdin, POSTs it to the kaos-control permission
// endpoint, and writes the response JSON to stdout. Exit code is always 0 so
// Claude Code does not treat the hook as an error.
package hookcmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Run is the entry point for the hook-helper subcommand.
// args should be os.Args[2:] (i.e. everything after "hook-helper").
func Run(args []string) {
	fs := flag.NewFlagSet("hook-helper", flag.ContinueOnError)
	server := fs.String("server", "", "kaos-control server address (host:port)")
	runID := fs.String("run-id", "", "agent run ID")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		// ContinueOnError: parse error already printed to stderr.
		writeAllow()
		return
	}

	if *server == "" || *runID == "" {
		fmt.Fprintln(os.Stderr, "hook-helper: --server and --run-id are required")
		writeAllow()
		return
	}

	// Read the per-run secret from the environment (FR5).
	secret := os.Getenv("KC_HOOK_SECRET")
	if secret == "" {
		fmt.Fprintln(os.Stderr, "hook-helper: KC_HOOK_SECRET not set")
		writeDeny("KC_HOOK_SECRET not set")
		return
	}

	// Read tool-call JSON from stdin.
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hook-helper: reading stdin: %v\n", err)
		writeDeny("stdin read error")
		return
	}

	url := fmt.Sprintf("http://%s/api/agent/%s/permission", *server, *runID)
	resp, err := postWithRetry(url, secret, body)
	if err != nil {
		// Server unreachable after retry → deny (NFR2).
		log.Printf("hook-helper: server unreachable: %v", err)
		writeDeny("server unreachable")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		writeDeny("response read error")
		return
	}

	// Pass the response through to stdout unchanged.
	_, _ = os.Stdout.Write(respBody)
}

// postWithRetry POSTs to the endpoint with the secret in the Authorization
// header. On connection failure it retries once after 500 ms (NFR2).
func postWithRetry(url, secret string, body []byte) (*http.Response, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	attempt := func() (*http.Response, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+secret)
		return client.Do(req)
	}

	resp, err := attempt()
	if err == nil {
		return resp, nil
	}

	// Wait 500 ms and retry once.
	time.Sleep(500 * time.Millisecond)
	return attempt()
}

// writeAllow writes a JSON allow response to stdout.
// Used when we cannot determine the right answer (e.g. missing args).
func writeAllow() {
	writeResponse("allow", "")
}

// writeDeny writes a JSON deny response to stdout.
func writeDeny(reason string) {
	writeResponse("deny", reason)
}

// writeResponse emits a Claude-Code-compatible PreToolUse hook response on
// stdout. Schema (per Claude's hooks API):
//
//   {"hookSpecificOutput":{
//     "hookEventName":"PreToolUse",
//     "permissionDecision":"allow"|"deny"|"ask",
//     "permissionDecisionReason":"..."
//   }}
//
// The simpler `{"decision":"..."}` shape that prior versions emitted is
// silently ignored by Claude, which falls back to interactive prompts and —
// in headless `-p` mode — surfaces as "you haven't granted it yet" errors.
func writeResponse(decision, reason string) {
	inner := map[string]string{
		"hookEventName":      "PreToolUse",
		"permissionDecision": decision,
	}
	if reason != "" {
		inner["permissionDecisionReason"] = reason
	}
	out := map[string]any{"hookSpecificOutput": inner}
	b, _ := json.Marshal(out)
	_, _ = os.Stdout.Write(b)
}
