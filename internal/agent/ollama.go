package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	go func() {
		defer cancel()
		defer close(progressCh)

		// Emit "started" event.
		select {
		case progressCh <- ProgressEvent{Raw: "started"}:
		default:
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			rb.Write([]byte(err.Error()))
			doneCh <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			msg := fmt.Sprintf("ollama returned HTTP %d", resp.StatusCode)
			rb.Write([]byte(msg))
			doneCh <- fmt.Errorf("%s", msg)
			return
		}

		// Stream NDJSON response lines.
		var fullResponse strings.Builder
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if line == "" {
				continue
			}
			ev := ProgressEvent{Raw: line}
			var parsed map[string]any
			if jsonErr := json.Unmarshal([]byte(line), &parsed); jsonErr == nil {
				ev.Event = parsed
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
		select {
		case progressCh <- completed:
		default:
		}

		doneCh <- nil
	}()

	return proc, nil
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
