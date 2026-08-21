---
title: Mobile-Native Application
type: architecture
status: approved
lineage: arch-mobile-native
labels:
    - architecture
    - catalog
    - mobile
    - offline-capable
    - medium-complexity
related_to:
    - architecture/tech-stacks/flutter.md
    - architecture/tech-stacks/go-grpc-microservices.md
    - architecture/tech-stacks/go-vue.md
summary: A phone/tablet app as the primary client, talking to a cloud backend; on-device state with sync, app-store distribution.
---

# Mobile-Native Application

**Focus:** phone/tablet as the primary client · on-device experience · backend sync · app-store distribution.

## Definition
The primary interface is a mobile app running on iOS/Android — either truly native (Swift/Kotlin) or cross-platform (Flutter, React Native) — backed by a cloud API for data, auth, and sync. The device handles UI, local caching, and device capabilities (camera, GPS, notifications); the backend handles shared data and business logic.

## Data strategy
**On-device store** (SQLite, Core Data, Room, or an offline-first DB like Realm/WatermelonDB) for responsiveness and offline use, **synchronised** to a cloud backend when connectivity allows. Conflict handling on sync is a first-class concern.

## Scaling
The **backend** scales like a cloud service (see [Single-Service SaaS](single-service-saas.md) / [Microservices](cloud-native-microservices.md)); the **client** scales by distribution through the app stores. Push notifications and background sync drive engagement.

## Best fit
Consumer apps, field/frontline tools, anything needing device hardware (camera, location, sensors), and offline-tolerant workflows.

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | Yes (offline-first + sync) |
| Collaboration / shared state | Yes (via backend) |
| Scale | Backend: high; client: app-store reach |
| Complexity to build | Moderate (app + backend + sync + store) |
| Team skill required | Moderate–high (mobile + backend) |

## Pros
- Best-in-class device UX and access to native capabilities.
- Works offline; syncs opportunistically.
- Discoverable via app stores; push re-engagement.

## Cons
- Two problems at once — the app *and* its backend.
- App-store review/release cadence and platform fragmentation.
- Offline sync + conflict resolution is genuinely hard.

## Suitable tech stacks
- [Flutter](../tech-stacks/flutter.md) — one codebase for iOS + Android (and desktop) with high-FPS UI.
- [Go + gRPC Microservices](../tech-stacks/go-grpc-microservices.md) — efficient, low-battery mobile-to-backend transport.
- [Go + Vue](../tech-stacks/go-vue.md) — lean REST/JSON backend + a companion web console.

## Related architectures
Its backend is typically a [[architecture/architectures/single-service-saas|Single-Service Cloud SaaS]] or [[architecture/architectures/cloud-native-microservices|Cloud-Native Microservices]]; the device tier overlaps with [[architecture/architectures/edge-hybrid|Edge / Distributed Hybrid]] and, for offline desktop equivalents, [[architecture/architectures/standalone-desktop|Standalone Desktop]].
