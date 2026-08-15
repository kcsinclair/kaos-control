---
title: Go + gRPC Microservices Backbone
type: tech-stack
status: draft
lineage: stack-go-grpc-microservices
labels:
    - tech-stack
    - catalog
    - backend
    - go
    - grpc
summary: Go services communicating over gRPC/Protobuf on HTTP/2. The gold standard for high-performance service-to-service traffic.
---

# Go + gRPC Microservices Backbone

**Focus:** high-performance distributed systems. The internal backbone for low-latency, strictly-typed service-to-service communication.

## Overview
**Go** provides goroutine concurrency and near-C execution speed — ideal for a high-throughput gRPC server — on top of a standard library with first-class HTTP/2 support. Where REST/JSON is right for public APIs, **Go + gRPC is the gold standard for internal service-to-service traffic** in high-scale microservices, where performance, type safety, and strict contracts matter.

## Communication layer
**gRPC over HTTP/2** — multiplexing, header compression, and four call styles:
- **Unary** — simple request/response.
- **Server streaming** — one request, a stream of responses (e.g. live price updates).
- **Client streaming** — a stream of messages up to the server.
- **Bidirectional streaming** — continuous both-way flow (chat, telemetry).

The contract lives in **`.proto` files** (Protocol Buffers): a language-neutral, binary, strictly-typed schema. Client and server code is generated from it, so breaking changes are caught at **compile time**, and a Go service can talk to a Python ML service or a TS frontend over the same contract.

## Data persistence
Database-per-service (PostgreSQL, Redis, or domain-appropriate stores); no shared schemas across services.

## Why it wins
- **Performance** — binary Protobuf + HTTP/2 multiplexing → far smaller payloads and lower serialization CPU than JSON.
- **Contract enforcement** — the `.proto` is the single source of truth; mismatches fail at compile time, not runtime.
- **Polyglot interop** — strict contracts across Go, Python, Java, TS.

### REST vs. gRPC
| Feature | REST (HTTP/1.1) | gRPC (HTTP/2) |
| --- | --- | --- |
| Payload | Text (JSON/XML) | Binary (Protobuf) |
| Communication | Request/response | Unary + bidirectional streaming |
| Contract | Implicit (docs) | Explicit (`.proto`) |
| Browser support | Universal | Limited (needs gRPC-web proxy) |
| Efficiency | Medium/low | Extremely high |

## Profile
| Trait | Value |
| --- | --- |
| Layer | Backend / inter-service transport |
| Languages | Go (+ polyglot clients) |
| Footprint | Low |
| Learning curve | Moderate–high (Protobuf, streaming) |
| Best use case | Internal microservice backbone, real-time streaming |

## Suits these architectures
- [Cloud-Native Microservices](../architectures/cloud-native-microservices.md) — the service-to-service backbone.
- [Event-Driven / Streaming](../architectures/event-driven-streaming.md) — streaming RPC + high-throughput consumers.
- [Mobile-Native](../architectures/mobile-native.md) — efficient, low-battery mobile-to-backend transport.

> Note: browsers can't speak gRPC directly — pair with a gRPC-web proxy or a REST/GraphQL edge (e.g. a [Go + Vue](go-vue.md) gateway) for public/browser traffic.

## Stack profile

See [[agent-directives-generation]]. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: go run ./cmd/<service>
  repo_layout:
    - {path: cmd/,      note: one main package per service}
    - {path: internal/, note: service implementations + shared libs}
    - {path: proto/,    note: protobuf definitions}
    - {path: gen/,      note: generated gRPC/protobuf code}
    - {path: tests/,    note: integration tests}
  roles:
    backend-developer:
      write_paths: [cmd, internal, proto]
      build: go build ./...
      lint: go vet ./...
      test: go test ./... -short
    frontend-developer:
      required: false          # service backbone — no UI
    test-developer:
      write_paths: [tests]
      test: go test -tags integration ./tests/...
```
