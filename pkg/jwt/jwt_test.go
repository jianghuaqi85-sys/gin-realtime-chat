package jwt

import (
	"errors"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	_, err := NewManager("", "app")
	if err == nil {
		t.Error("expected error for empty secret, got nil")
	}

	mgr, err := NewManager("secret123456789012345678901234567890", "app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil Manager")
	}
}

func TestManager_GenerateAndParse(t *testing.T) {
	mgr, _ := NewManager("secret123456789012345678901234567890", "gin-app")

	tokenStr, err := mgr.GenerateToken("u1", "alice", "admin", 1)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := mgr.ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if claims.UserID != "u1" || claims.Username != "alice" || claims.Role != "admin" {
		t.Errorf("unexpected claims: %+v", claims)
	}

	if claims.ID == "" {
		t.Error("expected non-empty JTI (ID) in claims")
	}
	if claims.Issuer != "gin-app" {
		t.Errorf("expected issuer 'gin-app', got '%s'", claims.Issuer)
	}
}

func TestManager_ExpiredToken(t *testing.T) {
	mgr, _ := NewManager("secret123456789012345678901234567890", "gin-app")

	tokenStr, _ := mgr.GenerateToken("u1", "alice", "admin", -1)

	time.Sleep(10 * time.Millisecond)

	_, err := mgr.ParseToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}
