# Service Level Objectives — orders-api

**Status**: Draft v0.1
**Owner**: SRE / Observability working group
**Last updated**: 2026-04-27

## Purpose

This document defines the first measurable Service Level Objective (SLO)
for the `orders-api` service in the `commerce` namespace. It is the
operational implementation of the reliability targets discussed in the
Master's thesis on standardized observability across heterogeneous
services. The aim is to demonstrate that a vendor-neutral observability
stack (OpenTelemetry + Prometheus) can support the full SRE alerting
discipline — not just dashboards.

## Service definition

`orders-api` is a public-facing HTTP service that accepts `POST /orders`
requests, calls `inventory-api` for stock availability, simulates a
payment authorization, and returns an order confirmation. It runs on
Kubernetes as a Deployment with 2 replicas exposed via NodePort.

## Service Level Indicators (SLIs)

We define two SLIs, both derived from the
`http_server_request_duration_seconds` histogram metric exported by the
OpenTelemetry-instrumented service.

### SLI-1: Availability

The proportion of `POST /orders` requests that complete successfully.
"Successful" excludes 5xx server errors. 4xx responses (e.g., HTTP 402
"payment declined") are treated as **successful** because they represent
correct business-logic responses, not service failures.

Formula:

sum(rate(http_server_request_duration_seconds_count{
exported_job="commerce/orders-api",
http_route="/orders",
http_response_status_code!~"5.."
}[5m]))
/
sum(rate(http_server_request_duration_seconds_count{
exported_job="commerce/orders-api",
http_route="/orders"
}[5m]))


### SLI-2: Latency

The proportion of `POST /orders` requests completing in under 500ms.

Formula:

sum(rate(http_server_request_duration_seconds_bucket{
exported_job="commerce/orders-api",
http_route="/orders",
le="0.5"
}[5m]))
/
sum(rate(http_server_request_duration_seconds_count{
exported_job="commerce/orders-api",
http_route="/orders"
}[5m]))


## Service Level Objectives (SLOs)

| SLI | Objective | Window |
|---|---|---|
| Availability | 99.0% | rolling 30 days |
| Latency (<500ms) | 95.0% | rolling 30 days |

## Error budget

For the **availability** SLO of 99% over 30 days:

- Allowed unavailability: 1% × 30 days × 24 hours = **7.2 hours / month**
- This is the error budget. If the service is unavailable for less than
  7.2 hours total in the rolling window, we are within SLO. If we exceed
  it, the service has missed its SLO and the team should pause feature
  work in favor of reliability work.

## Burn-rate alerting (Google SRE Workbook pattern)

A naive alert ("page when availability drops below 99% in the last 5
minutes") fires too often (false positives from brief blips) and too
late (small steady leaks go unnoticed for days). The multi-window
burn-rate pattern fixes both.

The principle: alert when the rate at which we're consuming error budget
is so fast that, if it continued, we'd exhaust the entire month's budget
in much less than a month.

| Severity | Burn rate | Short window | Long window | Time to exhaust budget |
|---|---|---|---|---|
| Page | 14.4× | 5m | 1h | 2 hours |
| Ticket | 6× | 30m | 6h | 5 days |

A `14.4×` burn rate means we are consuming error budget 14.4 times
faster than the SLO target permits. The alert requires the burn to be
sustained over BOTH the short window (sensitivity) AND the long window
(reduced false positives).

## Notes

- The **alert rules** that implement this SLO live in
  `observability/prometheus/rules/orders-api-slo.yaml`.
- This SLO targets `POST /orders` specifically. Health-check traffic
  (`/healthz`) is excluded from the SLI numerator and denominator.
- Kubernetes liveness probes that legitimately fail (e.g., during a
  rolling restart) should NOT count against the SLO; they're part of
  normal cluster operations.
- Real production deployments would have an Alertmanager wired to
  page (PagerDuty, Opsgenie) and ticket (Jira) channels. This local
  development environment ships the alert rules as code; routing is
  scoped to v2.

## Roadmap

- v0.2: Add a third SLI for inventory-api (latency only — it's an
  internal service with no business logic surface).
- v0.3: Wire Alertmanager and route alerts to a channel (initially
  email, later PagerDuty for true paging).
- v0.4: Add error budget burn dashboards in Grafana with a 28-day
  trailing budget bar.## Service Level Objectives (SLOs)

| SLI | Objective | Window |
|---|---|---|
| Availability | 99.0% | rolling 30 days |
| Latency (<500ms) | 95.0% | rolling 30 days |

## Error budget

For the **availability** SLO of 99% over 30 days:

- Allowed unavailability: 1% × 30 days × 24 hours = **7.2 hours / month**
- This is the error budget. If the service is unavailable for less than
  7.2 hours total in the rolling window, we are within SLO. If we exceed
  it, the service has missed its SLO and the team should pause feature
  work in favor of reliability work.

## Burn-rate alerting (Google SRE Workbook pattern)

A naive alert ("page when availability drops below 99% in the last 5
minutes") fires too often (false positives from brief blips) and too
late (small steady leaks go unnoticed for days). The multi-window
burn-rate pattern fixes both.

The principle: alert when the rate at which we're consuming error budget
is so fast that, if it continued, we'd exhaust the entire month's budget
in much less than a month.

| Severity | Burn rate | Short window | Long window | Time to exhaust budget |
|---|---|---|---|---|
| Page | 14.4× | 5m | 1h | 2 hours |
| Ticket | 6× | 30m | 6h | 5 days |

A `14.4×` burn rate means we are consuming error budget 14.4 times
faster than the SLO target permits. The alert requires the burn to be
sustained over BOTH the short window (sensitivity) AND the long window
(reduced false positives).

## Notes

- The **alert rules** that implement this SLO live in
  `observability/prometheus/rules/orders-api-slo.yaml`.
- This SLO targets `POST /orders` specifically. Health-check traffic
  (`/healthz`) is excluded from the SLI numerator and denominator.
- Kubernetes liveness probes that legitimately fail (e.g., during a
  rolling restart) should NOT count against the SLO; they're part of
  normal cluster operations.
- Real production deployments would have an Alertmanager wired to
  page (PagerDuty, Opsgenie) and ticket (Jira) channels. This local
  development environment ships the alert rules as code; routing is
  scoped to v2.

## Roadmap

- v0.2: Add a third SLI for inventory-api (latency only — it's an
  internal service with no business logic surface).
- v0.3: Wire Alertmanager and route alerts to a channel (initially
  email, later PagerDuty for true paging).
- v0.4: Add error budget burn dashboards in Grafana with a 28-day
  trailing budget bar.