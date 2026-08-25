// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

var (
	// ErrToolsSilentlyDropped indicates the provider accepted the request but dropped the tools parameter.
	ErrToolsSilentlyDropped = errors.New("tools parameter silently dropped by server/chat-template")
	// ErrToolsUnsupported indicates the model or provider explicitly does not support tools.
	ErrToolsUnsupported = errors.New("model does not support tools")
	// ErrModelNotFound indicates the requested model is absent from the
	// provider's /v1/models listing.
	ErrModelNotFound = errors.New("model not found on provider")
	// ErrEndpointUnreachable indicates the provider's /v1/models endpoint
	// could not be reached (connection refused, DNS failure, timeout, or a
	// 5xx response).
	ErrEndpointUnreachable = errors.New("provider endpoint unreachable")
)

// modelAvailabilityTimeout bounds the /v1/models preflight probe (NFR-1: the
// run must fail within 3 seconds when the model or endpoint is unavailable).
const modelAvailabilityTimeout = 3 * time.Second

// modelNotFoundRemediation is returned when the preflight model-availability
// probe confirms the provider is reachable but does not list the configured
// model.
var modelNotFoundRemediation = []string{
	"The configured model was not found in the provider's /v1/models listing.",
	"Verify the 'model:' value in this agent's config matches an ID the provider serves.",
	"If using Ollama or llama.cpp, confirm the model has been pulled/downloaded on the server.",
}

// endpointUnreachableRemediation is returned when the preflight
// model-availability probe cannot reach the provider's /v1/models endpoint.
var endpointUnreachableRemediation = []string{
	"The provider's /v1/models endpoint could not be reached within 3 seconds.",
	"Verify the provider's 'base_url' is correct and the server is running.",
	"Check network connectivity between kaos-control and the provider host.",
}

// openAIModelsListResponse is the standard OpenAI-compatible /v1/models
// listing shape. State is a best-effort llama.cpp extension: some servers
// report per-model load state ("unloaded"/"loading"/"loaded") alongside the
// id; it is absent on providers that don't support lazy loading.
type openAIModelsListResponse struct {
	Data []struct {
		ID    string `json:"id"`
		State string `json:"state,omitempty"`
	} `json:"data"`
}

// ModelAvailabilityStatus is the result of a successful /v1/models probe.
type ModelAvailabilityStatus struct {
	Available bool
	// LoadState is the provider-reported load state for the model, when the
	// provider exposes one (e.g. llama.cpp "unloaded"/"loading"/"loaded").
	// Empty when the provider doesn't report it.
	LoadState string
}

// probeModelAvailability issues a GET <base_url>/v1/models with a 3-second
// timeout and checks whether model is present in the listing. It is the
// shared implementation behind CheckModelAvailability (FR-2) and the
// warmup/lazy-load state extraction (FR-2/FR-5, Milestone 3).
func probeModelAvailability(ctx context.Context, client *http.Client, provider config.Provider, model string) (ModelAvailabilityStatus, error) {
	if client == nil {
		client = http.DefaultClient
	}
	probeCtx, cancel := context.WithTimeout(ctx, modelAvailabilityTimeout)
	defer cancel()

	url := buildEndpointURL(provider.BaseURL, "v1/models")
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url, nil)
	if err != nil {
		return ModelAvailabilityStatus{}, fmt.Errorf("%w: building request to %q: %v", ErrEndpointUnreachable, provider.Name, err)
	}
	applyProviderHeaders(req, provider)

	resp, err := client.Do(req)
	if err != nil {
		return ModelAvailabilityStatus{}, fmt.Errorf("%w: %s: %v", ErrEndpointUnreachable, provider.Name, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ModelAvailabilityStatus{}, fmt.Errorf("%w: reading /v1/models response from %q: %v", ErrEndpointUnreachable, provider.Name, err)
	}

	if resp.StatusCode != http.StatusOK {
		return ModelAvailabilityStatus{}, fmt.Errorf("%w: provider %q /v1/models returned HTTP %d: %s",
			ErrEndpointUnreachable, provider.Name, resp.StatusCode, extractErrorMessage(body))
	}

	var listing openAIModelsListResponse
	if err := json.Unmarshal(body, &listing); err != nil {
		return ModelAvailabilityStatus{}, fmt.Errorf("%w: parsing /v1/models response from %q: %v", ErrEndpointUnreachable, provider.Name, err)
	}

	for _, m := range listing.Data {
		if m.ID == model {
			return ModelAvailabilityStatus{Available: true, LoadState: m.State}, nil
		}
	}

	return ModelAvailabilityStatus{}, fmt.Errorf("%w: model %q not found on provider %q", ErrModelNotFound, model, provider.Name)
}

