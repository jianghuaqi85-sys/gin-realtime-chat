package main

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLoggingInterceptor_NoError(t *testing.T) {
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}

	resp, err := loggingInterceptor(context.Background(), "input", info, handler)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected 'ok', got %v", resp)
	}
}

func TestLoggingInterceptor_WithError(t *testing.T) {
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.InvalidArgument, "invalid argument")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}

	_, err := loggingInterceptor(context.Background(), "input", info, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestRecoveryInterceptor_Panic(t *testing.T) {
	handler := func(ctx context.Context, req any) (any, error) {
		panic("something went wrong")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/PanicMethod"}

	resp, err := recoveryInterceptor(context.Background(), "input", info, handler)
	if resp != nil {
		t.Errorf("expected nil response on panic, got %v", resp)
	}
	if err == nil {
		t.Fatal("expected error after panic recovery, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("expected Internal status code, got %v", st.Code())
	}
}
