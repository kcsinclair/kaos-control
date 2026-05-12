// SPDX-License-Identifier: AGPL-3.0-or-later

// fake_precheck_claude is a tiny Claude Code stand-in used by the precheck
// integration tests. It emits configurable stream-json events to stdout and
// then exits (or blocks), allowing tests to exercise the permission-mode
// precheck logic without a real Claude Code binary.
//
// Environment variables:
//
//	FAKE_CLAUDE_MODE           — value to emit in the permissionMode field.
//	                             If empty, emits the init event without the field.
//	                             If literally "omit-init", emits NO init event at all.
//	FAKE_CLAUDE_DELAY_MS       — milliseconds to sleep before emitting the init event.
//	FAKE_CLAUDE_HOLD_AFTER_INIT — if "true", blocks indefinitely after emitting the
//	                             init event (lets the precheck terminate the process).
//	FAKE_CLAUDE_ARGS_FILE      — if set, the binary writes os.Args as a JSON array
//	                             to this file before doing anything else.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	// Write argv to file for assertion in I4 (dual-flag test).
	if argsFile := os.Getenv("FAKE_CLAUDE_ARGS_FILE"); argsFile != "" {
		data, _ := json.Marshal(os.Args)
		_ = os.WriteFile(argsFile, data, 0o644)
	}

	mode := os.Getenv("FAKE_CLAUDE_MODE")
	delayMS := os.Getenv("FAKE_CLAUDE_DELAY_MS")
	hold := os.Getenv("FAKE_CLAUDE_HOLD_AFTER_INIT") == "true"

	if d, _ := strconv.Atoi(delayMS); d > 0 {
		time.Sleep(time.Duration(d) * time.Millisecond)
	}

	if mode != "omit-init" {
		ev := map[string]any{
			"type":    "system",
			"subtype": "init",
		}
		if mode != "" {
			ev["permissionMode"] = mode
		}
		data, _ := json.Marshal(ev)
		fmt.Println(string(data))
	}

	if hold {
		// Block until killed by the precheck supervisor.
		// time.Sleep avoids the Go deadlock detector that fires on select{}.
		time.Sleep(24 * time.Hour)
	}

	// Emit a successful result event so the supervisor sees a clean exit.
	result := map[string]any{"type": "result", "subtype": "success"}
	data, _ := json.Marshal(result)
	fmt.Println(string(data))
}
