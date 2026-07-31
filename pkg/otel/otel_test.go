package otel

import (
	"context"
	"testing"
	"time"

	globalotel "go.opentelemetry.io/otel"
)

func TestInitTracerAndGetTracer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	shutdown, err := InitTracer(ctx, "stdout", true)
	if err != nil {
		t.Fatalf("failed to init tracer: %v", err)
	}
	defer shutdown(context.Background())

	tr := GetTracer()
	if tr == nil {
		t.Fatal("expected non-nil Tracer from GetTracer()")
	}

	prop := globalotel.GetTextMapPropagator()
	if prop == nil {
		t.Fatal("expected non-nil TextMapPropagator after InitTracer")
	}
}
