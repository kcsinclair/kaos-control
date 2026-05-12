// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
)

// precheckState is the result of the init-event permission-mode precheck.
type precheckState int

const (
	precheckPending       precheckState = iota // init event not yet seen
	precheckPassed                             // init event seen and mode is acceptable
	precheckFailedMode                         // init event seen with a non-bypass mode
	precheckFailedTimeout                      // init event not seen within the timeout
)

// modeRemediation is the ordered list of remediation steps returned when the
// Claude Code process reports a non-bypassPermissions permission mode.
var modeRemediation = []string{
	"Run 'claude /permissions' in your terminal and enable bypass-permissions mode.",
	"Or set 'require_bypass_permissions: false' under 'agent:' in ~/.kaos-control/config.yaml to disable this check.",
	"Restart the agent after making the change.",
}

// timeoutRemediation is the ordered list of remediation steps returned when
// the Claude Code process does not emit a system/init event within the timeout.
var timeoutRemediation = []string{
	"The Claude Code binary did not emit a system/init event within the configured timeout.",
	"Verify that the 'claude' binary is installed and accessible in PATH.",
	"Increase 'init_event_timeout_seconds' under 'agent:' in ~/.kaos-control/config.yaml if needed.",
}

// runPrecheck drives the init-event permission-mode precheck alongside event
// forwarding. It reads from events, calls broadcast for each forwarded event,
// and applies the precheck when the first system/init event arrives.
//
// If the precheck fails (wrong mode or timeout), killFunc is called and the
// function returns the failure state and the observed permission mode (if any).
//
// If the events channel closes before the init event is seen (e.g. the binary
// crashed before emitting it), precheckPassed is returned and the normal exit
// path handles the failure reason.
func runPrecheck(
	events <-chan ProgressEvent,
	timeout time.Duration,
	requireBypass bool,
	runID string,
	broadcast func(hub.Event),
	killFunc func(),
) (precheckState, string) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	pending := true // precheck not yet resolved

	for {
		select {
		case <-timer.C:
			if pending {
				pending = false
				slog.Warn("agent: precheck timeout: no system/init event received",
					"run_id", runID, "timeout", timeout)
				killFunc()
				return precheckFailedTimeout, ""
			}

		case ev, ok := <-events:
			if !ok {
				// Channel closed; process exited without emitting an init event.
				// The normal exit handler will record the exit code as the failure reason.
				if pending {
					timer.Stop()
				}
				return precheckPassed, ""
			}

			// Forward the event to all hub subscribers regardless of precheck state.
			if broadcast != nil {
				payload := map[string]any{
					"run_id": runID,
					"line":   ev.Raw,
					"raw":    ev.Raw,
				}
				if ev.Event != nil {
					payload["event"] = ev.Event
				}
				broadcast(hub.Event{Type: "agent.progress", Payload: payload})
			}

			// Apply precheck only on the first system/init event.
			if pending && ev.Event != nil {
				evType, _ := ev.Event["type"].(string)
				evSubtype, _ := ev.Event["subtype"].(string)
				if evType == "system" && evSubtype == "init" {
					pending = false
					timer.Stop()

					mode, _ := ev.Event["permissionMode"].(string)
					if mode == "" {
						slog.Warn("agent: init event missing permissionMode field; treating as passed",
							"run_id", runID)
						return precheckPassed, ""
					}
					if mode == "bypassPermissions" {
						return precheckPassed, mode
					}
					if !requireBypass {
						slog.Warn("agent: permissionMode is not bypassPermissions but precheck is disabled",
							"run_id", runID, "mode", mode)
						return precheckPassed, mode
					}
					slog.Warn("agent: permissionMode is not bypassPermissions; terminating run",
						"run_id", runID, "mode", mode)
					killFunc()
					return precheckFailedMode, mode
				}
			}
		}
	}
}

// precheckFailureLogLine returns a JSON newline-terminated log entry for a
// precheck failure, suitable for appending to the run's on-disk log file.
func precheckFailureLogLine(runID, reason, observedMode string, remediation []string) []byte {
	payload := map[string]any{
		"type":                     "precheck_failure",
		"run_id":                   runID,
		"reason":                   reason,
		"observed_permission_mode": observedMode,
		"remediation":              remediation,
		"timestamp":                time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(payload)
	return append(b, '\n')
}
