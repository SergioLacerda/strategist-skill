package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otellog "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace/noop"
)

// Init initializes the global OTel providers from cfg.
// Returns a shutdown function that must be deferred before process exit.
// If cfg.Enabled() is false, installs noop providers — no network connections opened.
// The default slog handler is left untouched in this case: several commands
// (e.g. check-stale, install) emit their sole user-facing result line via
// slog.InfoContext with no fmt.Println companion, so discarding the default
// handler here would silently delete real output, not just duplicate noise.
func Init(cfg Config) (shutdown func(context.Context) error, err error) {
	if !cfg.Enabled() {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure()) //nolint:staticcheck // intentional for dev/self-hosted
	}

	exp, err := otlptracegrpc.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("telemetry: exporter: %w", err)
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Install a real log pipeline, then bridge slog to it. Without a
	// LoggerProvider the bridge writes to a no-op global provider and every
	// event (including the mission lifecycle and usage events) is dropped.
	lp, err := newLoggerProvider(cfg, res)
	if err != nil {
		_ = tp.Shutdown(context.Background()) //nolint:errcheck // best-effort cleanup on a failed setup
		return nil, err
	}
	otellog.SetLoggerProvider(lp)
	handler := otelslog.NewHandler(instrumentationName, otelslog.WithLoggerProvider(lp))
	slog.SetDefault(slog.New(handler))

	return func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), lp.Shutdown(ctx))
	}, nil
}

// newLoggerProvider builds the OTLP/gRPC log pipeline that carries slog records
// and the exported mission events to the same endpoint as the traces.
func newLoggerProvider(cfg Config, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	exp, err := otlploggrpc.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("telemetry: log exporter: %w", err)
	}
	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
		sdklog.WithResource(res),
	), nil
}
