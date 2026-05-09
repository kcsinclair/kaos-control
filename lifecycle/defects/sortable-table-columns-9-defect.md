---
title: ParseErrorsView.sort.test.ts fails to collect — vi.mock factory references top-level variable before initialization
type: defect
status: done
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
release: KC-Feature-Sprint
---

# ParseErrorsView.sort.test.ts fails to collect — vi.mock factory references top-level variable before initialization

## Reproduction Steps

1. Run `cd tests/web && pnpm test ParseErrorsView.sort --reporter=verbose`.
2. Observe the entire test file fails to collect — no individual tests run.
3. The error is a Vitest hoisting error from line 4 of the test file:
   ```
   Error: [vitest] There was an error when mocking a module. ...
   Caused by: ReferenceError: Cannot access 'mockErrors' before initialization
   ```
4. In `tests/web/ParseErrorsView.sort.test.ts`, line 28 declares a module-level variable:
   ```ts
   const mockErrors: ParseErrorRow[] = []
   ```
5. Lines 30–34 pass this variable inside a `vi.mock` factory:
   ```ts
   vi.mock('@/api/client', () => ({
     api: {
       get: vi.fn().mockResolvedValue({ errors: mockErrors }),
     },
   }))
   ```
6. Vitest hoists all `vi.mock(...)` calls to the top of the file at transform time, before `const mockErrors` is declared. The factory therefore runs before `mockErrors` exists and throws a `ReferenceError`.

## Expected Behaviour

All 10 test scenarios in `ParseErrorsView.sort.test.ts` (Milestone 4) should collect and run. The `@/api/client` mock should resolve GET calls with configurable error fixtures so each test can populate the view with known data.

## Actual Behaviour

The test file fails entirely during collection. Zero tests run:

```
 FAIL  ParseErrorsView.sort.test.ts [ ParseErrorsView.sort.test.ts ]
Error: [vitest] There was an error when mocking a module. If you are using "vi.mock" factory,
make sure there are no top level variables inside, since this call is hoisted to top of the file.
 ❯ ../../web/src/views/project/ParseErrorsView.vue:4:31
Caused by: ReferenceError: Cannot access 'mockErrors' before initialization
 ❯ ParseErrorsView.sort.test.ts:4:46

 Test Files  1 failed (1)
       Tests  no tests
```

## Logs / Output

Full error trace:

```
❯ ../../web/src/views/project/ParseErrorsView.vue:4:31
      2| import { computed, ref, onMounted, watch } from 'vue'
      3| import { useRoute } from 'vue-router'
      4| import { api } from '@/api/client'
         |                               ^
Caused by: ReferenceError: Cannot access 'mockErrors' before initialization
  ❯ ParseErrorsView.sort.test.ts:4:46
```

Fix: replace the top-level `const mockErrors` reference inside the `vi.mock` factory with a `vi.fn()` that can be configured per-test using `mockResolvedValue` / `mockImplementation`, or use `vi.hoisted()` to declare the variable in a way that survives hoisting. Example:

```ts
// Option A — vi.hoisted
const mockErrors = vi.hoisted(() => vi.fn().mockResolvedValue({ errors: [] }))

vi.mock('@/api/client', () => ({
  api: { get: mockErrors },
}))

// Then in each test:
mockErrors.mockResolvedValue({ errors: makeErrors() })
```
