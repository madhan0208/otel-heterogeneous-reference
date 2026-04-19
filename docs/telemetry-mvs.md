
# Telemetry Minimum Standard (MVS) v0.1

## Purpose

This document defines the minimum telemetry conventions that every service
within scope must implement. Its goals are:

1. Guarantee cross-service correlation between logs, metrics, and traces
2. Eliminate naming drift across language ecosystems (.NET, Go, Java, Python)
3. Enable backend-agnostic consumption (any OTLP-compatible backend)
4. Bound observability cost through explicit sampling and retention rules

**Scope**: HTTP-based application services running on Kubernetes.
**Out of scope for v0.1**: async workers, streaming, batch jobs, infrastructure metrics.

## 1. Resource attributes (required on every signal)

Every service MUST emit the following resource attributes on every span,
metric, and log record:

| Attribute | Source | Example | Notes |
|---|---|---|---|
| `service.name` | App config | `orders-api` | Stable, kebab-case, no environment suffix |
| `service.version` | Build-time | `1.4.2` | SemVer; injected at image build |
| `service.namespace` | Deployment | `commerce` | Business domain grouping |
| `service.instance.id` | Runtime | Pod UID | From Kubernetes downward API |
| `deployment.environment` | Deployment | `dev`, `staging`, `prod` | Lowercase only |
| `k8s.pod.name` | Runtime | Auto | From Kubernetes downward API |
| `k8s.namespace.name` | Runtime | Auto | From Kubernetes downward API |

These attributes are injected via environment variables following the
`OTEL_RESOURCE_ATTRIBUTES` standard. The Collector's `resourcedetection`
processor fills Kubernetes-specific fields automatically.

## 2. Span conventions

### 2.1 Span naming

- **HTTP server spans**: `{HTTP_METHOD} {route_template}` — e.g., `POST /orders`
- **HTTP client spans**: `{HTTP_METHOD} {target_service}` — e.g., `GET inventory-api`
- **Manual business spans**: `{domain}.{action}` — e.g., `orders.validate_payment`

### 2.2 Required span attributes

- **HTTP server**: `http.request.method`, `http.route`, `http.response.status_code`, `url.path`
- **HTTP client**: `http.request.method`, `server.address`, `http.response.status_code`
- **Business spans**: at least one `{domain}.{entity}.id` attribute when operating on a known entity

### 2.3 Prohibited span attributes

- No request bodies, response bodies, or headers as span attributes
- No PII: email, name, address, payment details
- No secrets: tokens, API keys, passwords (even masked)

Rationale: span attributes are indexed and stored per-span. Including high-cardinality
or sensitive data inflates cost and creates data-protection risk.

## 3. Metric conventions

### 3.1 RED metrics (required per HTTP endpoint)

- `http.server.request.duration` — histogram, seconds
- `http.server.request.count` — counter (derivable from histogram; kept explicit for clarity)
- `http.server.errors` — counter, filtered to 5xx responses

### 3.2 Required labels

Every metric MUST carry: `service.name`, `http.route`, `http.response.status_code`.

### 3.3 Naming rules

Follow OpenTelemetry semantic conventions. No custom renames. No prefix with team or company name.

## 4. Log conventions

### 4.1 Structured JSON only

Every log line MUST be emitted as a JSON object with at minimum:

```json
{
  "timestamp": "2026-04-18T14:23:01.123Z",
  "severity": "INFO",
  "body": "order created",
  "service.name": "orders-api",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7"
}
```

### 4.2 Trace correlation

When a log is emitted within an active span, `trace_id` and `span_id` MUST
be present on the log record. This is the single most important rule for
MTTR reduction: it enables one-click navigation between logs and traces in Grafana.

### 4.3 Severity

Use OpenTelemetry severity names: `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`.

## 5. Sampling

### 5.1 Traces

- **Head sampling at SDK**: `ParentBased(AlwaysOn)` — keep all traces in dev
- **Tail sampling at gateway** (prod profile): keep 100% of errors, 100% of slow traces (>1s), 10% baseline

### 5.2 Metrics

- No sampling. Metrics are aggregated at the SDK and exported every 60 seconds.

### 5.3 Logs

- No sampling of `ERROR` or higher severity
- `INFO` and below: sample at 100% in dev, 50% in prod (documented here, enforced in future version)

## 6. PII and secret redaction

The Collector gateway applies redaction via `attributes` and `transform`
processors for the following patterns: email addresses, bearer tokens,
credit card numbers (Luhn-validated), IBANs.

See `observability/collector/gateway-config.yaml` for the exact rules.

## 7. Compliance

A compliance test suite (`tests/mvs_compliance.py`) asserts that each
service emits the required resource attributes, span naming patterns, and
log-trace correlation. Services MUST pass this suite before merging to main.

## 8. Versioning

This standard is versioned. Breaking changes to required fields require
a major version bump. This document is v0.1.

## Appendix A — Rationale for specific choices

**Why OpenTelemetry semantic conventions verbatim?** Deviating creates incompatibility with pre-built dashboards, alerting rules, and vendor tooling. The cost of deviation compounds; the cost of conformance is one-time.

**Why structured JSON logs?** Parsing unstructured logs at query time is expensive and fragile. JSON is the lowest common denominator that every backend and SDK supports.

**Why enforce trace_id in logs?** Without it, logs and traces are separate silos that humans must join mentally during incidents. With it, navigation is one click, and MTTR drops measurably.

**Why reject per-service custom metric prefixes?** Prefixes break cross-service aggregation and require rewriting dashboards when teams reorganize. Use resource attributes for grouping instead.
