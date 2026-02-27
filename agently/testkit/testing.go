package testkit

import (
	"context"
	"testing"
	"time"
)

func TestContext(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

func CollectAny(t *testing.T, ch <-chan any, timeout time.Duration) []any {
	t.Helper()
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	deadline := time.After(timeout)
	items := make([]any, 0)
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				return items
			}
			items = append(items, item)
		case <-deadline:
			t.Fatalf("collect stream timeout after %s", timeout)
		}
	}
}
