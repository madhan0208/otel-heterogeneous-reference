# Post-mortem: Deliberate inventory-api pod failure (2026-04-29)

**Status**: Resolved
**Severity**: SEV-2 (test scenario, simulated production incident)
**Author**: Madhan Ramesh
**Date**: 2026-04-29

## Summary

Both `inventory-api` pods in the `commerce` namespace were deliberately
deleted at the same time during steady-state traffic of 1 order/second.
Result: `orders-api` started returning HTTP 502 to clients because
upstream calls to `inventory-api` failed with connection refused.
Kubernetes' Deployment controller recreated the pods, and traffic
returned to baseline within approximately 30 seconds.

The purpose of this experiment was to validate the SLO alert rule
defined in `docs/slo.md` and `observability/prometheus/rules/orders-api-slo.yaml`,
and to confirm that the observability stack (Prometheus + OTel
Collector + apps) correctly surfaces the impact of an incident.

## Timeline (all times UTC)

| Time | Event |
|---|---|
| 21:32:03 | Steady-state traffic started (1 req/s, all 200s) |
| 21:38:58 | `kubectl delete pod -n commerce -l app.kubernetes.io/name=inventory-api` issued |
| 21:38:58 | First 502 observed in client |
| 21:50:00 | Replacement pods reported `1/1 Running` |
| 21:50:00 | First 200 after recovery |
| 21:50:00 | Alert OrdersApiHighErrorBudgetBurnFast state observed: ... |
| 21:51:32 | Experiment ended |



## What the user saw

Clients calling `POST /orders` received HTTP 502 (Bad Gateway) for
roughly 30 seconds. After recovery, all subsequent orders succeeded
(with the usual ~5% rate of 402 "payment declined" responses, which
are business-correct, not failures).

## What the system did

1. Both inventory-api pods received SIGTERM and stopped serving traffic.
2. Kubernetes' Service object lost both endpoints; orders-api's HTTP
   client to `inventory-api.commerce.svc.cluster.local` got connection
   refused.
3. The Deployment controller noticed `replicas: 2` was no longer met
   and immediately created replacement pods.
4. Replacement pods went through `Pending` → `ContainerCreating` →
   `Running` → `Ready` (passing the readiness probe).
5. As soon as the readiness probe passed, the Service object added
   the new pod IP to its endpoints, and orders-api's connections
   resumed succeeding.

## What the observability stack saw

The OpenTelemetry instrumentation on orders-api correctly surfaced the
incident:

- Auto-instrumented HTTP client spans on orders-api recorded errors
  with HTTP status 502 (visible in spans, ready to be sent to Jaeger).
- Auto-instrumented HTTP server spans on orders-api recorded the
  external request as `POST /orders` returning 502.
- Histogram buckets for `http_server_request_duration_seconds`
  incremented in the `5xx` status code label.
- The Prometheus alert rule `OrdersApiHighErrorBudgetBurnFast`
  reached state: ... (Pending / Firing / remained Inactive).

Screenshots:
- `docs/chaos/01-pre-incident-alerts-inactive.png` — alerts inactive baseline
- `docs/chaos/02-pre-incident-baseline.png` — healthy traffic graph
- `docs/chaos/03-during-incident-5xx-spike.png` — 5xx spike in Prometheus
- `docs/chaos/04-alert-firing.png` — alert state during incident (or omitted if alert remained Inactive)
- `docs/chaos/05-recovery.png` — full incident arc with recovery

## What worked well

- **Detection was automatic.** The OpenTelemetry pipeline captured
  the 5xx burst without any operator intervention.
- **Recovery was automatic.** Kubernetes' built-in mechanisms
  restored the service without human action. Total impact ~30s.
- **Observability was complete.** Every relevant metric label was
  preserved through the OTLP pipeline; status code, route, service
  all visible in Prometheus.
- **The error budget framing made impact concrete.** "30 seconds of
  downtime" is hard to reason about; "consumed X% of monthly error
  budget" is immediately actionable.

## What didn't work / improvements

- **Recovery time depends on probe configuration.** Readiness probe
  interval × failure threshold means it takes ~15s after container
  start before the new pod gets traffic. We could shrink this for
  faster recovery, at the cost of more probe load.
- **No retry/circuit-breaker in orders-api.** When inventory-api was
  unreachable, orders-api returned 502 immediately. A real production
  system should retry with backoff and/or implement a circuit breaker.
  This would convert some 502s into successes (after retry) or faster
  failures (degraded mode).
- **No graceful degradation.** If inventory-api is genuinely down, is
  there ANY business response we could give? E.g., "your order is
  queued for stock confirmation"? The current design has only binary
  success/failure.
- **Single failure-domain risk.** Both inventory-api replicas could
  be killed by one mistake. In production we'd want a
  PodDisruptionBudget enforcing `minAvailable: 1`.

## Action items

| ID | Description | Owner | Priority |
|---|---|---|---|
| AI-1 | Add HTTP client retry-with-backoff to orders-api → inventory-api calls | Madhan | High |
| AI-2 | Add a PodDisruptionBudget for inventory-api requiring `minAvailable: 1` | Madhan | Medium |
| AI-3 | Document graceful-degradation policy: should orders-api accept orders when stock check is unavailable? | Madhan + product | Medium |
| AI-4 | Tune readiness probe timing (consider `failureThreshold: 1` with shorter `periodSeconds`) | Madhan | Low |

## Notes for the curious

This was a deliberate failure injection, not a real incident. The exact
sequence (`kubectl delete pod -l ...`) is the simplest possible chaos
experiment — equivalent to "pull the cord". Future experiments scoped
for v2:

- Network partition between orders-api and inventory-api
- Slow responses from inventory-api (latency injection)
- inventory-api returning 500s consistently (error injection via feature flag)