// CheckModelAvailability verifies that model exists in provider's /v1/models
// listing within a 3-second timeout. It returns (true, nil) when found;
// otherwise the returned error wraps ErrModelNotFound (model absent from a
// reachable provider) or ErrEndpointUnreachable (connection failure, timeout,
// or a non-2xx /v1/models response).
func CheckModelAvailability(ctx context.Context, client *http.Client, provider config.Provider, model string) (bool, error) {
	status, err := probeModelAvailability(ctx, client, provider, model)
	return status.Available, err
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIChatCompletionResponse struct {
	ID      string      `json:"id"`
	Choices []any       `json:"choices"`
	Usage   openAIUsage `json:"usage"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type openRouterModelsResponse struct {
	Data []struct {
		ID                  string   `json:"id"`
		Name                string   `json:"name"`
		SupportedParameters []string `json:"supported_parameters"`
	} `json:"data"`
}

// VerifyToolCapability verifies that the provider and model support function calling tools.
func VerifyToolCapability(ctx context.Context, client *http.Client, provider config.Provider, model string, tools []OpenAITool) error {
	if client == nil {
		client = http.DefaultClient
	}

	// 1. Gateway discovery (OpenRouter)
	if strings.Contains(provider.BaseURL, "openrouter.ai") || provider.Name == "openrouter" {
		if err := checkOpenRouterCapability(ctx, client, provider, model); err != nil {
			return err
		}
	}

	// 2. Token-delta comparison (with tools vs without tools)
	reqWithTools := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"tools":      tools,
		"max_tokens": 1,
		"stream":     false,
	}

	respWithTools, err := executePreflightCompletion(ctx, client, provider, reqWithTools)
	if err != nil {
		return err
	}

	reqWithoutTools := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"max_tokens": 1,
		"stream":     false,
	}

	respWithoutTools, err := executePreflightCompletion(ctx, client, provider, reqWithoutTools)
	if err != nil {
		return err
	}

	// Compare prompt tokens if usage information is provided
	if respWithTools.Usage.PromptTokens > 0 && respWithoutTools.Usage.PromptTokens > 0 {
		if respWithTools.Usage.PromptTokens <= respWithoutTools.Usage.PromptTokens {
			return fmt.Errorf("%w: provider %q model %q prompt_tokens unchanged with tools (%d with tools vs %d without tools)",
				ErrToolsSilentlyDropped, provider.Name, model, respWithTools.Usage.PromptTokens, respWithoutTools.Usage.PromptTokens)
		}
	}

	return nil
}

func checkOpenRouterCapability(ctx context.Context, client *http.Client, provider config.Provider, model string) error {
	modelsURL := buildEndpointURL(provider.BaseURL, "v1/models")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return fmt.Errorf("creating models request: %w", err)
	}
	applyProviderHeaders(req, provider)

	resp, err := client.Do(req)
	if err != nil {
		// If gateway models lookup fails with network error, fall through to token delta probing
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Non-200 on /v1/models: fall through to token delta probing
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var modelsResp openRouterModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil
	}

	for _, m := range modelsResp.Data {
		if m.ID == model {
			if len(m.SupportedParameters) > 0 {
				supportsTools := false
				for _, param := range m.SupportedParameters {
					if param == "tools" {
						supportsTools = true
						break
					}
				}
				if !supportsTools {
					return fmt.Errorf("%w: model %q on provider %q does not support 'tools' in supported_parameters",
						ErrToolsUnsupported, model, provider.Name)
				}
			}
			break
		}
	}
	return nil
}

func executePreflightCompletion(ctx context.Context, client *http.Client, provider config.Provider, reqBody map[string]any) (*openAIChatCompletionResponse, error) {
	url := buildEndpointURL(provider.BaseURL, "v1/chat/completions")
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling preflight request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating preflight request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	applyProviderHeaders(req, provider)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("preflight connection error to %s: %w", provider.Name, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading preflight response from %s: %w", provider.Name, err)
	}

	if resp.StatusCode == http.StatusBadRequest {
		// Mode B: explicit 400 rejection (e.g. Ollama "does not support tools")
		errMsg := extractErrorMessage(respBytes)
		return nil, fmt.Errorf("%w: provider %q model %q rejected tools parameter (HTTP 400): %s",
			ErrToolsUnsupported, provider.Name, reqBody["model"], errMsg)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := extractErrorMessage(respBytes)
		return nil, fmt.Errorf("provider %q returned HTTP %d: %s", provider.Name, resp.StatusCode, errMsg)
	}

	var parsed openAIChatCompletionResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshalling preflight response from %s: %w", provider.Name, err)
	}

	return &parsed, nil
}

func extractErrorMessage(respBytes []byte) string {
	var errObj struct {
		Error any `json:"error"`
	}
	if err := json.Unmarshal(respBytes, &errObj); err == nil && errObj.Error != nil {
		if s, ok := errObj.Error.(string); ok {
			return s
		}
		if m, ok := errObj.Error.(map[string]any); ok {
			if msg, ok := m["message"].(string); ok {
				return msg
			}
		}
	}
	trimmed := strings.TrimSpace(string(respBytes))
	if len(trimmed) > 200 {
		return trimmed[:200] + "..."
	}
	return trimmed
}

func buildEndpointURL(baseURL, endpointPath string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	endpointPath = strings.TrimLeft(endpointPath, "/")
	if strings.HasSuffix(baseURL, "/v1") && strings.HasPrefix(endpointPath, "v1/") {
		endpointPath = strings.TrimPrefix(endpointPath, "v1/")
	}
	return baseURL + "/" + endpointPath
}

func applyProviderHeaders(req *http.Request, provider config.Provider) {
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}
	for k, v := range provider.ExtraHeaders {
		req.Header.Set(k, v)
	}
}
