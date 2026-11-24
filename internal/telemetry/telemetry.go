package telemetry

import (
	"context"
	"errors"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Start enables optional local OTel export, without a collector or network dependency.
// Payloads, tokens, repository paths, and evidence text are never span attributes.
func Start() (func(context.Context) error, error) {
	if os.Getenv("HOLOCRON_TELEMETRY_STDOUT") != "1" {
		return func(context.Context) error { return nil }, nil
	}
	spans, err := stdouttrace.New(stdouttrace.WithWriter(os.Stderr))
	if err != nil {
		return nil, err
	}
	metrics, err := stdoutmetric.New(stdoutmetric.WithWriter(os.Stderr))
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(trace.WithBatcher(spans))
	mp := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(metrics, metric.WithInterval(10*time.Second))))
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	return func(ctx context.Context) error { return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx)) }, nil
}
