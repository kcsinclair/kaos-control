---
title: Static HTML / CSS / JS (No-Framework Frontend)
type: tech-stack
status: draft
lineage: stack-static-html-js
labels:
    - tech-stack
    - catalog
    - frontend
    - static
    - html
    - javascript
    - low-complexity
summary: Hand-authored static HTML/CSS/vanilla-JS with no framework build step — the simplest possible web front-end. kaos-control's marketing site (kaos-control.io) uses this shape.
---

# Static HTML / CSS / JS (No-Framework Frontend)

**Focus:** simplicity. The least machinery that puts a good site online — plain files, no framework, no bundler required.

## Overview
A site authored directly as HTML, CSS, and vanilla JavaScript. There is no framework and no compile step to speak of — the source *is* (or nearly is) the deployed output. Optional TypeScript can be used purely for type-checking authoring scripts without changing the runtime. Ideal for landing pages, marketing sites, and simple single- or multi-page apps that don't justify a framework. *(This is `kaos-control.io`'s stack: hand-authored pages under `htdocs/`, which is itself the build output — the `build` step is a no-op.)*

## Communication layer
Served as **static files** from any web host or CDN. Any dynamic behaviour is **client-side** `fetch()` to a separate REST API or serverless function; there is no server on the request path.

## Data persistence
**None server-side.** Content is embedded in the pages, or read client-side from a remote API. Small amounts of state can live in `localStorage`/`IndexedDB`.

## Quality tooling
Because there's no framework to lean on, lean on checks instead: **html-validate** for markup, **Playwright** for end-to-end/browser tests, **@axe-core/playwright** for accessibility, and optional **`tsc --noEmit`** for typed authoring scripts. (This is exactly the kaos-control.io test setup.)

## Profile
| Trait | Value |
| --- | --- |
| Layer | Frontend (static) |
| Languages | HTML, CSS, JS (optional TS for typecheck) |
| Footprint | Minimal — no runtime, no server |
| Learning curve | Low |
| Best use case | Marketing sites, landing pages, simple SPAs |

## Suits these architectures
- [Static Website / JAMstack](../architectures/static-site.md) — its natural home; static files on a CDN.

## When to reach for something else
- Content-heavy site with many pages / a blog → [Hugo](hugo.md) generates and templates it for you.
- Needs request-time server logic, auth, or writes → a full stack like [Go + Vue](go-vue.md) or [Simple PHP](php-simple.md).

## Stack profile

kaos-control reads this profile to configure the coding agents for this stack (repo layout, per-role write paths, and build/lint/test/run commands); it also doubles as a plain-language primer on how the stack is laid out and run. A static site has **no backend role**; the
frontend role owns the whole site. `write_paths` are stack source roots only.

```yaml
stack_profile:
  run: npx serve htdocs              # or: python3 -m http.server -d htdocs
  repo_layout:
    - {path: htdocs/,         note: hand-authored static site (this IS the deploy output)}
    - {path: website-assets/, note: images, CSS, JS}
    - {path: tests/,          note: Playwright + axe accessibility tests}
  roles:
    backend-developer:
      required: false           # static site — no server-side code
    frontend-developer:
      write_paths: [htdocs, website-assets]
      build: ""                 # none — htdocs/ is the deployable output
      lint: npx html-validate htdocs
      test: pnpm test           # playwright + axe
    test-developer:
      write_paths: [tests]
      test: pnpm test
```
