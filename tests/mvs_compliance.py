import re
import requests
from opentelemetry.proto.collector.trace.v1.trace_service_pb2 import ExportTraceServiceRequest

def get_attr(resource, key):
    for attr in resource.attributes:
        if attr.key == key:
            return attr.value.string_value
    return None

def _send_test_span(env="dev"):
    req = ExportTraceServiceRequest()
    rs = req.resource_spans.add()

    a1 = rs.resource.attributes.add(key="service.name")
    a1.value.string_value = "orders-api"
    a2 = rs.resource.attributes.add(key="service.version")
    a2.value.string_value = "1.4.2"
    a3 = rs.resource.attributes.add(key="deployment.environment")
    a3.value.string_value = "production"

    scope_span = rs.scope_spans.add()
    span = scope_span.spans.add()
    span.name = "POST /orders"

    requests.post("http://localhost:4319", data=req.SerializeToString())

def test_span_naming(otlp_capture):
    _send_test_span()
    req = otlp_capture[0]
    span_name = req.resource_spans[0].scope_spans[0].spans[0].name
    method, path = span_name.split(" ", 1)
    assert method in {"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
    assert path.startswith("/")

def test_service_name_kebab_case(otlp_capture):
    _send_test_span()
    req = otlp_capture[0]
    service_name = get_attr(req.resource_spans[0].resource, "service.name")
    assert re.fullmatch(r"[a-z][a-z0-9-]*", service_name)

def test_service_version_semver(otlp_capture):
    _send_test_span()
    req = otlp_capture[0]
    version = get_attr(req.resource_spans[0].resource, "service.version")
    assert re.fullmatch(r"\d+\.\d+\.\d+", version)

def test_deployment_environment_valid(otlp_capture):
    _send_test_span()
    req = otlp_capture[0]
    env = get_attr(req.resource_spans[0].resource, "deployment.environment")
    assert env in {"dev", "staging", "prod"}