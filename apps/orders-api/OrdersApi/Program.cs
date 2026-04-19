using System.Diagnostics;
using OpenTelemetry.Logs;
using OpenTelemetry.Metrics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;
using Serilog;
using Serilog.Formatting.Compact;

var builder = WebApplication.CreateBuilder(args);

// ---------------------------------------------------------------------------
// MVS §1 — Resource attributes (injected via env vars, with sensible defaults)
// ---------------------------------------------------------------------------
var serviceName = Environment.GetEnvironmentVariable("OTEL_SERVICE_NAME") ?? "orders-api";
var serviceVersion = Environment.GetEnvironmentVariable("SERVICE_VERSION") ?? "0.1.0";
var environment = Environment.GetEnvironmentVariable("DEPLOYMENT_ENVIRONMENT") ?? "dev";

var resource = ResourceBuilder.CreateDefault()
    .AddService(serviceName: serviceName, serviceVersion: serviceVersion)
    .AddAttributes(new KeyValuePair<string, object>[]
    {
        new("service.namespace", "commerce"),
        new("deployment.environment", environment),
    });

// ---------------------------------------------------------------------------
// MVS §4 — Structured JSON logging with trace/span correlation
// ---------------------------------------------------------------------------
Log.Logger = new LoggerConfiguration()
    .Enrich.FromLogContext()
    .WriteTo.Console(new CompactJsonFormatter())
    .CreateLogger();
builder.Host.UseSerilog();

// ---------------------------------------------------------------------------
// MVS §2.1 — Manual business spans via a dedicated ActivitySource
// ---------------------------------------------------------------------------
var activitySource = new ActivitySource("orders-api");
builder.Services.AddSingleton(activitySource);

// ---------------------------------------------------------------------------
// OpenTelemetry wiring — traces, metrics, logs to OTLP
// ---------------------------------------------------------------------------
var otlpEndpoint = Environment.GetEnvironmentVariable("OTEL_EXPORTER_OTLP_ENDPOINT")
    ?? "http://localhost:4317";

builder.Services.AddOpenTelemetry()
    .WithTracing(t => t
        .SetResourceBuilder(resource)
        .AddSource("orders-api")
        .AddAspNetCoreInstrumentation()
        .AddHttpClientInstrumentation()
        .AddOtlpExporter(o => o.Endpoint = new Uri(otlpEndpoint)))
    .WithMetrics(m => m
        .SetResourceBuilder(resource)
        .AddAspNetCoreInstrumentation()
        .AddHttpClientInstrumentation()
        .AddRuntimeInstrumentation()
        .AddOtlpExporter(o => o.Endpoint = new Uri(otlpEndpoint)));

builder.Logging.AddOpenTelemetry(o =>
{
    o.SetResourceBuilder(resource);
    o.AddOtlpExporter(e => e.Endpoint = new Uri(otlpEndpoint));
    o.IncludeScopes = true;
    o.IncludeFormattedMessage = true;
});

// ---------------------------------------------------------------------------
// HTTP client for the downstream inventory-api
// ---------------------------------------------------------------------------
builder.Services.AddHttpClient("inventory", client =>
{
    var baseUrl = Environment.GetEnvironmentVariable("INVENTORY_API_URL")
        ?? "http://localhost:8081";
    client.BaseAddress = new Uri(baseUrl);
});

var app = builder.Build();

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

// Liveness / readiness probe (Kubernetes will hit this later)
app.MapGet("/healthz", () => Results.Ok(new { status = "ok" }));

// POST /orders — the main business endpoint.
//   1. Call inventory-api to check stock
//   2. Simulate a payment authorisation (5% random failure rate)
//   3. Return the created order
app.MapPost("/orders", async (
    OrderRequest req,
    IHttpClientFactory httpFactory,
    ActivitySource src,
    ILogger<Program> log) =>
{
    using var activity = src.StartActivity("orders.create");
    activity?.SetTag("orders.item_id", req.ItemId);
    activity?.SetTag("orders.qty", req.Qty);

    log.LogInformation("Received order for item {ItemId} qty {Qty}", req.ItemId, req.Qty);

    // Call inventory — the HttpClient is auto-instrumented, propagates trace context
    var client = httpFactory.CreateClient("inventory");
    var stockResp = await client.GetAsync($"/stock/{req.ItemId}");
    if (!stockResp.IsSuccessStatusCode)
    {
        activity?.SetStatus(ActivityStatusCode.Error, "inventory lookup failed");
        log.LogWarning("Inventory lookup failed for {ItemId}: {Status}",
            req.ItemId, (int)stockResp.StatusCode);
        return Results.Problem("inventory unavailable", statusCode: 502);
    }

    // Payment authorisation — manual span for the business step
    using (var payActivity = src.StartActivity("orders.authorize_payment"))
    {
        await Task.Delay(Random.Shared.Next(20, 80));
        if (Random.Shared.NextDouble() < 0.05)
        {
            payActivity?.SetStatus(ActivityStatusCode.Error, "payment declined");
            log.LogWarning("Payment declined for {ItemId}", req.ItemId);
            return Results.Problem("payment declined", statusCode: 402);
        }
    }

    var orderId = Guid.NewGuid().ToString();
    log.LogInformation("Order {OrderId} created", orderId);
    return Results.Ok(new { orderId, req.ItemId, req.Qty });
});

app.Run();

record OrderRequest(string ItemId, int Qty);