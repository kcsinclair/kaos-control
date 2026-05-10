---
title: "Backend Plan: Roadmap Gantt Period Display Options"
type: plan-backend
status: in-development
lineage: roadmap-gantt-period-options
parent: lifecycle/requirements/roadmap-gantt-period-options-2.md
created: "2026-05-10T00:00:00+10:00"
labels:
    - roadmaps
    - enhancement
release: KC-Release0
assignees:
    - role: backend-developer
      who: agent
---

# Backend Plan: Roadmap Gantt Period Display Options

This feature is primarily frontend-driven — all release data is already served by
the existing REST API and the frontend computes the Gantt time axis client-side.
The backend scope is limited to:

1. Adding a configurable default period mode to the per-project configuration
   (`lifecycle/config.yaml`) so that product owners can set the out-of-the-box
   experience without code changes.
2. Exposing that configuration value through the existing project-config API
   endpoint so the frontend can read it on load.

Related plans: [[roadmap-gantt-period-options]] (frontend plan handles all UI
logic; test plan covers integration verification).

---

## Milestone 1: Add `RoadmapConfig` to the project configuration struct

### Description

Extend the `Project` struct in `internal/config/config.go` with a new
`RoadmapConfig` section that holds a `DefaultPeriodMode` field. This field
accepts `"autoscale"` (default), `"month"`, `"quarter"`, `"half-year"`, or
`"year"`. When set to one of the fixed-period values, the frontend will
initialise in fixed-period mode with that window; when set to `"autoscale"`,
the frontend starts in autoscale mode.

### Files to change

- `internal/config/config.go` — add `RoadmapConfig` struct and field on `Project`; update `defaultProject()` to set `DefaultPeriodMode: "autoscale"`.

### Acceptance criteria

- [ ] `RoadmapConfig` struct exists with `DefaultPeriodMode string` field, tagged `yaml:"default_period_mode" json:"default_period_mode"`.
- [ ] `Project` struct has `Roadmap RoadmapConfig` field tagged `yaml:"roadmap,omitempty" json:"roadmap,omitempty"`.
- [ ] `defaultProject()` sets `Roadmap.DefaultPeriodMode` to `"autoscale"`.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 2: Validate the `default_period_mode` value

### Description

Add validation in the `Project.validate()` method (or equivalent validation
path called by `LoadProject`) to reject unknown values for
`default_period_mode`. Accepted values: `"autoscale"`, `"month"`, `"quarter"`,
`"half-year"`, `"year"`, or empty string (treated as `"autoscale"`).

### Files to change

- `internal/config/config.go` — add validation logic in the existing `validate` function (around line 385) after the existing stage/agent checks.

### Acceptance criteria

- [ ] Loading a config with `roadmap.default_period_mode: "quarter"` succeeds.
- [ ] Loading a config with `roadmap.default_period_mode: "weekly"` returns a descriptive error.
- [ ] Empty / omitted `default_period_mode` defaults to `"autoscale"` without error.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 3: Expose roadmap config via the project-config API

### Description

The existing project-config endpoint (serves the parsed `Project` struct as
JSON) already serialises the full struct. Because the new `Roadmap` field is
tagged with `json:"roadmap,omitempty"`, it will automatically appear in the
API response once the struct is updated — no handler changes needed. This
milestone verifies that the field is present in the response and documents the
contract for the [[roadmap-gantt-period-options]] frontend plan.

### Files to change

- No code changes expected. If the project-config handler cherry-picks fields
  instead of serialising the full struct, add `Roadmap` to the response type.
  Check `internal/http/` for the handler.

### Acceptance criteria

- [ ] `GET /api/p/{project}/config` (or equivalent endpoint) includes `"roadmap": {"default_period_mode": "..."}` in the JSON response.
- [ ] When `lifecycle/config.yaml` omits the `roadmap` section entirely, the endpoint returns the default value (`"autoscale"`).
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 4: Unit tests for config loading and validation

### Description

Add unit tests in `internal/config/config_test.go` covering the new roadmap
configuration parsing and validation. Follow the existing test patterns in
that file (e.g., `writeMinimalProjectConfig` helper).

### Files to change

- `internal/config/config_test.go` — add test cases for valid values, invalid values, omitted section, and default fallback.

### Acceptance criteria

- [ ] Test: valid `default_period_mode` values (`"autoscale"`, `"month"`, `"quarter"`, `"half-year"`, `"year"`) parse without error.
- [ ] Test: invalid value (e.g., `"weekly"`) returns error containing a descriptive message.
- [ ] Test: omitted `roadmap` section results in `DefaultPeriodMode == "autoscale"`.
- [ ] All tests pass: `go test ./internal/config/ -run Roadmap -v`.
