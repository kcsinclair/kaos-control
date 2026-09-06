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

// LlamaCPPDriver implements Driver by sending requests to a llama.cpp server.
type LlamaCPPDriver struct {
	Instances  []config.LlamaCPPInstance
	HTTPClient *http.Client
}

// llamaCPPProcess implements Process for a streaming llama.cpp request.
type llamaCPPProcess struct {
	cancel   context.CancelFunc
	progress chan ProgressEvent
	stderr   *ringBuf
	done     chan error
}

func (p *llamaCPPProcess) Wait() error              { return <-p.done }
func (p *llamaCPPProcess) Progress() <-chan ProgressEvent { return p.progress }
func (p *llamaCPPProcess) StderrTail() string       { return p.stderr.String() }
func (p *llamaCPPProcess) Kill() error {
	p.cancel()
	return nil
}

// Start implements Driver. It resolves the instance, builds the request body,
// and spawns a goroutine to stream response tokens.
func (d *LlamaCPPDriver) Start(ctx context.Context, run Run) (Process, error) {
	// Resolve the instance by name.
	instanceName := run.LlamaCPPInstanceName
	var inst *config.LlamaCPPInstance
	for i := range d.Instances {
		if d.Instances[i].Name == instanceName {
			inst = &d.Instances[i]
			break
		}
	}
	if inst == nil {
		return nil, fmt.Errorf("llama.cpp instance %q not found", instanceName)
	}

	// Separate system and user prompts.
	systemPrompt, userPrompt := splitPrompt(run.PromptText)

	// Apply defaults for inference params.
	maxTokens := run.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096 // reasonable default
	}
	temperature := run.Temperature
	if temperature <= 0 {
		temperature = 0.7 // reasonable default
	}

	// Build JSON body for chat completion.
	body := map[string]any{
		"model":       run.Model,
		"messages":    []map[string]any{},
		"stream":      true,
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}

	if systemPrompt != "" {
		body["messages"] = append(body["messages"].([]map[string]any), map[string]any{
			"role":    "system",
			"content": systemPrompt,
		})
	}
	body["messages"] = append(body["messages"].([]map[string]any), map[string]any{
		"role":    "user",
		"content": userPrompt,
	})

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshalling llama.cpp request: %w", err)
	}

	// Build HTTP request.
	apiURL := inst.BaseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("building llama.cpp http request: %w", err)
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

	proc := &llamaCPPProcess{
		cancel:   cancel,
		progress: progressCh,
		stderr:   rb,
		done:     doneCh,
	}

	client := d.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	// Open the per-run log file if configured.
	var logFile *os.File
	if run.LogPath != "" {
		if err := os.MkdirAll(filepath.Dir(run.LogPath), 0o755); err != nil {
			slog.Warn("llama.cpp agent: creating log dir failed", "path", run.LogPath, "err", err)
		} else if f, err := os.OpenFile(run.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err != nil {
			slog.Warn("llama.cpp agent: opening log file failed", "path", run.LogPath, "err", err)
		} else {
			logFile = f
			fmt.Fprintf(logFile, "# kaos-control agent run %s\n# agent=%s role=%s driver=llama.cpp instance=%s model=%s\n# started=%s\n",
				run.RunID, run.AgentName, run.Role, instanceName, run.Model, time.Now().Format(time.RFC3339))
			if systemPrompt != "" {
				fmt.Fprintf(logFile, "\n# system_prompt:\n%s\n", systemPrompt)
			}
			fmt.Fprintf(logFile, "\n# user_prompt:\n%s\n\n", userPrompt)
		}
	}

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
			msg := fmt.Sprintf("llama.cpp returned HTTP %d", resp.StatusCode)
			rb.Write([]byte(msg))
			writeLog("# error: " + msg)
			doneCh <- fmt.Errorf("%s", msg)
			return
		}

		// Stream JSON response chunks.
		var fullResponse strings.Builder
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "data: ") {
				line = line[6:]
			}
			writeLog(line)

			var chunk map[string]any
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue
			}

			ev := ProgressEvent{Raw: line, Event: chunk}

			// Accumulate response text.
			if choices, ok := chunk["choices"].([]any); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]any); ok {
					if delta, ok := choice["delta"].(map[string]any); ok {
						if content, ok := delta["content"].(string); ok {
							fullResponse.WriteString(content)
						}
					}
				}
			}

			select {
			case progressCh <- ev:
			default:
			}

			// Check for completion.
			if done, ok := chunk["done"].(bool); ok && done {
				continue
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
		select {
		case progressCh <- completed:
		default:
		}

		doneCh <- nil
	}()

	return proc, nil
}
