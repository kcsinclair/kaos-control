# Manual Verification: DevOps CLI — Frontend Regression

Test plan reference: `lifecycle/test-plans/kaos-control-devops-cli-5-test.md` — Milestone 6

Perform these steps on a running dev server (`make run`) with the `kc-dev` branch checked
out. The two frontend seams verified here are the project users view (`linux_user` display)
and the run history view attributing CLI-triggered runs to the resolved user.

---

## Prerequisites

- kaos-control server running (`make run`)
- A project registered with at least one user binding that has `linux_user:` set and one
  that does not
- A bearer token for a `product-owner` or `devops` user, stored in `$KAOS_CONTROL_TOKEN`
  or available as a `--token` argument

---

## Step 1 — Project users view: linux_user column

1. Open the project in kaos-control and navigate to **Settings → Users** (or the equivalent
   project users panel in the UI).
2. Confirm that for bindings with `linux_user:` set, the value is displayed alongside the
   email and roles.
3. Confirm that for bindings WITHOUT `linux_user:`, the column/field is absent or blank
   rather than showing an error or `undefined`.

**Expected:**
- Mapped bindings show `linux_user` value cleanly.
- Unmapped bindings render without errors or layout breakage.
- No browser console errors on the page.

---

## Step 2 — CLI-triggered run appears in RunHistory.vue

1. Trigger a pipeline run from the CLI:
   ```sh
   kaos-control devops run --project <project-name> [--token $KAOS_CONTROL_TOKEN] quick-pass
   ```
2. Open the project in kaos-control and navigate to the **DevOps → Pipelines** view.
3. Select the `quick-pass` pipeline and open its run history.
4. Confirm the CLI-triggered run appears in the list with a status of `passed`.
5. Click the run to open the log view and confirm the log streams to completion without
   errors.

**Expected:**
- The CLI-triggered run is visible in run history.
- Attributed user shown (if UI exposes attribution) is the resolved kaos-control email,
  not a raw Linux username.
- Log view renders the NDJSON output correctly (step names, output, completion status).
- No console errors; `pnpm build` remains clean.

---

## Step 3 — No console errors after both steps

Open the browser DevTools console and confirm:
- Zero errors after loading the users view.
- Zero errors after loading the run history and log view.
