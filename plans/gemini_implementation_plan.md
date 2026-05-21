# Add Support for Gemini Models in kaos-control

This plan describes how we will introduce support for Google Gemini models as a first-class agent driver in `kaos-control`. Users will be able to configure agents in `lifecycle/config.yaml` with `driver: gemini` and select standard models like `gemini-2.5-flash` or `gemini-1.5-pro`.

## User Review Required

> [!IMPORTANT]
> The Gemini driver requires a Google Gemini API Key. By default, it will retrieve the key from the `GEMINI_API_KEY` environment variable. Ensure this environment variable is set on the system running `kaos-control`.

> [!NOTE]
> The Gemini API's streaming response (`:streamGenerateContent`) is delivered as a JSON array of response chunks (delimited by `[` and `]`, separated by commas) rather than Server-Sent Events (SSE) or simple NDJSON. We will implement an incremental JSON token-stream parser in Go to decode this format safely and efficiently in real time.

## Open Questions

There are no major open questions. The implementation follows the pattern established by the existing `ollama` driver to maintain 100% compatibility with the frontend's websocket event model.

---

## Proposed Changes

### Go Backend Components

We will create a new `GeminiDriver` that implements the pluggable `Driver` interface, integrates with the `agent.Manager`, and handles config validation.

#### [NEW] [gemini.go](file:///Users/keith/Code/kaos-control/internal/agent/gemini.go)

- Implement `GeminiDriver struct` with the `Start(ctx context.Context, run Run) (Process, error)` method.
- Implement `geminiProcess struct` implementing `Process`.
- Use the standard `splitPrompt` to extract system and user instructions.
- Target the Google Gemini AI Studio endpoint:
  `https://generativelanguage.googleapis.com/v1beta/models/{model}:streamGenerateContent?key={apiKey}`
- Parse the streaming JSON array response using `json.Decoder` with `.Token()` and `.More()` for memory efficiency.
- Emit progress events with the type `output` and contents `text` to match the web frontend's formatting expectations in `formatEvent()`.
- Record execution logs under `<logsDir>/<run_id>.log` with consistent headers and footers.

#### [MODIFY] [agent.go](file:///Users/keith/Code/kaos-control/internal/agent/agent.go)

- Register the new `"gemini"` driver inside the `drivers` map in `New()`:
  ```go
  m.drivers = map[string]Driver{
      "claude-code-cli":  &ClaudeCodeDriver{},
      "ollama":           &OllamaDriver{Instances: ollamaInstances},
      "claude-mediated":  hookDriver,
      "shell-stub":       &ShellStubDriver{},
      "gemini":           &GeminiDriver{}, // NEW
  }
  ```

#### [MODIFY] [config.go](file:///Users/keith/Code/kaos-control/internal/config/config.go)

- In `validateProject`, add validation for the `gemini` driver:
  ```go
  if a.Driver == "gemini" {
      if a.Model == "" {
          return fmt.Errorf("project config: agent %q has driver=gemini but missing model", a.Name)
      }
  }
  ```

---

### Web Frontend Components

We will update the frontend Vue files to add `gemini` as a valid driver option, display placeholder models, and display the "Gemini" badge.

#### [MODIFY] [api.ts](file:///Users/keith/Code/kaos-control/web/src/types/api.ts)

- Update type annotations for `AgentSummary` driver to include `'gemini'`.

#### [MODIFY] [AgentConfigForm.vue](file:///Users/keith/Code/kaos-control/web/src/components/agent/AgentConfigForm.vue)

- Add `'gemini'` to `driver` type union inside the form state and raw payload.
- Add `Gemini` as a radio button option under **Driver**.
- Render a dedicated Model input field for Gemini when `driver === 'gemini'` with a placeholder like `e.g. gemini-2.5-flash, gemini-1.5-pro`.
- Add basic validation for Gemini (ensure model name is filled in).

#### [MODIFY] [AgentPanelRow.vue](file:///Users/keith/Code/kaos-control/web/src/components/agent/AgentPanelRow.vue)

- Map `agent.driver === 'gemini'` to return `'Gemini'` for display.
- Add a custom CSS color badge for Gemini (e.g. HSL or deep indigo/blue theme).

#### [MODIFY] [AgentsRunsView.vue](file:///Users/keith/Code/kaos-control/web/src/views/project/AgentsRunsView.vue)

- Map `a.driver === 'gemini'` to return `'Gemini'` for display.
- Render the badge with CSS theme matching `AgentPanelRow.vue`.

---

## Verification Plan

### Automated Tests
- We will add unit tests to verify the Gemini stream decoding logic (specifically tests covering both complete/successful streams, and malformed/error responses).
- We can run Go tests using the terminal:
  `go test ./internal/agent/...`

### Manual Verification
- Define an agent using `driver: gemini` and `model: gemini-2.5-flash` in `lifecycle/config.yaml`.
- Launch an agent run via the Web UI and check:
  - Streaming updates rendering correctly in real-time.
  - Final full log is recorded correctly to `.log` file.
  - The UI successfully displays the Gemini badge.
