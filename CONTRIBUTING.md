# Contributing to kaos-control

Thanks for taking an interest. This project is **AGPLv3** licensed — see
[LICENSE](LICENSE). AGPL was chosen because it is OSI-approved open source
*and* its network-use clause is the specific lever that prevents a cloud
provider from rehosting kaos-control as a paid service without publishing
their modifications — the SaaS loophole that GPLv3, MIT, and Apache 2 all
leave open. As the sole copyright holder the maintainer can also offer a
commercial licence to organisations that can't accept AGPL terms; the DCO
sign-off model below keeps that relicensing option intact without requiring
a CLA from contributors.

## Developer Certificate of Origin (DCO)

Every commit must be **signed off** to certify that you wrote the patch (or
otherwise have the right to submit it under the project's licence). This is
the same lightweight sign-off used by the Linux kernel and many other large
projects — no contributor licence agreement (CLA) to print, sign, and post.

Sign off your commits with the `-s` flag:

```sh
git commit -s -m "feat(graph): add priority filter to 2D view"
```

This appends a line like the following to your commit message:

```
Signed-off-by: Your Name <your.email@example.com>
```

By signing off, you are certifying the four points of the
[Developer Certificate of Origin](https://developercertificate.org/) — in
plain English: *"I wrote this, or I have the right to contribute it, and I
understand it will be redistributed under the project's licence."*

If you forget to sign off, amend the most recent commit with:

```sh
git commit --amend --signoff
```

For older commits, you can rebase and sign off in bulk:

```sh
git rebase --signoff main
```

Pull requests with unsigned commits will be asked to add the sign-off before
they can be merged.

## How to contribute

1. **Open an issue first** for anything non-trivial. The maintainer is happy
   to discuss approach and direction before you write code — saves rework.
   Small fixes (typos, obvious bugs) can go straight to a PR.
2. **Fork the repo** and create a feature branch from `main`.
3. **Make focused commits** — one logical change per commit, present-tense
   commit subjects (`add foo`, not `added foo`), with the DCO sign-off line.
4. **Run the tests** — `make test-unit` for backend, `pnpm --dir tests/web
   test` for frontend.
5. **Run the linter** — `make lint`. Both `go vet` and `staticcheck` should
   pass clean.
6. **Open a pull request** against `main`. Describe what changed and why; if
   the change relates to a lifecycle artifact (`lifecycle/...`), reference
   it in the PR description.

## Coding conventions

- **Backend (Go)**: standard `go fmt` + `staticcheck` clean. Package layout
  follows the existing `internal/<area>/` pattern.
- **Frontend (Vue/TS)**: existing `tsconfig` strict settings; component-level
  unit tests live in `tests/web/` (Vitest + happy-dom).
- **Commit messages**: see existing `git log` for style. Common prefixes:
  `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `build:`, `agent(<name>):`.
- **Lifecycle artifacts**: if you're adding a feature that needs a plan or
  test, write the plan/test artifact under `lifecycle/` and link it from the
  PR. The project's own development uses the same lifecycle it enforces.

## Licence headers on new source files

Every Go, TypeScript, and Vue source file in this repository carries an
[SPDX](https://spdx.dev/) licence-identifier comment near the top:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package foo
```

```ts
// SPDX-License-Identifier: AGPL-3.0-or-later

import { …
```

```vue
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
…
</script>
```

When you add a new `.go`, `.ts`, or `.vue` file, please include the matching
SPDX header. The CI lint pass (once enabled) will check for it; reviewers
will ask you to add it before merge. No copyright line is needed — your
ownership of your contribution is recorded by `git log` and your DCO
sign-off (see [NOTICE](NOTICE)).

For Go files with `//go:build` constraints, place the SPDX line *above* the
build tag:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration
```

## What you are agreeing to by contributing

- Your contribution will be distributed under the project's **AGPLv3**
  licence.
- You retain copyright in your contribution; there is no copyright
  assignment.
- The maintainer may also offer the project under a separate commercial
  licence to organisations that cannot accept AGPL terms. Because there is
  no copyright assignment, *your* contribution can only be relicensed with
  your permission — but the maintainer's *original* code (and any code
  contributed under a future explicit relicensing agreement) may be offered
  under both licences.

If any of that gives you pause, please raise it on the PR or in an issue
before merging — better to discuss than to land code under terms a
contributor doesn't understand.

## Reporting a security issue

Please **do not** file public issues for security vulnerabilities. Email the
maintainer directly (see the repository contact / git log) so a fix can be
prepared before disclosure.
