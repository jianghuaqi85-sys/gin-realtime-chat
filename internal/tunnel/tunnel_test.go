package tunnel

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewManagerAndGetURL(t *testing.T) {
	m := NewManager("invalid-path-cloudflared", "8080", ".test_tunnel_url")
	if m.GetURL() != "" {
		t.Errorf("expected empty initial URL, got %s", m.GetURL())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := m.Start(ctx)
	if err == nil {
		t.Fatal("expected error for invalid binary path, got nil")
	}

	m.Stop()
	if _, err := os.Stat(".test_tunnel_url"); !os.IsNotExist(err) {
		t.Error("expected urlFile to be removed by Stop")
	}
}
