// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// ErrModelLoadTimeout indicates the model did not produce a first token
// within the dedicated loading timeout, most likely because it is still
// lazy-loading weights into memory (local-model-operability Milestone 3).
var ErrModelLoadTimeout = errors.New("model did not respond within the loading timeout")

// OpenAICompatibleDriver implements Driver by communicating with an OpenAI-compatible
// /v1/chat/completions endpoint using a multi-turn tool calling agent loop.
type OpenAICompatibleDriver struct {
	Providers  []config.Provider
	HTTPClient *http.Client
}

// openAIProcess implements Process for an OpenAI-compatible driver run.
type openAIProcess struct {
	cancel   context.CancelFunc
	progress chan ProgressEvent
	stderr   *ringBuf
	done     chan error
}

func (p *openAIProcess) Wait() error                    { return <-p.done }
func (p *openAIProcess) Progress() <-chan ProgressEvent { return p.progress }
func (p *openAIProcess) StderrTail() string             { return p.stderr.String() }
func (p *openAIProcess) Kill() error {
	p.cancel()
	return nil
}

type openAIChatMessage struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type openAIStreamChoice struct {
	Index int `json:"index"`
	Delta struct {
		Role             string `json:"role"`
		Content          string `json:"content"`
		ReasoningContent string `json:"reasoning_content"`
		ToolCalls        []struct {
			Index    int    `json:"index"`
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

type openAIStreamChunk struct {
	ID      string               `json:"id"`
	Choices []openAIStreamChoice `json:"choices"`
}

// Start initiates the OpenAI-compatible agent run.
func (d *OpenAICompatibleDriver) Start(ctx context.Context, run Run) (Process, error) {
	providerName := run.ProviderName
	var prov *config.Provider
	for i := range d.Providers {
		if d.Providers[i].Name == providerName {
			prov = &d.Providers[i]
			break
		}
	}
	if prov == nil {
		return nil, fmt.Errorf("provider %q not found", providerName)
	}

	if run.Model == "" {
		return nil, fmt.Errorf("model is required for openai-compatible agent %q", run.AgentName)
	}

	client := d.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	tools := DefaultOpenAITools()

	// Timeout configuration
	timeout := time.Duration(run.TimeoutMinutes) * time.Minute
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)

	// Preflight capability verification (FR-5b)
	if err := VerifyToolCapability(runCtx, client, *prov, run.Model, tools); err != nil {
		cancel()
		return nil, fmt.Errorf("preflight capability check failed: %w", err)
	}

	maxIterations := run.MaxToolIterations
	if maxIterations <= 0 {
		maxIterations = 25
	}

	systemPrompt, userPrompt := splitPrompt(run.PromptText)

	var messages []openAIChatMessage
	if systemPrompt != "" {
		messages = append(messages, openAIChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, openAIChatMessage{Role: "user", Content: userPrompt})

	rb := newRingBuf(4 * 1024)
	progressCh := make(chan ProgressEvent, 64)
	doneCh := make(chan error, 1)

	proc := &openAIProcess{
		cancel:   cancel,
		progress: progressCh,
		stderr:   rb,
		done:     doneCh,
	}

	executor := &ToolExecutor{
		ProjectRoot:  run.ProjectRoot,
		AllowedPaths: run.AllowedPaths,
	}

	mask := func(s string) string {
		if prov.APIKey != "" {
			return strings.ReplaceAll(s, prov.APIKey, "***")
		}
		return s
	}

	var logFile *os.File
	if run.LogPath != "" {
		if err := os.MkdirAll(filepath.Dir(run.LogPath), 0o755); err != nil {
			slog.Warn("openai-compatible agent: creating log dir failed", "path", run.LogPath, "err", err)
		} else if f, err := os.OpenFile(run.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err != nil {
			slog.Warn("openai-compatible agent: opening log file failed", "path", run.LogPath, "err", err)
		} else {
			logFile = f
			fmt.Fprintf(logFile, "# kaos-control agent run %s\n# agent=%s role=%s driver=openai-compatible provider=%s base_url=%s model=%s\n# started=%s\n",
				run.RunID, run.AgentName, run.Role, prov.Name, prov.BaseURL, run.Model, time.Now().Format(time.RFC3339))
			if systemPrompt != "" {
				fmt.Fprintf(logFile, "\n# system_prompt:\n%s\n", mask(systemPrompt))
			}
			fmt.Fprintf(logFile, "\n# user_prompt:\n%s\n\n", mask(userPrompt))
		}
	}

	writeLog := func(s string) {
		if logFile != nil {
			masked := mask(s)
			_, _ = logFile.WriteString(masked)
			if !strings.HasSuffix(masked, "\n") {
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

		startTime := time.Now()
		var ttftRecorded atomic.Bool
		var loadTimedOut atomic.Bool
		recordTTFT := func() {
			if ttftRecorded.CompareAndSwap(false, true) {
				if run.OnTTFT != nil {
					run.OnTTFT(time.Since(startTime).Milliseconds())
				}
				select {
				case progressCh <- ProgressEvent{
					Raw: "generating",
					Event: map[string]any{
						"type":  "status",
						"stage": "generating",
					},
				}:
				default:
				}
			}
		}

		// Warmup detection (Milestone 3): local inference servers often
		// lazy-load multi-gigabyte weights on the first request, pausing
		// 30-120s with no visible progress. The warmup watcher surfaces that
		// to the UI once 5s elapse without a token; the load-timeout watcher
		// bounds the wait with a dedicated, configurable timeout (default
		// 60s) distinct from the overall run timeout, so a stuck load fails
		// clearly instead of silently consuming the whole run budget.
		//
		// Both watchers run in their own goroutine and race their timer
		// against stopTimers, which the deferred cleanup below closes (and
		// waits on) before progressCh is closed — without that, a watcher
		// could still be sending on progressCh after this goroutine closes
		// it, which panics.
		loadTimeout := run.ModelLoadingTimeout
		if loadTimeout <= 0 {
			loadTimeout = 60 * time.Second
		}
		stopTimers := make(chan struct{})
		var timersWG sync.WaitGroup

		timersWG.Add(1)
		go func() {
			defer timersWG.Done()
			select {
			case <-time.After(5 * time.Second):
			case <-stopTimers:
				return
			}
			if ttftRecorded.Load() {
				return
			}
			select {
			case progressCh <- ProgressEvent{
				Raw: "warming_up",
				Event: map[string]any{
					"type":    "status",
					"stage":   "warming_up",
					"message": "Awaiting first token (model may be warming up)...",
				},
			}:
			case <-stopTimers:
			}
		}()

		timersWG.Add(1)
		go func() {
			defer timersWG.Done()
			select {
			case <-time.After(loadTimeout):
			case <-stopTimers:
				return
			}
			if ttftRecorded.Load() {
				return
			}
			loadTimedOut.Store(true)
			cancel()
		}()

		defer func() {
			close(stopTimers)
			timersWG.Wait()
		}()

		// wrapLoadTimeout replaces a context-cancellation error with a clear
		// ErrModelLoadTimeout when the cancellation was caused by the
		// load-timeout watcher (rather than a user Kill or the overall run
		// timeout).
		wrapLoadTimeout := func(err error) error {
			if loadTimedOut.Load() && !ttftRecorded.Load() {
				return fmt.Errorf("%w after %s", ErrModelLoadTimeout, loadTimeout)
			}
			return err
		}

		endpointURL := buildEndpointURL(prov.BaseURL, "v1/chat/completions")

		var recoveredTotalCount int

		for turn := 1; turn <= maxIterations; turn++ {
			if err := runCtx.Err(); err != nil {
				err = wrapLoadTimeout(err)
				rb.Write([]byte(mask(err.Error())))
				writeLog("# error: " + err.Error())
				doneCh <- err
				return
			}

			writeLog(fmt.Sprintf("\n# turn %d", turn))

			reqBody := map[string]any{
				"model":    run.Model,
				"messages": messages,
				"tools":    tools,
				"stream":   true,
			}
			bodyBytes, err := json.Marshal(reqBody)
			if err != nil {
				rb.Write([]byte(mask(err.Error())))
				writeLog("# error: " + err.Error())
				doneCh <- err
				return
			}

			httpReq, err := http.NewRequestWithContext(runCtx, http.MethodPost, endpointURL, bytes.NewReader(bodyBytes))
			if err != nil {
				rb.Write([]byte(mask(err.Error())))
				writeLog("# error: " + err.Error())
				doneCh <- err
				return
			}
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Accept", "text/event-stream")
			applyProviderHeaders(httpReq, *prov)

			resp, err := client.Do(httpReq)
			if err != nil {
				err = wrapLoadTimeout(err)
				rb.Write([]byte(mask(err.Error())))
				writeLog("# error: " + err.Error())
				doneCh <- wrapHTTPError(err, 0)
				return
			}

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				errMsg := fmt.Sprintf("provider %q returned HTTP %d: %s", prov.Name, resp.StatusCode, extractErrorMessage(respBody))
				rb.Write([]byte(mask(errMsg)))
				writeLog("# error: " + errMsg)
				doneCh <- wrapHTTPError(fmt.Errorf("%s", errMsg), resp.StatusCode)
				return
			}

			type toolCallBuilder struct {
				id       string
				callType string
				name     string
				args     strings.Builder
			}
			toolCallBuilders := make(map[int]*toolCallBuilder)
			var turnContent strings.Builder
			var finishReason string

			sc := bufio.NewScanner(resp.Body)
			sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

			for sc.Scan() {
				line := sc.Text()
				if line == "" {
					continue
				}
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				dataContent := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if dataContent == "[DONE]" {
					break
				}

				writeLog(dataContent)

				var chunk openAIStreamChunk
				if err := json.Unmarshal([]byte(dataContent), &chunk); err == nil {
					var rawMap map[string]any
					_ = json.Unmarshal([]byte(dataContent), &rawMap)
					select {
					case progressCh <- ProgressEvent{Raw: mask(dataContent), Event: rawMap}:
					default:
					}

					for _, choice := range chunk.Choices {
						if choice.Delta.Content != "" {
							recordTTFT()
							turnContent.WriteString(choice.Delta.Content)
						}
						for _, tc := range choice.Delta.ToolCalls {
							b, exists := toolCallBuilders[tc.Index]
							if !exists {
								b = &toolCallBuilder{callType: "function"}
								toolCallBuilders[tc.Index] = b
							}
							if tc.ID != "" {
								b.id = tc.ID
							}
							if tc.Type != "" {
								b.callType = tc.Type
							}
							if tc.Function.Name != "" {
								b.name += tc.Function.Name
							}
							if tc.Function.Arguments != "" {
								b.args.WriteString(tc.Function.Arguments)
							}
						}
						if choice.FinishReason != nil {
							finishReason = *choice.FinishReason
						}
					}
				}
			}

			scanErr := sc.Err()
			resp.Body.Close()
			if scanErr != nil {
				// A cancelled runCtx (load timeout, overall run timeout, or a
				// user Kill) is what actually explains the scan error; report
				// that cancellation instead of falling through to "no tool
				// calls" below, which would otherwise record an empty
				// response as a successful completion.
				if ctxErr := runCtx.Err(); ctxErr != nil {
					err := wrapLoadTimeout(ctxErr)
					rb.Write([]byte(mask(err.Error())))
					writeLog("# error: " + err.Error())
					doneCh <- err
					return
				}
				rb.Write([]byte(mask(scanErr.Error())))
				writeLog("# error: " + scanErr.Error())
				doneCh <- scanErr
				return
			}

			// Assemble structured tool calls ordered by index
			var indices []int
			for idx := range toolCallBuilders {
				indices = append(indices, idx)
			}
			sort.Ints(indices)

			var turnToolCalls []ToolCall
			for _, idx := range indices {
				b := toolCallBuilders[idx]
				callID := b.id
				if callID == "" {
					callID = fmt.Sprintf("call_%d", idx+1)
				}
				turnToolCalls = append(turnToolCalls, ToolCall{
					ID:   callID,
					Type: b.callType,
					Function: FunctionCallInfo{
						Name:      b.name,
						Arguments: b.args.String(),
					},
				})
			}

			// Fallback: Native tool-call recovery (FR-5a) if no structured tool calls were emitted
			if len(turnToolCalls) == 0 && turnContent.Len() > 0 {
				recovered, remaining := ParseNativeCalls(turnContent.String())
				if len(recovered) > 0 {
					turnToolCalls = recovered
					turnContent.Reset()
					turnContent.WriteString(remaining)
					recoveredTotalCount += len(recovered)
					writeLog(fmt.Sprintf("# recovered %d native tool call(s) (FR-5a)", len(recovered)))
				}
			}

			// If tool calls were requested, execute them and continue the loop
			if len(turnToolCalls) > 0 {
				contentStr := turnContent.String()
				var contentAny any
				if contentStr != "" {
					contentAny = contentStr
				}
				messages = append(messages, openAIChatMessage{
					Role:      "assistant",
					Content:   contentAny,
					ToolCalls: turnToolCalls,
				})

				for _, tc := range turnToolCalls {
					writeLog(fmt.Sprintf("# executing tool %s (id: %s) with args: %s", tc.Function.Name, tc.ID, tc.Function.Arguments))

					toolResult, execErr := executor.Execute(runCtx, tc.Function.Name, tc.Function.Arguments)
					if execErr != nil {
						rb.Write([]byte(mask(execErr.Error())))
						writeLog("# error executing tool: " + execErr.Error())
						doneCh <- execErr
						return
					}

					writeLog(fmt.Sprintf("# tool result (%s): %s", tc.ID, toolResult))
					messages = append(messages, openAIChatMessage{
						Role:       "tool",
						ToolCallID: tc.ID,
						Name:       tc.Function.Name,
						Content:    toolResult,
					})
				}
				continue
			}

			// No tool calls: final answer reached
			completed := ProgressEvent{
				Raw: "completed",
				Event: map[string]any{
					"type":     "completed",
					"response": turnContent.String(),
				},
			}
			writeLog("# event: completed")
			writeLog(turnContent.String())
			if recoveredTotalCount > 0 {
				writeLog(fmt.Sprintf("# summary: recovered_native_tool_calls=%d finish_reason=%s", recoveredTotalCount, finishReason))
			} else {
				writeLog(fmt.Sprintf("# summary: finish_reason=%s", finishReason))
			}
			select {
			case progressCh <- completed:
			default:
			}

			doneCh <- nil
			return
		}

		// Turn loop cap exceeded
		capErr := fmt.Errorf("max tool iterations cap (%d) reached without finish_reason: stop: %w", maxIterations, ErrMaxIterationsReached)
		rb.Write([]byte(mask(capErr.Error())))
		writeLog("# error: " + capErr.Error())
		doneCh <- capErr
	}()

	return proc, nil
}
