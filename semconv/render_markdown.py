#!/usr/bin/env python3
"""Render Markdown documentation from the Weaver semantic convention registry.

Reads semconv/hugoboss.yaml and produces semconv/generated/attributes.md.
This complements the Weaver validation step: the YAML is the single source
of truth, validated by `weaver registry check`, and rendered to docs by
this script.

Run from the repository root:
    python3 semconv/render_markdown.py
"""

from pathlib import Path
import yaml

REPO_ROOT = Path(__file__).resolve().parent.parent
SCHEMA_PATH = REPO_ROOT / "semconv" / "hugoboss.yaml"
OUTPUT_PATH = REPO_ROOT / "semconv" / "generated" / "attributes.md"

GROUP_DESCRIPTIONS = {
    "registry.order": (
        "Order attributes",
        "Custom attributes for order-related operations. Emitted by orders-api.",
        "Application",
    ),
    "registry.inventory": (
        "Inventory attributes",
        "Custom attributes for inventory operations. Emitted by inventory-api.",
        "Application",
    ),
    "registry.payment": (
        "Payment attributes",
        "Custom attributes for payment processing. Emitted by orders-api.",
        "Application",
    ),
    "registry.app": (
        "Organizational attributes",
        "Added by the OpenTelemetry Collector via the k8sattributes processor. "
        "Services do NOT emit these directly — they are sourced from Kubernetes pod labels.",
        "Collector",
    ),
}


def render_type(attr_type):
    """Render the type field, which may be a string or a structured enum."""
    if isinstance(attr_type, str):
        return attr_type
    if isinstance(attr_type, dict) and "members" in attr_type:
        members = [m["value"] for m in attr_type["members"]]
        return f"enum: {', '.join(members)}"
    return str(attr_type)


def render_examples(examples):
    """Render the examples list."""
    if not examples:
        return ""
    return ", ".join(f"`{e}`" for e in examples)


def render_group(group):
    """Render one attribute group as a Markdown section."""
    group_id = group["id"]
    title, description, _ = GROUP_DESCRIPTIONS.get(
        group_id, (group_id, group.get("brief", ""), "Application")
    )

    out = [f"## {title}", "", description, ""]
    out.append("| Attribute | Type | Requirement | Stability | Brief |")
    out.append("|---|---|---|---|---|")

    for attr in group["attributes"]:
        attr_id = attr["id"]
        attr_type = render_type(attr["type"])
        req = attr.get("requirement_level", "")
        stab = attr.get("stability", "")
        brief = attr.get("brief", "").replace("\n", " ")
        out.append(f"| `{attr_id}` | {attr_type} | {req} | {stab} | {brief} |")

    return "\n".join(out)


def main():
    with open(SCHEMA_PATH, "r") as f:
        schema = yaml.safe_load(f)

    sections = [
        "# Hugo Boss custom telemetry attribute registry",
        "",
        "**Generated** from `semconv/hugoboss.yaml`. Do not edit by hand.",
        "Re-run `python3 semconv/render_markdown.py` to refresh after schema changes.",
        "",
        "The registry defines four attribute groups covering business-domain, "
        "infrastructure, and organizational concerns. The schema is validated by "
        "OpenTelemetry Weaver via `weaver registry check --registry=./semconv`.",
        "",
        "## Layer ownership",
        "",
        "| Group | Layer | Owner |",
        "|---|---|---|",
        "| `order.*` | Application code | Service developer |",
        "| `inventory.*` | Application code | Service developer |",
        "| `payment.*` | Application code | Service developer |",
        "| `app.*` | OTel Collector | Platform / SRE |",
        "",
    ]

    for group in schema["groups"]:
        sections.append(render_group(group))
        sections.append("")

    sections.extend([
        "## Stability",
        "",
        "All attributes are currently at stability level `development`. "
        "Promotion to `stable` requires field-testing and explicit team commitment. "
        "See the Telemetry Minimum Standard document for the governance process.",
        "",
    ])

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT_PATH.write_text("\n".join(sections))
    print(f"Wrote {OUTPUT_PATH.relative_to(REPO_ROOT)}")


if __name__ == "__main__":
    main()