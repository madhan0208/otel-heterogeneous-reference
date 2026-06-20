# Hugo Boss custom telemetry attribute registry

**Generated** from `semconv/hugoboss.yaml`. Do not edit by hand.
Re-run `python3 semconv/render_markdown.py` to refresh after schema changes.

The registry defines four attribute groups covering business-domain, infrastructure, and organizational concerns. The schema is validated by OpenTelemetry Weaver via `weaver registry check --registry=./semconv`.

## Layer ownership

| Group | Layer | Owner |
|---|---|---|
| `order.*` | Application code | Service developer |
| `inventory.*` | Application code | Service developer |
| `payment.*` | Application code | Service developer |
| `app.*` | OTel Collector | Platform / SRE |

## Order attributes

Custom attributes for order-related operations. Emitted by orders-api.

| Attribute | Type | Requirement | Stability | Brief |
|---|---|---|---|---|
| `order.channel` | enum: web, mobile, store | required | development | The channel through which the order was placed. |
| `order.brand_line` | enum: boss, hugo, boss_orange | required | development | The brand line of the ordered item. |
| `order.value_eur` | double | recommended | development | Total order value in EUR. Use only as span attribute, never as metric label (high cardinality). |
| `order.customer_tier` | enum: standard, vip | recommended | development | Customer tier influencing SLA expectations. |

## Inventory attributes

Custom attributes for inventory operations. Emitted by inventory-api.

| Attribute | Type | Requirement | Stability | Brief |
|---|---|---|---|---|
| `inventory.warehouse` | string | required | development | Identifier of the warehouse fulfilling the inventory lookup. |
| `inventory.region` | enum: DACH, EMEA, APAC | recommended | development | Geographic region of the warehouse. |

## Payment attributes

Custom attributes for payment processing. Emitted by orders-api.

| Attribute | Type | Requirement | Stability | Brief |
|---|---|---|---|---|
| `payment.processor` | enum: mock, stripe, adyen | recommended | development | The payment processor handling the transaction. |

## Organizational attributes

Added by the OpenTelemetry Collector via the k8sattributes processor. Services do NOT emit these directly — they are sourced from Kubernetes pod labels.

| Attribute | Type | Requirement | Stability | Brief |
|---|---|---|---|---|
| `app.team` | string | required | development | Team owning the service. Sourced from Kubernetes pod label by Collector k8sattributes processor. |
| `app.cost_center` | string | required | development | Cost center for chargeback. Sourced from Kubernetes pod label by Collector. |

## Stability

All attributes are currently at stability level `development`. Promotion to `stable` requires field-testing and explicit team commitment. See the Telemetry Minimum Standard document for the governance process.
