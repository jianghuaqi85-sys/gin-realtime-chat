package limiter

import (
	"testing"
)

func TestGenerateRandomString(t *testing.T) {
	str1 := generateRandomString(4)
	str2 := generateRandomString(4)

	if len(str1) != 8 { // 4 bytes in hex string is 8 chars
		t.Errorf("expected hex length 8 for 4 bytes, got %d (%s)", len(str1), str1)
	}

	if str1 == str2 {
		t.Errorf("expected different random strings, got %s and %s", str1, str2)
	}
}

func TestNewLimiter(t *testing.T) {
	lim := NewLimiter(nil)
	if lim == nil {
		t.Fatal("expected non-nil Limiter")
	}
}
