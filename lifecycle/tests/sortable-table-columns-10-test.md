---
title: ParseErrorsView sort tests — vi.mock hoisting fix
type: test
status: approved
lineage: sortable-table-columns
parent: lifecycle/defects/sortable-table-columns-9-defect.md
---

# ParseErrorsView sort tests — vi.mock hoisting fix

Fixes the broken test file `tests/web/ParseErrorsView.sort.test.ts` so that
all 10 Milestone 4 scenarios collect and run. The file previously failed to
collect because a `vi.mock` factory referenced a top-level `const` variable
(`mockErrors`) that Vitest's hoisting had not yet initialised.

## Change made

`tests/web/ParseErrorsView.sort.test.ts` — no new scenarios; the fix makes the
existing scenarios runnable.

### Root cause

Vitest hoists all `vi.mock(...)` calls to the top of the transformed module,
before any `const`/`let`/`var` declarations in the source file. The original
code passed `mockErrors` (a module-level array) directly inside the factory,
which produced a `ReferenceError: Cannot access 'mockErrors' before
initialization`.

### Fix applied

1. Replaced the top-level `const mockErrors: ParseErrorRow[] = []` and its use
   inside the `vi.mock` factory with a `vi.hoisted()` call:

   ```ts
   const mockApiGet = vi.hoisted(() => vi.fn().mockResolvedValue({ errors: [] }))

   vi.mock('@/api/client', () => ({
     api: { get: mockApiGet },
   }))
   ```

   `vi.hoisted()` runs its callback inside the hoist boundary, so `mockApiGet`
   is available when the factory executes.

2. Simplified `injectErrors` from an `async` function that re-imported
   `@/api/client` into a plain synchronous helper that calls
   `mockApiGet.mockResolvedValue(...)` directly.

3. Removed `await` from all `injectErrors(...)` call sites (now synchronous).

4. Updated the "Reload after sort" test to call `mockApiGet.mockResolvedValue`
   directly instead of doing a dynamic import + `vi.mocked(api.get)`.

## Scenarios covered

All 10 scenarios from Milestone 4 of `lifecycle/tests/sortable-table-columns-6-test.md`:

| Scenario | Status |
|----------|--------|
| File asc — errors sorted by path ascending | covered |
| File desc — second click reverses | covered |
| File reset — third click restores original order | covered |
| Error asc — errors sorted by message ascending | covered |
| Error desc — second click reverses | covered |
| Three-state cycle (File) — asc → desc → reset via `aria-sort` | covered |
| Single active indicator | covered |
| Asc indicator visible after first click | covered |
| Desc indicator visible after second click; asc gone | covered |
| Reload after sort — button re-fetches without crash | covered |
