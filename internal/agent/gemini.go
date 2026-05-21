// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Gemini API Request Types
type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents          []geminiContent          `json:"contents"`
	SystemInstruction *geminiSystemInstruction `json:"systemInstruction,omitempty"`
}

// Gemini API Response Types
type geminiResponsePart struct {
	Text string `json:"text"`
}

type geminiResponseContent struct {
	Role  string               `json:"role"`
	Parts []geminiResponsePart `json:"parts"`
}

type geminiCandidate struct {
	Content      geminiResponseContent `json:"content"`
	FinishReason string                `json:"finishReason"`
	Index        int                   `json:"index"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiResponseChunk struct {
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
}

// GeminiDriver implements Driver by sending requests to the Google Gemini REST API.
type GeminiDriver struct {
	HTTPClient *http.Client
}

// geminiProcess implements Process for a streaming Gemini request.
type geminiProcess struct {
	cancel   context.CancelFunc
	progress chan ProgressEvent
	stderr   *ringBuf
	done     chan error
}

func (p *geminiProcess) Wait() error              { return <-p.done }
func (p *geminiProcess) Progress() <-chan ProgressEvent { return p.progress }
func (p *geminiProcess) StderrTail() string        { return p.stderr.String() }
func (p *geminiProcess) Kill() error {
	p.cancel()
	return nil
}

// Start implements Driver. It resolves the API key and base URL, builds the request body,
// and spawns a goroutine to stream responses using Go's incremental JSON stream decoder.
func (d *GeminiDriver) Start(ctx context.Context, run Run) (Process, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	baseURL := os.Getenv("GEMINI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}

	// Separate system and user prompts.
	systemPrompt, userPrompt := splitPrompt(run.PromptText)

	// Build JSON request body.
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: userPrompt},
				},
			},
		},
	}
	if systemPrompt != "" {
		reqBody.SystemInstruction = &geminiSystemInstruction{
			Parts: []geminiPart{
				{Text: systemPrompt},
			},
		}
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling gemini request: %w", err)
	}

	// Build HTTP request.
	apiURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?key=%s", baseURL, run.Model, apiKey)
	httpReq, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("building gemini http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

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

	proc := &geminiProcess{
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
			slog.Warn("gemini agent: creating log dir failed", "path", run.LogPath, "err", err)
		} else if f, err := os.OpenFile(run.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err != nil {
			slog.Warn("gemini agent: opening log file failed", "path", run.LogPath, "err", err)
		} else {
			logFile = f
			fmt.Fprintf(logFile, "# kaos-control agent run %s\n# agent=%s role=%s driver=gemini model=%s\n# started=%s\n",
				run.RunID, run.AgentName, run.Role, run.Model, time.Now().Format(time.RFC3339))
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
			errMsgBytes, _ := io.ReadAll(resp.Body)
			msg := fmt.Sprintf("gemini returned HTTP %d: %s", resp.StatusCode, string(errMsgBytes))
			rb.Write([]byte(msg))
			writeLog("# error: " + msg)
			doneCh <- fmt.Errorf("%s", msg)
			return
		}

		// Decode the JSON array streamed by Gemini.
		dec := json.NewDecoder(resp.Body)

		// Stream starts with opening delimiter '['.
		t, err := dec.Token()
		if err != nil {
			msg := fmt.Sprintf("reading stream start: %s", err.Error())
			rb.Write([]byte(msg))
			writeLog("# error: " + msg)
			doneCh <- err
			return
		}
		delim, ok := t.(json.Delim)
		if !ok || delim != '[' {
			msg := fmt.Sprintf("expected stream start '[', got %v", t)
			rb.Write([]byte(msg))
			writeLog("# error: " + msg)
			doneCh <- fmt.Errorf("%s", msg)
			return
		}

		var fullResponse strings.Builder
		var lastDone *geminiResponseChunk

		// Stream candidate response chunks using dec.More() inside the array.
		for dec.More() {
			var chunk geminiResponseChunk
			if err := dec.Decode(&chunk); err != nil {
				msg := fmt.Sprintf("decoding stream chunk: %s", err.Error())
				rb.Write([]byte(msg))
				writeLog("# error: " + msg)
				doneCh <- err
				return
			}

			// Capture the last chunk so we can extract usage stats later
			lastDone = &chunk

			// Extract text chunk from candidates
			var textChunk string
			if len(chunk.Candidates) > 0 {
				candidate := chunk.Candidates[0]
				for _, part := range candidate.Content.Parts {
					textChunk += part.Text
				}
			}

			if textChunk != "" {
				fullResponse.WriteString(textChunk)
			}

			chunkBytes, err := json.Marshal(chunk)
			var rawLine string
			if err == nil {
				rawLine = string(chunkBytes)
			} else {
				rawLine = fmt.Sprintf(`{"type":"chunk","text":%q}`, textChunk)
			}

			writeLog(rawLine)

			// Progress event containing the chunk text for output
			ev := ProgressEvent{
				Raw: rawLine,
				Event: map[string]any{
					"type": "output",
					"text": textChunk,
				},
			}

			select {
			case progressCh <- ev:
			default:
			}
		}

		// Read the closing delimiter ']'.
		_, _ = dec.Token()

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
			if summary := formatGeminiSummary(lastDone); summary != "" {
				writeLog("\n" + summary)
			}
		}

		select {
		case progressCh <- completed:
		default:
		}

		doneCh <- nil
	}()

	return proc, nil
}

// formatGeminiSummary formats the final metadata block.
func formatGeminiSummary(chunk *geminiResponseChunk) string {
	var parts []string
	if len(chunk.Candidates) > 0 && chunk.Candidates[0].FinishReason != "" {
		parts = append(parts, "finish_reason="+chunk.Candidates[0].FinishReason)
	}
	if chunk.UsageMetadata.PromptTokenCount > 0 {
		parts = append(parts, fmt.Sprintf("prompt_tokens=%d", chunk.UsageMetadata.PromptTokenCount))
	}
	if chunk.UsageMetadata.CandidatesTokenCount > 0 {
		parts = append(parts, fmt.Sprintf("candidates_tokens=%d", chunk.UsageMetadata.CandidatesTokenCount))
	}
	if chunk.UsageMetadata.TotalTokenCount > 0 {
		parts = append(parts, fmt.Sprintf("total_tokens=%d", chunk.UsageMetadata.TotalTokenCount))
	}
	if len(parts) == 0 {
		return ""
	}
	return "# summary: " + strings.Join(parts, " ")
}
