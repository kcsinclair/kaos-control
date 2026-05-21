# Implementation Plan - Gemini CLI Agent Driver Support

This plan outlines the design and implementation for adding support for the Google Gemini agent runner using the Antigravity CLI tool (`agy`). This runner will be exposed as a first-class driver called `gemini-cli` in `kaos-control`.

## User Review Required

> [!IMPORTANT]
> The `gemini-cli` driver executes `agy --dangerously-skip-permissions --prompt "<prompt>"` in the background. Because it relies on the `agy` CLI tool being globally or locally available, the system will look for `agy` on the system `PATH` by default, but also allows developers to customize the binary path.
> 
> The progress of the CLI tool will stream in real-time to the frontend live logs by mapping all non-JSON stdout lines to structured `ProgressEvent`s with `"type": "output"`.

## Proposed Changes

We will introduce a new driver called `gemini-cli` across the backend and frontend components.

---

### Backend Driver Implementation

#### [NEW] [gemini_cli.go](file:///Users/keith/Code/kaos-control/internal/agent/gemini_cli.go)
- Define `GeminiCliDriver struct` conforming to the `Driver` interface.
- Implement `buildArgs(run Run)` returning `["--dangerously-skip-permissions", "--prompt", run.PromptText]`.
- Implement `Start(ctx context.Context, run Run) (Process, error)`:
  - Default `BinaryPath` to `"agy"` if empty.
  - Spawn the subprocess with piped `StdoutPipe` and `StderrPipe`.
  - Tee stdout and stderr to the per-run log file at `run.LogPath` if configured.
  - Broadcast a `"started"` progress event initially.
  - Scan stdout line-by-line:
    - Attempt to unmarshal line as JSON.
    - If valid JSON, propagate it.
    - If not valid JSON, wrap it as a `ProgressEvent` with `"type": "output"` and `"text": line + "\n"`.
  - Pipe stderr into the standard ring buffer and log file.

#### [NEW] [gemini_cli_test.go](file:///Users/keith/Code/kaos-control/internal/agent/gemini_cli_test.go)
- Implement unit tests for `GeminiCliDriver`.
- Use a re-executing helper process pattern using `os.Args[0]` to safely mock the execution of the CLI without relying on any external system binary.
- Verify argument generation, stream progress scanning, non-JSON conversion, and environment variables.

#### [MODIFY] [agent.go](file:///Users/keith/Code/kaos-control/internal/agent/agent.go)
- Register the new driver under `"gemini-cli"` in the `Manager.drivers` registry map inside `NewManager`:
  ```go
  "gemini-cli": &GeminiCliDriver{},
  ```

---

### Frontend UI Integration

#### [MODIFY] [api.ts](file:///Users/keith/Code/kaos-control/web/src/types/api.ts)
- Update the JSDoc driver comments on `AgentSummary` to list `'gemini-cli'`.

#### [MODIFY] [AgentConfigForm.vue](file:///Users/keith/Code/kaos-control/web/src/components/agent/AgentConfigForm.vue)
- Expand the `driver` union type to include `'gemini-cli'`.
- Render a new radio button choice for "Gemini CLI (agy)":
  ```html
  <label class="acf-radio-label">
    <input v-model="driver" type="radio" value="gemini-cli" />
    Gemini CLI (agy)
  </label>
  ```
- Because `agy` uses default/configured models and does not accept a `--model` CLI argument, we do not require or display a "Model" text input when `driver === 'gemini-cli'`.

#### [MODIFY] [AgentPanelRow.vue](file:///Users/keith/Code/kaos-control/web/src/components/agent/AgentPanelRow.vue)
- Update `driverLabel(agent)` to return `'Gemini CLI'` when `driver === 'gemini-cli'`.
- Add gorgeous styling for the badge in the panel list using a premium Teal palette:
  ```css
  .panel-driver[data-driver="gemini-cli"] {
    background: #ccfbf1;
    color: #0f766e;
  }
  ```

#### [MODIFY] [AgentsRunsView.vue](file:///Users/keith/Code/kaos-control/web/src/views/project/AgentsRunsView.vue)
- Update `agentDriver()` to return `'Gemini CLI'` when `driver === 'gemini-cli'`.
- Add premium Teal badge styling in the active runs table:
  ```css
  .driver-badge[data-driver="gemini-cli"] {
    background: #ccfbf1;
    color: #0f766e;
  }
  ```

---

## Verification Plan

### Automated Tests
- Run Go unit tests:
  ```bash
  go test -v ./internal/agent/...
  ```
- Run Vue Typecheck:
  ```bash
  npm run type-check
  ```

### Manual Verification
- Launch `kaos-control` web interface.
- Open/Create an agent, select "Gemini CLI (agy)" as the driver, and save.
- Run a job with the Gemini CLI agent and observe the live log output streaming raw line-by-line CLI progress to the progress panel.
