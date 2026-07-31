package grpcclient

import (
	"testing"
)

func TestNewClientAndConn(t *testing.T) {
	cli, err := NewClient("localhost:50051")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer cli.Close()

	if cli.Conn() == nil {
		t.Fatal("expected non-nil Conn() from client")
	}
}
