---
title: Hugo (Static Site Generator)
type: tech-stack
status: approved
lineage: stack-hugo
labels:
    - tech-stack
    - catalog
    - frontend
    - static
    - ssg
    - go
    - low-complexity
summary: Markdown content compiled to a fast static site by Hugo (Go). Themes, layouts, and a build step — ideal for blogs, docs, and marketing sites. tek42.io uses this.
---

# Hugo (Static Site Generator)

**Focus:** content-first static sites. Author in Markdown, template with layouts, build to a fast static bundle.

## Overview
Hugo is a Go-based static site generator: content is written as **Markdown** with front matter under `content/`, rendered through **layouts** and a **theme**, and compiled into a static `public/` directory. Builds are famously fast, and the output is plain static files with no runtime. Best for sites with real content structure — blogs, documentation, multi-page marketing — where hand-authoring every page (see [Static HTML / CSS / JS](static-html-js.md)) would be tedious. *(This is `tek42.io`'s stack: `hugo.toml` config, a `hugo-bootstrap-theme`, content in `content/`, build/deploy shell scripts, output in `public/`.)*

## Communication layer
Served as **static files** from any web host or CDN — no server on the request path. Dynamic bits (forms, search, comments) are client-side JS against external services/APIs.

## Data persistence
**Build-time.** Content and structured data live in `content/` and `data/` (Markdown, TOML/YAML/JSON) and are compiled into pages. No request-time database.

## Quality tooling
A single **`hugo`** binary builds everything (no Node toolchain required). Pair with link-checking, HTML validation, and Playwright/axe end-to-end + accessibility checks against the built `public/` output, the same way a hand-authored static site is verified.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Frontend (static, generated) |
| Languages | Markdown + Go templates (theme); a little JS |
| Footprint | Minimal — one binary, static output |
| Learning curve | Low–moderate (templating + theme concepts) |
| Best use case | Blogs, documentation, content-rich marketing sites |

## Suits these architectures
- [Static Website / JAMstack](../architectures/static-site.md) — content compiled to static files on a CDN.

## When to reach for something else
- A handful of pages with no content model → hand-authored [Static HTML / CSS / JS](static-html-js.md) is simpler still.
- Needs request-time server logic, auth, or writes → a full stack like [Go + Vue](go-vue.md) or [Simple PHP](php-simple.md).

## Stack profile

kaos-control reads this profile to configure the coding agents for this stack (repo layout, per-role write paths, and build/lint/test/run commands); it also doubles as a plain-language primer on how the stack is laid out and run. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: hugo server -D
  repo_layout:
    - {path: content/,  note: Markdown content}
    - {path: layouts/,  note: template overrides}
    - {path: themes/,   note: theme(s)}
    - {path: static/,   note: static assets}
    - {path: hugo.toml, note: site config}
  roles:
    backend-developer:
      required: false          # static site generator — no server-side code
    frontend-developer:
      write_paths: [content, layouts, static, data, i18n]
      build: hugo --minify
      lint: hugo --printPathWarnings   # build-time template/path checks
      test: hugo --renderToMemory      # smoke: the site builds without error
    test-developer:
      write_paths: [tests]
      test: ""                         # add link-check / Playwright as needed
```
