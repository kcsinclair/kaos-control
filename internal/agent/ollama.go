// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// OllamaDriver implements Driver by sending requests to an Ollama REST API.
type OllamaDriver struct {
	Instances  []config.OllamaInstance
	HTTPClient *http.Client
}

// ollamaProcess implements Process for a streaming Ollama request.
type ollamaProcess struct {
	cancel   context.CancelFunc
	progress chan ProgressEvent
	stderr   *ringBuf
	done     chan error
}

func (p *ollamaProcess) Wait() error              { return <-p.done }
func (p *ollamaProcess) Progress() <-chan ProgressEvent { return p.progress }
func (p *ollamaProcess) StderrTail() string        { return p.stderr.String() }
func (p *ollamaProcess) Kill() error {
	p.cancel()
	return nil
}

// Start implements Driver. It resolves the instance, builds the request body,
// and spawns a goroutine to stream NDJSON events.
func (d *OllamaDriver) Start(ctx context.Context, run Run) (Process, error) {
	// Resolve the instance by name stored in the Run's AgentName field — we
	// look it up via the driver's Instances list. The caller stores the
	// OllamaInstanceName in Run.Model (reused) would be confusing; instead the
	// Manager copies it into a dedicated field. We transmit it via a small
	// sentinel embedded in PromptText is also wrong. The cleanest approach at
	// this call site is to carry the instance name via Run itself, so we extend
	// Run with an OllamaInstanceName field.  See Run struct in agent.go.
	instanceName := run.OllamaInstanceName
	var inst *config.OllamaInstance
	for i := range d.Instances {
		if d.Instances[i].Name == instanceName {
			inst = &d.Instances[i]
			break
		}
	}
	if inst == nil {
		return nil, fmt.Errorf("ollama instance %q not found", instanceName)
	}

	// Separate system and user prompts.
	systemPrompt, userPrompt := splitPrompt(run.PromptText)

	// Choose endpoint.
	endpoint := run.OllamaEndpoint
	if endpoint == "" {
		endpoint = "chat"
	}

	// Build JSON body.
	var bodyBytes []byte
	var err error
	switch endpoint {
	case "generate":
		body := map[string]any{
			"model":  run.Model,
			"prompt": userPrompt,
			"stream": true,
		}
		if systemPrompt != "" {
			body["system"] = systemPrompt
		}
		bodyBytes, err = json.Marshal(body)
	default: // "chat"
		var messages []map[string]any
		if systemPrompt != "" {
			messages = append(messages, map[string]any{"role": "system", "content": systemPrompt})
		}
		messages = append(messages, map[string]any{"role": "user", "content": userPrompt})
		bodyBytes, err = json.Marshal(map[string]any{
			"model":    run.Model,
			"messages": messages,
			"stream":   true,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("marshalling ollama request: %w", err)
	}

	// Build HTTP request.
	apiURL := inst.BaseURL + "/api/" + endpoint
	httpReq, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("building ollama http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if inst.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+inst.APIKey)
	}

	// Timeout: use TimeoutMinutes from run (0 → 5 minutes default).
	timeout := time.Duration(run.TimeoutMinutes) * time.Minute
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	httpReq = httpReq.WithContext(runCtx)

	rb := newRingBuf(4 * 1024)
	progressCh := make(chan ProgressEvent, 64)
	doneCh := make(chan error, 1)

	proc := &ollamaProcess{
		cancel:   cancel,
		progress: progressCh,
		stderr:   rb,
		done:     doneCh,
	}

	client := d.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	// Open the per-run log file if configured. Mirrors the ClaudeCodeDriver
	// header/footer convention so on-disk run logs are consistent across drivers.
	var logFile *os.File
	if run.LogPath != "" {
		if err := os.MkdirAll(filepath.Dir(run.LogPath), 0o755); err != nil {
			slog.Warn("ollama agent: creating log dir failed", "path", run.LogPath, "err", err)
		} else if f, err := os.OpenFile(run.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err != nil {
			slog.Warn("ollama agent: opening log file failed", "path", run.LogPath, "err", err)
		} else {
			logFile = f
			fmt.Fprintf(logFile, "# kaos-control agent run %s\n# agent=%s role=%s driver=ollama instance=%s model=%s endpoint=%s\n# started=%s\n",
				run.RunID, run.AgentName, run.Role, instanceName, run.Model, endpoint, time.Now().Format(time.RFC3339))
			if systemPrompt != "" {
				fmt.Fprintf(logFile, "\n# system_prompt:\n%s\n", systemPrompt)
			}
			fmt.Fprintf(logFile, "\n# user_prompt:\n%s\n\n", userPrompt)
		}
	}

	// writeLog appends a line + newline to the log file (no-op if not configured).
	writeLog := func(s string) {
		if logFile != nil {
			_, _ = logFile.WriteString(s)
			if !strings.HasSuffix(s, "\n") {
				_, _ = logFile.WriteString("\n")
			}
		}
	}

	go func() {
		defer cancel()
		defer close(progressCh)
		defer func() {
			if logFile != nil {
				fmt.Fprintf(logFile, "\n# finished=%s\n", time.Now().Format(time.RFC3339))
				_ = logFile.Close()
			}
		}()

		// Emit "started" event.
		writeLog("# event: started")
		select {
		case progressCh <- ProgressEvent{Raw: "started"}:
		default:
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			rb.Write([]byte(err.Error()))
			writeLog("# error: " + err.Error())
			doneCh <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			msg := fmt.Sprintf("ollama returned HTTP %d", resp.StatusCode)
			rb.Write([]byte(msg))
			writeLog("# error: " + msg)
			doneCh <- fmt.Errorf("%s", msg)
			return
		}

		// Stream NDJSON response lines.
		var fullResponse strings.Builder
		var lastDone map[string]any // last NDJSON event with "done": true — carries Ollama's run stats.
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if line == "" {
				continue
			}
			writeLog(line)
			ev := ProgressEvent{Raw: line}
			var parsed map[string]any
			if jsonErr := json.Unmarshal([]byte(line), &parsed); jsonErr == nil {
				ev.Event = parsed
				if d, ok := parsed["done"].(bool); ok && d {
					lastDone = parsed
				}
				// Accumulate response text for final event.
				if endpoint == "generate" {
					if r, ok := parsed["response"].(string); ok {
						fullResponse.WriteString(r)
					}
				} else {
					if msg, ok := parsed["message"].(map[string]any); ok {
						if c, ok := msg["content"].(string); ok {
							fullResponse.WriteString(c)
						}
					}
				}
			}
			select {
			case progressCh <- ev:
			default:
			}
		}

		if scanErr := sc.Err(); scanErr != nil {
			rb.Write([]byte(scanErr.Error()))
			writeLog("# error: " + scanErr.Error())
			doneCh <- scanErr
			return
		}

		// Emit completed event with full response.
		completed := ProgressEvent{
			Raw: "completed",
			Event: map[string]any{
				"type":     "completed",
				"response": fullResponse.String(),
			},
		}
		writeLog("# event: completed")
		writeLog(fullResponse.String())
		if lastDone != nil {
			writeLog("\n" + formatOllamaSummary(lastDone))
		}
		select {
		case progressCh <- completed:
		default:
		}

		doneCh <- nil
	}()

	return proc, nil
}

// formatOllamaSummary turns the final `done:true` NDJSON event into a one-line
// human-readable summary suitable for the tail of the run log. Durations come
// from the Ollama API in nanoseconds; eval tok/s is derived from eval_count
// over eval_duration when both are present.
func formatOllamaSummary(done map[string]any) string {
	asInt := func(v any) (int64, bool) {
		switch n := v.(type) {
		case float64:
			return int64(n), true
		case int64:
			return n, true
		case int:
			return int64(n), true
		}
		return 0, false
	}
	asDur := func(v any) (time.Duration, bool) {
		if ns, ok := asInt(v); ok {
			return time.Duration(ns), true
		}
		return 0, false
	}

	var parts []string
	if reason, ok := done["done_reason"].(string); ok && reason != "" {
		parts = append(parts, "done_reason="+reason)
	}
	if total, ok := asDur(done["total_duration"]); ok {
		parts = append(parts, fmt.Sprintf("total=%s", total.Round(time.Millisecond)))
	}
	if load, ok := asDur(done["load_duration"]); ok {
		parts = append(parts, fmt.Sprintf("load=%s", load.Round(time.Millisecond)))
	}
	if pec, ok := asInt(done["prompt_eval_count"]); ok {
		if ped, ok2 := asDur(done["prompt_eval_duration"]); ok2 && ped > 0 {
			rate := float64(pec) / ped.Seconds()
			parts = append(parts, fmt.Sprintf("prompt_eval=%d (%s, %.1f tok/s)", pec, ped.Round(time.Millisecond), rate))
		} else {
			parts = append(parts, fmt.Sprintf("prompt_eval=%d", pec))
		}
	}
	if ec, ok := asInt(done["eval_count"]); ok {
		if ed, ok2 := asDur(done["eval_duration"]); ok2 && ed > 0 {
			rate := float64(ec) / ed.Seconds()
			parts = append(parts, fmt.Sprintf("eval=%d (%s, %.1f tok/s)", ec, ed.Round(time.Millisecond), rate))
		} else {
			parts = append(parts, fmt.Sprintf("eval=%d", ec))
		}
	}
	return "# summary: " + strings.Join(parts, " ")
}

// splitPrompt splits a prompt on the ---SYSTEM--- / ---USER--- delimiter
// convention. If the delimiter is absent the entire text is the user prompt.
func splitPrompt(text string) (system, user string) {
	const delim = "---SYSTEM---"
	const userDelim = "---USER---"

	sysIdx := strings.Index(text, delim)
	if sysIdx < 0 {
		return "", text
	}

	after := text[sysIdx+len(delim):]
	userIdx := strings.Index(after, userDelim)
	if userIdx < 0 {
		// Everything after ---SYSTEM--- is the system prompt; no user section.
		return strings.TrimSpace(after), ""
	}

	return strings.TrimSpace(after[:userIdx]), strings.TrimSpace(after[userIdx+len(userDelim):])
}
