package logger

import (
	"context"
	"testing"
)

func TestInitAndGet(t *testing.T) {
	Init("debug")
	if Get() == nil {
		t.Fatal("expected non-nil logger instance")
	}
}

func TestCtx(t *testing.T) {
	Init("info")
	ctx := context.Background()
	entry := Ctx(ctx)
	if entry == nil {
		t.Fatal("expected non-nil entry from Ctx(ctx)")
	}
}

func TestWithFields(t *testing.T) {
	Init("info")
	entry := WithFields(map[string]interface{}{"key": "val"})
	if entry == nil {
		t.Fatal("expected non-nil entry from WithFields")
	}
}
