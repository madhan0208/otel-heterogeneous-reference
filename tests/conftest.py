import threading
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
from opentelemetry.proto.collector.trace.v1.trace_service_pb2 import ExportTraceServiceRequest

import pytest

captured = []

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get('Content-Length'))
        body = self.rfile.read(length)

        req = ExportTraceServiceRequest()
        req.ParseFromString(body)
        captured.append(req)

        self.send_response(200)
        self.end_headers()

    def log_message(self, fmt, *args):
        pass  # stop printing request logs into pytest output

@pytest.fixture()
def otlp_capture():
    captured.clear()
    server = HTTPServer(('localhost', 4319), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    time.sleep(0.1)
    yield captured
    server.shutdown()