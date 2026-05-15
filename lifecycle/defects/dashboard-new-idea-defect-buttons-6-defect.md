---
title: Dashboard header btn-new-idea precedes btn-new-defect, violating FR-4 DOM order
type: defect
status: draft
lineage: dashboard-new-idea-defect-buttons
parent: lifecycle/tests/test-dashboard-new-idea-defect-buttons-e2e.md
labels:
  - defect
assignees:
  - role: frontend-developer
    who: agent
---

# Dashboard header btn-new-idea precedes btn-new-defect, violating FR-4 DOM order

## Reproduction Steps

1. Open `web/src/views/project/DashboardView.vue` and inspect the `.header-actions` div (around line 47).
2. Note that the `<button class="btn-new-idea">` element appears at line 48, before `<button class="btn-new-defect">` at line 55.
3. Run the Vitest suite: `cd tests/web && pnpm exec vitest run dashboard-new-idea-defect-buttons`
4. Observe test M1-TC3 fail with:
   ```
   AssertionError: expected 783 to be less than 200
   ```
   (btn-new-defect's position in the HTML string is 783; btn-new-idea's is 200 — meaning idea comes first.)

## Expected Behaviour

Per FR-4, "New Defect" must appear **left** of "New Idea" in the dashboard header. In DOM terms `btn-new-defect` must precede `btn-new-idea` inside `.header-actions`.

```html
<!-- correct order -->
<div class="header-actions">
  <button class="btn-new-defect">…</button>
  <button class="btn-new-idea">…</button>
  …
</div>
```

## Actual Behaviour

The template has `btn-new-idea` first and `btn-new-defect` second:

```html
<!-- actual order (wrong) -->
<div class="header-actions">
  <button class="btn-new-idea">…</button>
  <button class="btn-new-defect">…</button>
  …
</div>
```

This means "New Idea" is rendered on the left and "New Defect" on the right, which is the reverse of FR-4.

## Logs / Output

```
❯ dashboard-new-idea-defect-buttons.test.ts  (22 tests | 1 failed) 109ms
   ❯ DashboardView — button presence and layout (M1) > M1-TC3: .btn-new-defect precedes .btn-new-idea in DOM order (FR-4: Defect left, Idea right)
     → expected 783 to be less than 200
 FAIL  dashboard-new-idea-defect-buttons.test.ts > DashboardView — button presence and layout (M1) > M1-TC3: .btn-new-defect precedes .btn-new-idea in DOM order (FR-4: Defect left, Idea right)
AssertionError: expected 783 to be less than 200
 ❯ dashboard-new-idea-defect-buttons.test.ts:224:44
```

All other 21 tests in the suite pass.

## Fix guidance

In `web/src/views/project/DashboardView.vue`, swap the order of the two buttons inside `.header-actions` so that `btn-new-defect` comes before `btn-new-idea`. No logic changes are required — only template reordering.
