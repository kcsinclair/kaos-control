---
title: Static Website / JAMstack
type: architecture
status: draft
lineage: arch-static-site
labels:
    - architecture
    - catalog
    - static
    - low-complexity
related_to:
    - architecture/tech-stacks/static-html-js.md
    - architecture/tech-stacks/hugo.md
summary: Pre-built static files (hand-authored or generated) served from any web host or CDN, with no application server on the request path.
---

# Static Website / JAMstack

**Focus:** pre-rendered static assets · served from a web host / CDN · no server-side application on the request path.

## Definition
The site is a set of static files — HTML, CSS, JS, images — produced either by hand or by a build step (a static site generator), then served directly by a web host or CDN. There is no application server rendering pages per request. Any dynamic behaviour runs **client-side** in the browser (calling separate APIs or serverless functions) or is baked in **at build time**. This is the "JAMstack" shape at its simplest, and the lowest-complexity way to put a site online.

## Data strategy
Usually **none on the server**. Content is either embedded at build time (Markdown/data files compiled into pages) or fetched **client-side** from a separate API or serverless backend. There is no request-time database — the "database" is the build input or a remote service.

## Scaling
**Trivially horizontal.** Static files behind a CDN scale to very high read volume for near-zero cost and effort; there is no server to saturate. The constraint is build time and content volume, not concurrency.

## Best fit
Marketing sites, landing pages, documentation, blogs, portfolios, product brochures — anything content-first and read-mostly. *(kaos-control's own marketing site `kaos-control.io` is this shape, as is the Hugo-built `tek42.io`.)*

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | Cacheable; fully static assets |
| Collaboration / shared state | No (read-mostly; dynamic bits via external APIs) |
| Scale | Very high read volume, near-zero cost |
| Complexity to build | Very low |
| Team skill required | Low |

## Pros
- Cheapest and simplest thing to host and operate — no server, no runtime to patch.
- Extremely fast (CDN-served) and highly available; a huge attack surface simply isn't there.
- Trivial rollbacks (redeploy the previous build); content lives in git.

## Cons
- No request-time server logic — anything dynamic needs client-side JS + a separate API/serverless backend.
- Personalisation, auth, and write operations are awkward (they belong to an API, not the site).
- Large content sets can mean slow full-site rebuilds.

## Suitable tech stacks
- [Static HTML / CSS / JS](../tech-stacks/static-html-js.md) — hand-authored, no framework build (kaos-control.io's shape).
- [Hugo](../tech-stacks/hugo.md) — Markdown content compiled to a fast static site (tek42.io's shape).

## Related architectures
When the site needs request-time logic or writes, add a backend and it shades into [[architecture/architectures/serverless-faas|Serverless / FaaS]] (functions behind the static front-end) or a [[architecture/architectures/single-service-saas|Single-Service Cloud SaaS]] (a full app). For a simple *server-rendered* site instead of a static one, see the [Simple PHP](../tech-stacks/php-simple.md) stack under [[architecture/architectures/local-web|Local Web Application]].
