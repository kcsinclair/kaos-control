---
title: Edge / Distributed Hybrid
type: architecture
status: approved
lineage: arch-edge-hybrid
labels:
    - architecture
    - catalog
    - offline-capable
    - realtime
    - high-complexity
related_to:
    - architecture/tech-stacks/go-vue.md
    - architecture/tech-stacks/flutter.md
    - architecture/tech-stacks/python-fastapi.md
summary: Splits computing between local edge nodes (low-latency, time-critical logic) and the cloud (heavy processing, long-term storage).
---

# Edge / Distributed Hybrid

**Focus:** low-latency localised processing + cloud synchronisation · IoT / autonomous-vehicle use cases.

## Definition
A hybrid architecture that distributes intelligence between the central cloud and **edge** nodes (local gateways, mobile devices, specialised hardware). Critical, time-sensitive logic runs at the edge to minimise latency; non-critical data syncs to the cloud for heavy processing and long-term storage.

## Data strategy
Tiered — localised/ephemeral state for immediate action (cached at the edge) plus periodic/asynchronous synchronisation of aggregated data to a central cloud database for global visibility.

## Scaling
**Distributed horizontal** — deploy more edge devices into the field and expand cloud capacity to absorb increased telemetry/sync overhead.

## Best fit
Autonomous vehicles (instant collision avoidance vs. long-term map updates), smart-city infrastructure, industrial IoT in remote mining or oil platforms.

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | Yes (edge continues during outages) |
| Collaboration / shared state | Cloud-synced |
| Scale | High (many devices) |
| Complexity to build | High |
| Team skill required | High (distributed + embedded) |

## Pros
- Extremely low latency for critical operations.
- Bandwidth efficiency (send summaries, not raw data).
- Keeps working during intermittent connectivity.

## Cons
- Complex conflict-resolution / synchronisation logic.
- Large, hard-to-secure surface area across many physical locations.
- Managing distributed software versions is hard.

## Suitable tech stacks
- [Go + Vue](../tech-stacks/go-vue.md) — small, efficient single-binary edge agent.
- [Flutter](../tech-stacks/flutter.md) — UI on edge/handheld devices.
- [Python + FastAPI](../tech-stacks/python-fastapi.md) — cloud-side ML/aggregation.

## Related architectures
The cloud tier is typically [[architecture/architectures/event-driven-streaming|Event-Driven / Streaming]] or [[architecture/architectures/cloud-native-microservices|Cloud-Native Microservices]]; the device tier overlaps with [[architecture/architectures/mobile-native|Mobile-Native]].
