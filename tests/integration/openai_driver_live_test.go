// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
)

// ── Milestone 4 — Live Target Verification Tests ─────────────────────────────

func isLiveTargetReachable(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// TestOpenAIDriver_LiveTarget_LlamaServer tests live execution against llama.cpp (leia.packsin.com:7442).
func TestOpenAIDriver_LiveTarget_LlamaServer(t *testing.T) {
	const llamaAddr = "leia.packsin.com:7442"
	const llamaURL = "http://" + llamaAddr

	if !isLiveTargetReachable(llamaAddr, 2*time.Second) {
		t.Skipf("llama.cpp live server %s not reachable; skipping live target test", llamaAddr)
	}

	models := []string{
		"gemma-4-26B-A4B-it-UD-Q8_K_XL",
		"gpt-oss-20b-Q8_0",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			root := t.TempDir()
			ideaPath := filepath.Join(root, "lifecycle", "ideas", "live-test.md")
			_ = os.MkdirAll(filepath.Dir(ideaPath), 0o755)
			_ = os.WriteFile(ideaPath, []byte("# Live Idea\nTest live tool calling."), 0o644)

			reqPath := filepath.Join(root, "lifecycle", "requirements", "live-test-2.md")
			_ = os.MkdirAll(filepath.Dir(reqPath), 0o755)

			drv := &agent.OpenAICompatibleDriver{
				Providers: []config.Provider{
					{
						Name:    "live-llama",
						BaseURL: llamaURL,
						Driver:  "openai-compatible",
					},
				},
				HTTPClient: &http.Client{Timeout: 30 * time.Second},
			}

			run := agent.Run{
				RunID:             "live-llama-run",
				AgentName:         "live-llama-agent",
				Role:              "analyst",
				Model:             model,
				PromptText:        "---SYSTEM---\nYou are an analyst agent. Read lifecycle/ideas/live-test.md using read_file and write a concise requirement file to lifecycle/requirements/live-test-2.md using write_file.\n---USER---\nPlease execute the requirement creation now.",
				ProjectRoot:       root,
				AllowedPaths:      []string{"lifecycle/ideas", "lifecycle/requirements"},
				ProviderName:      "live-llama",
				MaxToolIterations: 10,
				TimeoutMinutes:    2,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()

			start := time.Now()
			proc, err := drv.Start(ctx, run)
			if err != nil {
				t.Fatalf("Start failed against %s (%s): %v", llamaAddr, model, err)
			}

			_ = collectEvents(t, proc, 25*time.Second)
			if err := proc.Wait(); err != nil {
				t.Fatalf("Wait failed against %s (%s): %v", llamaAddr, model, err)
			}
			elapsed := time.Since(start)

			t.Logf("Live llama.cpp (%s) completed in %v", model, elapsed)
		})
	}
}

// TestOpenAIDriver_LiveTarget_Ollama tests live execution against Ollama (leia.packsin.com:11434).
func TestOpenAIDriver_LiveTarget_Ollama(t *testing.T) {
	const ollamaAddr = "leia.packsin.com:11434"
	const ollamaURL = "http://" + ollamaAddr

	if !isLiveTargetReachable(ollamaAddr, 2*time.Second) {
		t.Skipf("Ollama live server %s not reachable; skipping live target test", ollamaAddr)
	}

	models := []string{
		"qwen3-coder:30b",
		"gemma4:26b",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			root := t.TempDir()
			ideaPath := filepath.Join(root, "lifecycle", "ideas", "live-ollama.md")
			_ = os.MkdirAll(filepath.Dir(ideaPath), 0o755)
			_ = os.WriteFile(ideaPath, []byte("# Live Idea Ollama\nTest tool calling."), 0o644)

			reqPath := filepath.Join(root, "lifecycle", "requirements", "live-ollama-2.md")
			_ = os.MkdirAll(filepath.Dir(reqPath), 0o755)

			drv := &agent.OpenAICompatibleDriver{
				Providers: []config.Provider{
					{
						Name:    "live-ollama",
						BaseURL: ollamaURL,
						Driver:  "openai-compatible",
					},
				},
				HTTPClient: &http.Client{Timeout: 30 * time.Second},
			}

			run := agent.Run{
				RunID:             "live-ollama-run",
				AgentName:         "live-ollama-agent",
				Role:              "analyst",
				Model:             model,
				PromptText:        "---SYSTEM---\nYou are an analyst agent. Read lifecycle/ideas/live-ollama.md using read_file and write a concise requirement file to lifecycle/requirements/live-ollama-2.md using write_file.\n---USER---\nPlease execute now.",
				ProjectRoot:       root,
				AllowedPaths:      []string{"lifecycle/ideas", "lifecycle/requirements"},
				ProviderName:      "live-ollama",
				MaxToolIterations: 10,
				TimeoutMinutes:    2,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()

			start := time.Now()
			proc, err := drv.Start(ctx, run)
			if err != nil {
				t.Fatalf("Start failed against %s (%s): %v", ollamaAddr, model, err)
			}

			_ = collectEvents(t, proc, 25*time.Second)
			if err := proc.Wait(); err != nil {
				t.Fatalf("Wait failed against %s (%s): %v", ollamaAddr, model, err)
			}
			elapsed := time.Since(start)

			t.Logf("Live Ollama (%s) completed in %v", model, elapsed)
		})
	}
}
