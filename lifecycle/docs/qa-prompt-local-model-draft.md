---
title: "Draft: qa agent prompt tuned for local models"
type: doc
status: draft
lineage: qa-prompt-local-model-draft
created: "2026-08-27T10:30:00+10:00"
priority: normal
labels:
    - doc
    - draft
    - agent
    - qa
    - local-models
release: KC-Release6
---

# Draft: qa agent prompt tuned for local models

**For review — not yet applied to `lifecycle/config.yaml`.**

## Why the current prompt fails on a local model

Run `2b5f167167c854df` (qa, gemma-4-26B) hit the 25-iteration cap having made
**10 `list_dir`, 12 `read_file`, 3 `grep`, and zero writes**. Two causes:

1. **It could not do the job at all.** The prompt says "run the relevant
   integration tests", but the driver had no shell tool. It spent the run
   hunting for a way to execute tests that did not exist. *(Now fixed — bash is
   available opt-in via `bash_allowlist`.)*
2. **"read `lifecycle/architecture/`" is a discovery task, not a read.** The
   agent must list the root, list `standards/`, list `decisions/`, then read ~13
   files — roughly **16 of its 25 iterations** before it starts testing.

The architecture obligation is a **quality gate and stays**. What changes is
that the files are *named* (no hunting) and read *when they are needed* (only
when there are failures to triage — which the current prompt already implies
with "Before triaging failures").

Expected cost: **0 architecture reads when tests pass**, ~6–8 targeted reads
when triaging, versus ~16 exploratory calls today.

## Draft prompt

```yaml
  - name: qa
    role:
      - qa
    provider: leia-llamacpp
    driver: openai-compatible
    model: gemma-4-26B-A4B-it-UD-Q8_K_XL
    active_status: in-qa
    source_types:
      - test
    timeout_minutes: 0
    max_tool_iterations: 40          # was 25; running tests + triage needs more
    bash_allowlist:                  # enables the bash tool (opt-in)
      - "make test-unit"
      - "make test-integration"
      - "make lint"
      - "cd tests/web && pnpm test"
    allowed_write_paths:
      - lifecycle/defects
      - lifecycle/architecture/decisions
    prompt_templates:
      qa: |
        You are a QA agent for the Innovation Maker project.
        Your target artifact is {target_path}.

        Work through these steps IN ORDER. Do not explore the repository.

        STEP 1 — Run the tests.
        Run these commands with the bash tool, one at a time, and keep the
        output of each:
          make test-unit
          make test-integration
        Only these commands are permitted; do not try others.

        STEP 2 — If every test passed:
        Write nothing. Report "All tests passed" and stop. Do not read any
        further files.

        STEP 3 — Only if one or more tests FAILED, read these exact files
        before triaging (do not list directories, read them by path):
          lifecycle/architecture/architecture-summary.md
          lifecycle/architecture/standards/index-is-a-cache.md
          lifecycle/architecture/standards/secrets-handling.md
          lifecycle/architecture/standards/filesystem-sandboxing.md
        Then list lifecycle/architecture/decisions/ and read ONLY the ADRs
        whose filename relates to the failing area.

        STEP 4 — For EACH failing test, write ONE defect file in
        lifecycle/defects/<short-slug>.md with this frontmatter:
          title: <short defect title>
          type: defect
          status: draft
          lineage: <lineage of the feature being tested>
          parent: {target_path}
          labels: [defect]
          created: <current date-time, RFC3339>
          assignees:
            - role: <backend-developer | frontend-developer | test-developer>
              who: agent

        Choose the role by which layer failed:
          Backend API / logic / data -> backend-developer
          UI / component / state     -> frontend-developer
          Flaky or wrong test        -> test-developer

        Body sections, in this order:
          ## Reproduction Steps      (exact, numbered, including the command)
          ## Expected Behaviour
          ## Actual Behaviour
          ## Logs / Output           (paste the real failing output)

        Judge each failure against what you read in STEP 3. If a failure is a
        deliberate, defensible deviation from the recorded architecture or
        standards rather than a defect, do NOT file a defect and do NOT silently
        accept it: write an ADR in lifecycle/architecture/decisions/
        (type: adr) capturing the decision, its context, and its consequences.

        Write files. Do not summarise what you would write.
```

## What changed, and why

| Change | Reason |
|---|---|
| Numbered steps, "do not explore" | Weak models wander without an explicit order of operations |
| Exact commands, matched to `bash_allowlist` | Removes the "how do I run tests?" hunt that consumed the run |
| Architecture read moved to STEP 3 (failures only) | Costs **zero** iterations on a green run; obligation preserved |
| Architecture files named by full path | Removes 3 `list_dir` calls plus blind reads |
| ADRs read selectively by filename relevance | 6 ADRs today and growing; reading all of them will not scale |
| `max_tool_iterations: 40` | Running 2 suites + reading 4 files + writing N defects legitimately exceeds 25 |
| "Write files. Do not summarise." | Local models often describe the file instead of writing it |

## Open points for review

1. **Test commands** — are `make test-unit` / `make test-integration` the right
   pair for a qa run, or should the frontend suite (`cd tests/web && pnpm test`)
   be included by default? It is in the allowlist but not in STEP 1.
2. **Allowlist breadth** — entries are matched as glob patterns against the
   command string, so `make test-*` would also permit `make test-unit; curl …`.
   The draft uses exact commands instead of wildcards for that reason.
3. **Selective ADR reading** — asking a weak model to judge "which ADRs are
   relevant" may be unreliable. The alternative is reading all 6 (costly, and
   worsens as ADRs accumulate). A middle option is to name the 2–3 ADRs that
   most often bear on test failures.
4. **`max_tool_iterations: 40`** is an estimate, not measured. Worth checking
   against a real green run and a real 2-failure run.
