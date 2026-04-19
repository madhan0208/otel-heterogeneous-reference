package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// -----------------------------------------------------------------------------
// MVS §1 — Resource attributes
// -----------------------------------------------------------------------------
func newResource() (*resource.Resource, error) {
	env := getenv("DEPLOYMENT_ENVIRONMENT", "dev")
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes("",
			semconv.ServiceName(getenv("OTEL_SERVICE_NAME", "inventory-api")),
			semconv.ServiceVersion(getenv("SERVICE_VERSION", "0.1.0")),
			semconv.ServiceNamespace("commerce"),
			attribute.String("deployment.environment", env),
		),
	)
}

// -----------------------------------------------------------------------------
// OpenTelemetry wiring — traces, metrics, logs over OTLP
// -----------------------------------------------------------------------------
func initOTel(ctx context.Context) (func(context.Context) error, error) {
	endpoint := getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	res, err := newResource()
	if err != nil {
		return nil, err
	}

	// Traces
	traceExp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Metrics
	metricExp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp,
			sdkmetric.WithInterval(60*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// Logs — MVS §4: structured, trace-correlated
	logExp, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)

	// Wire slog to OTel — logs written via slog automatically carry trace_id/span_id
	otelHandler := otelslog.NewHandler("inventory-api",
		otelslog.WithLoggerProvider(lp))
	// Wrap stdout handler with trace-context enrichment so local logs also get trace_id/span_id
	stdoutBase := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	stdoutHandler := &traceContextHandler{inner: stdoutBase}
	slog.SetDefault(slog.New(multiHandler{otelHandler, stdoutHandler}))

	return func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx), lp.Shutdown(ctx))
	}, nil
}

// -----------------------------------------------------------------------------
// HTTP handlers
// -----------------------------------------------------------------------------
type stockResp struct {
	ItemID    string `json:"itemId"`
	Available int    `json:"available"`
}

// MVS §2.1 — manual business span around the lookup
func stockHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer("inventory-api")
	ctx, span := tracer.Start(ctx, "inventory.lookup")
	defer span.End()

	itemID := r.PathValue("itemId")
	span.SetAttributes(attribute.String("inventory.item.id", itemID))

	// Simulated latency — real systems do a DB lookup here
	time.Sleep(time.Duration(10+rand.Intn(40)) * time.Millisecond)

	available := rand.Intn(20)
	slog.InfoContext(ctx, "stock lookup",
		"itemId", itemID,
		"available", available)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stockResp{ItemID: itemID, Available: available})
}

func healthz(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprint(w, `{"status":"ok"}`)
}

// -----------------------------------------------------------------------------
// main
// -----------------------------------------------------------------------------
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdown, err := initOTel(ctx)
	if err != nil {
		slog.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		sctx, scancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer scancel()
		_ = shutdown(sctx)
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /stock/{itemId}", stockHandler)

	// otelhttp wraps the whole mux — auto-instruments every request
	handler := otelhttp.NewHandler(mux, "inventory-api",
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents))

	srv := &http.Server{
		Addr:    ":8081",
		Handler: handler,
	}

	go func() {
		slog.Info("inventory-api starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	sctx, scancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer scancel()
	_ = srv.Shutdown(sctx)
}

// traceContextHandler wraps an slog.Handler and automatically adds
// trace_id and span_id from the active span in the context, so that
// logs printed to stdout (for Fluent Bit/Loki) are correlated with traces
// in exactly the same way as the OTel logs pipeline (MVS §4.2).
type traceContextHandler struct {
	inner slog.Handler
}

func (h *traceContextHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.inner.Enabled(ctx, lvl)
}

func (h *traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		sc := span.SpanContext()
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, r)
}

func (h *traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceContextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *traceContextHandler) WithGroup(name string) slog.Handler {
	return &traceContextHandler{inner: h.inner.WithGroup(name)}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// multiHandler fans a single slog.Record out to multiple handlers.
// Useful during local development where we want to see logs on stdout
// AND ship them via OTLP at the same time.
type multiHandler []slog.Handler

func (m multiHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, h := range m {
		if h.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (m multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range m {
		if err := h.Handle(ctx, r.Clone()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make(multiHandler, len(m))
	for i, h := range m {
		next[i] = h.WithAttrs(attrs)
	}
	return next
}

func (m multiHandler) WithGroup(name string) slog.Handler {
	next := make(multiHandler, len(m))
	for i, h := range m {
		next[i] = h.WithGroup(name)
	}
	return next
}
