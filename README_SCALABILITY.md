# Aegis Scalability Architecture

## Overview

Aegis scales by combining fast, low‑latency gRPC for synchronous work with resilient control primitives (timeouts, circuit breaking, retries, and concurrent call limiting) and an asynchronous Kafka fallback for overflow and failure paths. The result is graceful degradation under stress, stable tail latency, and predictable throughput as services scale horizontally.

## Design Pillars

- Stateless service interactions via gRPC for easy horizontal scaling
- Strict request timeouts (1s) to cap tail latency and protect callers
- Circuit breaker with half‑open probing to prevent cascading failures
- Exponential backoff with jitter to smooth retries under contention
- Concurrent call limiting for proactive backpressure
- Kafka fallback to decouple producers from overloaded/unavailable consumers

## Request Flow Under Load

1. Client issues a gRPC request with a 1‑second deadline.
2. Circuit breaker is consulted; if open, the request is short‑circuited and routed to Kafka.
3. If allowed, the call proceeds with concurrent call limits enforced.
4. On timeout or retriable errors, the client applies bounded retries with backoff + jitter.
5. After failure or when policy dictates, the request is sent to a service‑specific Kafka topic for eventual processing.

## Resilience and Flow Control

- Timeouts: Every gRPC call has a 1‑second deadline, guaranteeing bounded wait times.
- Circuit breaker: Transitions Closed → Open after consecutive failures; Half‑Open probes for recovery and closes after a small number of successes.
- Concurrent call limiting: Caps in‑flight calls per service to protect memory/CPU and downstream capacity.
- Retries: Exponential backoff with jitter; only for safe, transient errors (timeout, service unavailable, circuit open).
- Thread safety: All primitives are safe under concurrency, avoiding race conditions in hot paths.

## Asynchronous Fallback (Kafka)

- Automatic fallback: Failed/timeout requests are serialized (JSON) and published to service‑specific topics.
- Decoupling: Producers continue at steady throughput while consumers scale independently.
- Backpressure: Kafka absorbs bursts; consumers can scale by increasing partitions and group members.


## Consistency and Idempotency

- Eventual consistency: Async fallback ensures progress despite transient failures.

## Tuning and Sizing

- Timeouts: Keep at 1s for interactive paths; shorten or lengthen only with evidence.
- Circuit breaker: Tune failure threshold and open timeout per service; start with failure threshold ≈ 5 and half‑open success threshold ≈ 2.
- Retry policy: Cap attempts; prefer small initial delay and bounded max backoff with jitter.
- Concurrency limits: Set per‑service based on downstream capacity and CPU/memory headroom.
- Kafka: Scale partitions to match consumer parallelism; monitor consumer lag and adjust.

## Deployment Notes

- Horizontal scaling: Services and clients are stateless; scale replicas without coordination.
- API gateway: Replace blocking HTTP proxy calls with resilient gRPC client usage for lower latency and better failure isolation.
- Infrastructure: Add Kafka brokers and topic management; use Docker Compose/Kubernetes to scale services and consumers.

## Testing and Validation

- Unit tests: Cover circuit breaker transitions, retry behavior, and Kafka fallback.
- Integration tests: Exercise end‑to‑end resilient client paths with induced timeouts/failures.
- Concurrency tests: Validate thread safety and performance under multi‑goroutine load.
- Load testing: Ramp QPS to observe latency, error rates, and fallback behavior.

## Cross‑References

- shared/grpc/client.go: Core resilient gRPC client with timeout, retries, fallback
- shared/circuitbreaker/: Circuit breaker with Closed/Open/Half‑Open and limits
- shared/retry/: Exponential backoff with jitter and error‑aware retry policy
- shared/kafka/: Kafka producer, topic mapping, and mocks for testing
- proto/: Service definitions for Market, Wallet, and Settlement
