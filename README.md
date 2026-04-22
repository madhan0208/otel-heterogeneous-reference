# otel-heterogeneous-reference

> Reference implementation of a Telemetry Minimum Standard (MVS) across heterogeneous services using OpenTelemetry.

**Status**: two services running, distributed tracing verified. OTel Collector, Prometheus, Grafana, Jaeger, Loki, SLO alerts, chaos experiments.

## What this is

An end-to-end example of instrumenting a multi-language, multi-service application on Kubernetes with OpenTelemetry — traces, metrics, and logs — exported to vendor-neutral backends (Jaeger, Prometheus, Loki). Every telemetry decision is driven by a Telemetry Minimum Standard (MVS) document that services must conform to.

Companion artifact to an ongoing Master's thesis on enterprise observability standardization in heterogeneous IT landscapes.

## Why this exists

Most observability tutorials instrument a single service in a single language and call it done. Real enterprises run heterogeneous stacks — .NET, Java, Go, Python — with inconsistent telemetry conventions, unreliable trace propagation, and dashboards that don't compose across teams. This project demonstrates a different approach: **define the standard first, implement it identically across languages, keep the backend plug-replaceable.**

## Architecture

_Architecture diagram coming soon._

The system consists of:

- **orders-api** — .NET 8 minimal API, calls inventory-api to validate stock
- **inventory-api** — Go service returning stock availability
- **OpenTelemetry Collector** — agent (DaemonSet) + gateway (Deployment) pattern
- **Jaeger** — trace backend
- **Prometheus + Alertmanager** — metrics and SLO-based alerting
- **Loki** — log aggregation
- **Grafana** — single pane of glass across all three signals

## Quick start

```bash
docker compose up --build
# In another terminal:
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"itemId":"sku-123","qty":2}'
```

Two services start, orders-api calls inventory-api, and a single trace ID flows through both — verifying end-to-end W3C trace context propagation across .NET and Go.

## Documentation

- **[Telemetry Minimum Standard (MVS)](docs/telemetry-mvs.md)** — the standard every service conforms to _(coming soon)_
- **[SLO definitions](docs/slos.md)** — availability and latency targets with burn-rate alerts _(coming soon)_
- **[Chaos experiments](docs/chaos/)** — documented failure scenarios with post-mortems _(coming soon)_
- **[Architectural decisions](DECISIONS.md)** — ADR log explaining key choices

## Repository structure
