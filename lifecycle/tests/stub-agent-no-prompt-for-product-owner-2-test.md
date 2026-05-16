---
title: "Test: Stub Agent Prompt Template for product-owner Role"
type: test
status: draft
lineage: stub-agent-no-prompt-for-product-owner
parent: lifecycle/defects/stub-agent-no-prompt-for-product-owner.md
---

# Test: Stub Agent Prompt Template for product-owner Role

Integration tests verifying that a shell-stub agent with a `product-owner`
`prompt_templates` entry correctly accepts run requests (HTTP 202), and that an
agent missing that entry returns HTTP 409 with error code `run_error`.

Relates to defect [[stub-agent-no-prompt-for-product-owner]]: TC3 and TC4 of
`flows/10-artefact-run-count-column.spec.ts` were failing because `stub-agent`
had no `product-owner` prompt template, causing every run against `rc-pill.md`
and `rc-ws.md` to return 409.

## Scenarios Covered

### `tests/integration/stub_agent_prompt_template_test.go`

**`TestStubAgent_PromptTemplate_ProductOwnerPresent_Succeeds` (TC1)**

- Shell-stub agent configured with `role: [product-owner]` and
  `prompt_templates.product-owner` set.
- POST `/agents/stub-with-po-tpl/run` with no `role` in the request body.
- The server falls back to `Roles[0]` = `"product-owner"`, finds the template,
  and returns HTTP 202 with a `run_id`.
- The run completes with `status: "done"`.

**`TestStubAgent_PromptTemplate_ProductOwnerMissing_Returns409` (TC2)**

- Shell-stub agent configured with `role: [product-owner]` but
  `prompt_templates` only contains an `"analyst"` entry (no `product-owner`).
- POST `/agents/stub-missing-po-tpl/run` with no `role` in the request body.
- The server attempts to look up `prompt_templates["product-owner"]`, fails,
  and returns HTTP 409 with `{"error":{"code":"run_error","message":"..."}}`.

**`TestStubAgent_PromptTemplate_ExplicitProductOwnerRole_Succeeds` (TC3)**

- Same agent as TC1 but with `"role": "product-owner"` explicitly in the
  request body.
- Verifies the explicit-role path in `StartRun` alongside the implicit
  fallback covered by TC1.
- Returns HTTP 202 and run completes with `status: "done"`.

**`TestStubAgent_PromptTemplate_ExplicitRoleMissingTemplate_Returns409` (TC4)**

- Same agent as TC1 (has `product-owner` template only), but request body
  sends `"role": "qa"` — a valid project role that has no template.
- Returns HTTP 409 with `{"error":{"code":"run_error","message":"..."}}`.

## Test File

| File | Type | Command |
|------|------|---------|
| `tests/integration/stub_agent_prompt_template_test.go` | Go integration | `go test -tags integration ./tests/... -run TestStubAgent_PromptTemplate` |
