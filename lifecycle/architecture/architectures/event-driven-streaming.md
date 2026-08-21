---
title: Event-Driven / Streaming Architecture
type: architecture
status: approved
lineage: arch-event-driven-streaming
labels:
    - architecture
    - catalog
    - realtime
    - high-scale
    - high-complexity
related_to:
    - architecture/tech-stacks/go-grpc-microservices.md
    - architecture/tech-stacks/java-spring-angular.md
    - architecture/tech-stacks/python-fastapi.md
summary: System flow driven by asynchronous events through a broker; high-throughput stream processing with eventual consistency.
---

# Event-Driven / Streaming Architecture

**Focus:** high-volume transactions · asynchronous messaging · real-time stream processing · eventual consistency.

## Definition
System flow is determined by **events** — significant changes in state (a click, a sensor reading, a completed payment). Components communicate asynchronously through an event broker (Apache Kafka, RabbitMQ) rather than direct request/response.

## Data strategy
**Eventual consistency.** Instead of immediate ACID transactions across the system, data is a stream of immutable facts; state is reconstructed by replaying logs or through materialized views of the event stream.

## Scaling
**Horizontal via partitioning** — add partitions to a topic and more consumers to a consumer group for massive parallel stream processing.

## Best fit
Real-time fraud detection in banking, high-frequency IoT telemetry, real-time analytics engines, complex supply-chain tracking.

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | No |
| Collaboration / shared state | Yes (async) |
| Scale | Very high (throughput) |
| Complexity to build | High |
| Team skill required | High (streaming + consistency) |

## Pros
- Extreme decoupling; very high throughput for large data volumes.
- React to events in real time.
- Resilient to service interruptions (events queue up).

## Cons
- Hard to manage eventual consistency.
- Difficult to trace end-to-end request flows.
- Handling out-of-order and duplicate events adds complexity.

## Suitable tech stacks
- [Go + gRPC Microservices](../tech-stacks/go-grpc-microservices.md) — streaming RPC + high-throughput producers/consumers.
- [Java + Spring Boot / Angular](../tech-stacks/java-spring-angular.md) — mature Kafka ecosystem.
- [Python + FastAPI](../tech-stacks/python-fastapi.md) — analytics and ML consumers on the stream.

## Related architectures
Almost always layered on top of [[architecture/architectures/cloud-native-microservices|Cloud-Native Microservices]], and a natural fit for the cloud side of an [[architecture/architectures/edge-hybrid|Edge / Distributed Hybrid]].
