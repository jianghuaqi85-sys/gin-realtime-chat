package otel

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const defaultServiceName = "gin-high-performance"

// InitTracer 初始化全局 TracerProvider 及 W3C TextMapPropagator
func InitTracer(ctx context.Context, endpoint string, insecure bool) (func(context.Context) error, error) {
	exporter, err := createExporter(ctx, endpoint, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(defaultServiceName),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.AlwaysSample())),
	)

	otel.SetTracerProvider(tp)

	// 注册全局 W3C 传播器（用于跨服务传递 TraceID/SpanID）
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func createExporter(ctx context.Context, endpoint string, insecure bool) (tracesdk.SpanExporter, error) {
	if endpoint == "" || endpoint == "stdout" {
		return stdouttrace.New(
			stdouttrace.WithWriter(os.Stdout),
			stdouttrace.WithPrettyPrint(),
		)
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
	}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	return otlptracehttp.New(ctx, opts...)
}

func GetTracer() trace.Tracer {
	return otel.Tracer(defaultServiceName)
}
