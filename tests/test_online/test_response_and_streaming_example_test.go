package online_test

import (
	"context"
	"testing"
	"time"

	"github.com/AgentEra/Agently-Go/agently/types"
)

func collectAnyLimited(t *testing.T, ctx context.Context, ch <-chan any, timeout time.Duration, limit int) []any {
	t.Helper()
	deadline := time.After(timeout)
	out := make([]any, 0)
	for {
		if limit > 0 && len(out) >= limit {
			return out
		}
		select {
		case item, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, item)
		case <-deadline:
			t.Fatalf("collect timeout after %s", timeout)
		case <-ctx.Done():
			t.Fatalf("context canceled while collecting stream: %v", ctx.Err())
		}
	}
}

func TestBasicResponseAndStreamingModes(t *testing.T) {
	main := newOnlineMain(t)

	{
		req := main.CreateRequest("example-delta")
		req.Input("Explain Go goroutines in one sentence.")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		stream, err := req.GetGenerator(ctx, "delta")
		if err != nil {
			t.Fatalf("delta stream failed: %v", err)
		}
		items := collectAnyLimited(t, ctx, stream, 90*time.Second, 120)
		if len(items) == 0 {
			t.Fatalf("expected non-empty delta stream")
		}
	}

	{
		req := main.CreateRequest("example-specific")
		req.Input("Explain Go channels in one sentence.")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		stream, err := req.GetGenerator(ctx, "specific", []string{"delta", "reasoning_delta", "tool_calls"})
		if err != nil {
			t.Fatalf("specific stream failed: %v", err)
		}
		items := collectAnyLimited(t, ctx, stream, 90*time.Second, 120)
		if len(items) == 0 {
			t.Fatalf("expected non-empty specific stream")
		}
		foundSpecific := false
		for _, item := range items {
			msg, ok := item.(types.ResponseMessage)
			if !ok {
				continue
			}
			if msg.Event == types.ResponseEventDelta || msg.Event == types.ResponseEventReasoning || msg.Event == types.ResponseEventToolCalls {
				foundSpecific = true
				break
			}
		}
		if !foundSpecific {
			t.Fatalf("specific stream did not include expected events: %#v", items)
		}
	}

	{
		req := main.CreateRequest("example-instant")
		req.Input("Output JSON with steps array and summary field.")
		req.Output(map[string]any{
			"steps":   []any{"string"},
			"summary": "string",
		})

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		stream, err := req.GetGenerator(ctx, "instant")
		if err != nil {
			t.Fatalf("instant stream failed: %v", err)
		}
		items := collectAnyLimited(t, ctx, stream, 90*time.Second, 180)
		if len(items) == 0 {
			t.Fatalf("expected non-empty instant stream")
		}
		foundPath := false
		for _, item := range items {
			evt, ok := item.(types.StreamingData)
			if !ok {
				continue
			}
			if evt.Path != "" {
				foundPath = true
				break
			}
		}
		if !foundPath {
			t.Fatalf("instant stream should contain at least one path event: %#v", items)
		}
	}
}
