# Manual Verification: 3D Graph Edge Contrast

Test plan reference: `lifecycle/test-plans/3d-graph-edge-contrast-5-test.md` — Milestone 4

Perform these steps on a running dev server (`make run`) with the feature branch checked out.

---

## Prerequisites

- A project registered in kaos-control with **≥ 20 artifacts** spanning multiple lineages.
- At least one artifact of each lifecycle stage so that **parent**, **depends_on**, **related_to**, **timeline**, and **assigned** edges all appear in the graph.
- Chrome or Firefox with DevTools available.

---

## Step 1 — Load the 3D graph in dark theme

1. Open the project in kaos-control and navigate to the **Map** view.
2. Confirm the dark theme is active (navy/slate canvas background `#0f172a`).
3. Click the **3D** tab (or equivalent control) to switch to the ForceGraph3D view.

**Expected:** Graph renders without console errors. Canvas background is dark navy.

---

## Step 2 — Verify edge visibility in dark theme

1. Look at edges connecting nodes. Pan and rotate if needed.
2. Confirm **all** edge kinds are clearly distinguishable against the dark background — no edge should appear invisible or extremely faint.

**Expected:**
- `parent` / `related_to` edges (slate-400 `#94a3b8`): light grey, clearly visible.
- `depends_on` edges (orange-500 `#f97316`): warm orange, clearly visible.
- `blocks` edges (red-500 `#ef4444`): red, clearly visible.
- `timeline` edges (blue-500 `#3b82f6`): bright blue, clearly visible.
- `assigned` edges (slate-600 `#475569`): subdued but visible — not invisible.

---

## Step 3 — Verify edge visibility in light theme

1. Toggle to light theme (sun icon in the header).
2. Confirm canvas background changes to white `#ffffff`.
3. Repeat visual inspection of edge kinds.

**Expected:**
- `parent` / `related_to` edges (slate-500 `#64748b`): medium grey, clearly visible.
- `depends_on` edges (orange-600 `#ea580c`): orange, clearly visible.
- `blocks` edges (red-600 `#dc2626`): red, clearly visible.
- `timeline` edges (blue-600 `#2563eb`): blue, clearly visible.
- `assigned` edges (slate-500 `#64748b`): subdued but visible — not washed out.

---

## Step 4 — Verify timeline edge prominence (dark theme)

1. Switch back to dark theme.
2. Locate a `timeline` edge in the graph.

**Expected:** Timeline edges are the **thickest** and **most prominent** edges visible — noticeably thicker than `parent`/`depends_on`/`related_to` edges.

---

## Step 5 — Verify assigned edge is least prominent (dark theme)

1. Locate an `assigned` edge (artifact → release links).

**Expected:** Assigned edges are the **thinnest** and **least prominent** — visually subordinate to all other edge kinds, but still clearly visible (not invisible).

---

## Step 6 — Theme toggle — edges update immediately

1. While looking at the 3D graph, toggle between dark and light themes rapidly (3–4 times).

**Expected:**
- Edge colours update immediately after each toggle.
- No page reload required.
- No layout jitter — node positions are preserved during the theme change.
- No blank-canvas flash or WebGL error in the console.

---

## Step 7 — Verify no node regression

1. In both themes, inspect several nodes.

**Expected:**
- Node colours, sizes, and label rendering are unchanged from the pre-feature baseline.
- Priority rings (if present) are not affected.
- Approved-test blue torus rings (if present) are not affected.

---

## Step 8 — Verify force simulation stability during theme toggle

1. Let the graph settle (force simulation stops).
2. Toggle the theme.

**Expected:**
- Node positions do not change after theme toggle.
- Force simulation does not restart (nodes do not scatter and re-settle).

---

## Pass / Fail Criteria

| Step | Description                                    | Result |
|------|------------------------------------------------|--------|
| 1    | Graph loads in dark theme without errors       | Pass / Fail |
| 2    | All edges visible in dark theme                | Pass / Fail |
| 3    | All edges visible in light theme               | Pass / Fail |
| 4    | Timeline edges visually most prominent         | Pass / Fail |
| 5    | Assigned edges visually least prominent        | Pass / Fail |
| 6    | Theme toggle updates edges without reload/jitter | Pass / Fail |
| 7    | No regression on node colours / sizes / labels | Pass / Fail |
| 8    | Force simulation does not restart on toggle    | Pass / Fail |

All 8 steps must pass before the QA milestone is signed off.

---

## Notes

- Non-timeline edges intentionally render at ~75 % opacity (alpha `bf` in the hex colour) to reduce clutter in dense graphs. A slight transparency is expected and correct.
- Timeline edge labels (duration text) should appear above the edge in the correct palette text colour.
- If any edge appears completely invisible at the default zoom, that is a fail for steps 2 or 3.
