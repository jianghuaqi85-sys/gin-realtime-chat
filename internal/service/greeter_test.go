package service

import (
	"context"
	"strings"
	"testing"

	pb "github.com/example/gin-high-performance/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSayHello_NormalAndDefault(t *testing.T) {
	svc := NewGreeterService()

	res1, err := svc.SayHello(context.Background(), &pb.HelloRequest{Name: "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.Message != "Hello, Alice!" {
		t.Errorf("expected 'Hello, Alice!', got '%s'", res1.Message)
	}

	res2, err := svc.SayHello(context.Background(), &pb.HelloRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Message != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got '%s'", res2.Message)
	}
}

func TestSayHello_NameTooLong(t *testing.T) {
	svc := NewGreeterService()
	longName := strings.Repeat("a", 101)

	_, err := svc.SayHello(context.Background(), &pb.HelloRequest{Name: longName})
	if err == nil {
		t.Fatal("expected error for long name, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument code, got %v", st.Code())
	}
}

func TestSayHello_ContextCanceled(t *testing.T) {
	svc := NewGreeterService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel context immediately

	_, err := svc.SayHello(ctx, &pb.HelloRequest{Name: "Bob"})
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Canceled {
		t.Fatalf("expected Canceled code, got %v", st.Code())
	}
}
