Service 1 Language - .NET 8 (Leverages my thesis work)
Service 2 Language - Go 1.22+ (Cloud native lingua franca)
Local Kubernetes - kind (faster than minikube , multi-node config
Container runtime - Docker Desktop (Works with kind out of box)
License - MIT (Permissive . standard for portfolio repos)


# Architectural Decisions

This document records the key architectural choices made for this project,
in ADR (Architecture Decision Record) style. Each entry captures the decision,
the rationale, and the alternatives considered.

## ADR-001: Heterogeneous language choice

- **Decision**: Implement two services in different languages — .NET 8 (orders-api) and Go 1.22 (inventory-api).
- **Rationale**: The core thesis of this project is vendor-neutral, language-neutral telemetry. A single-language demo would not prove the claim. .NET and Go together cover the majority of enterprise cloud-native workloads.
- **Alternatives considered**: Python + Go, all .NET, all Go. Rejected because they either diluted the heterogeneity story or didn't reflect enterprise reality.

## ADR-002: kind over minikube for local Kubernetes

- **Decision**: Use kind (Kubernetes IN Docker) for local cluster provisioning.
- **Rationale**: Faster cluster startup, built-in multi-node support via config file, better CI integration, and the kind config itself is a useful artifact to show.
- **Alternatives considered**: minikube (slower boot), k3d (fine but less adopted), Docker Desktop's built-in K8s (single-node only).

## ADR-003: Agent + Gateway Collector topology

- **Decision**: Deploy the OpenTelemetry Collector in two tiers — DaemonSet agents on each node, Deployment gateway with 2 replicas.
- **Rationale**: Agents handle local batching and resource enrichment cheaply, close to the workload. The gateway centralizes sampling policy, PII redaction, and multi-backend export. This mirrors production-realistic topologies at companies operating at meaningful scale.
- **Alternatives considered**: Single-tier (agent only) — simpler but no central policy enforcement. Gateway only — puts all load on one tier, defeats the purpose.

## ADR-004: Three backends side-by-side (Jaeger, Prometheus, Loki)

- **Decision**: Configure the Collector gateway with three parallel exporters — traces to Jaeger, metrics to Prometheus, logs to Loki.
- **Rationale**: This is the executable proof of the vendor-neutral claim. Swapping Jaeger for Tempo, or Loki for Elastic, requires changing only the Collector config, not application code. Demonstrates the core value proposition from the thesis.
- **Alternatives considered**: A single unified backend (e.g., Grafana Cloud). Rejected because it contradicts the vendor-neutral thesis.

## ADR-005: Telemetry Minimum Standard (MVS) defined before implementation

- **Decision**: Write `docs/telemetry-mvs.md` specifying required resource attributes, span conventions, metric naming, log structure, and sampling policy **before** writing any instrumentation code.
- **Rationale**: The MVS is the primary artifact of the Master's thesis this project accompanies. Writing the standard first ensures both services conform to the same conventions, and the document itself becomes reviewable evidence of design-first thinking.
- **Alternatives considered**: Derive conventions from code ("start instrumenting, see what we need"). Rejected because it produces inconsistent results across services, which is exactly the problem this project is meant to solve.

## ADR-006: Chiseled .NET and distroless Go runtime images

- **Decision**: Use `mcr.microsoft.com/dotnet/aspnet:8.0-jammy-chiseled` for .NET and `gcr.io/distroless/static-debian12:nonroot` for Go.
- **Rationale**: Minimal attack surface, smaller image sizes, non-root by default. Standard practice for production cloud-native deployments and a concrete talking point for security-aware reviewers.
- **Alternatives considered**: Alpine (smaller but libc issues with .NET), standard Debian/Ubuntu base (larger, runs as root).
