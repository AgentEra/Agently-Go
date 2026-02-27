package responseparser_test

import (
	"context"
	"testing"
	"time"

	entry "github.com/AgentEra/Agently-Go/agently"
	rp "github.com/AgentEra/Agently-Go/agently/builtins/plugins/response_parser"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func collectAny(t *testing.T, ch <-chan any, timeout time.Duration) []any {
	t.Helper()
	out := make([]any, 0)
	deadline := time.After(timeout)
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, item)
		case <-deadline:
			t.Fatalf("collect timeout after %s", timeout)
		}
	}
}

func buildParser(t *testing.T) *rp.AgentlyResponseParser {
	t.Helper()
	main := entry.NewAgently()
	req := main.CreateRequest("response-parser-semantic")
	req.Input("stream semantic")
	req.Output(map[string]any{"answer": "string", "status": "string"})

	response := make(chan types.ResponseMessage, 8)
	response <- types.ResponseMessage{Event: types.ResponseEventOriginalDelta, Data: `{"id":"x"}`}
	response <- types.ResponseMessage{Event: types.ResponseEventDelta, Data: `{"answer":"he`}
	response <- types.ResponseMessage{Event: types.ResponseEventToolCalls, Data: []any{map[string]any{"name": "sum"}}}
	response <- types.ResponseMessage{Event: types.ResponseEventDelta, Data: `llo","status":"ok"}`}
	response <- types.ResponseMessage{Event: types.ResponseEventDone, Data: `{"answer":"hello","status":"ok"}`}
	response <- types.ResponseMessage{Event: types.ResponseEventOriginalDone, Data: map[string]any{"done": true}}
	close(response)

	parserIface := rp.New("agent", "resp", req.Prompt(), response, req.Settings())
	parser, ok := parserIface.(*rp.AgentlyResponseParser)
	if !ok {
		t.Fatalf("expected AgentlyResponseParser, got %T", parserIface)
	}
	return parser
}

func TestResponseParserStreamTypes(t *testing.T) {
	ctx1, cancel1 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel1()
	parser := buildParser(t)
	delta, err := parser.GetStream(ctx1, "delta", nil)
	if err != nil {
		t.Fatalf("GetStream(delta) failed: %v", err)
	}
	deltaItems := collectAny(t, delta, 3*time.Second)
	if len(deltaItems) == 0 {
		t.Fatalf("expected delta items")
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	parser = buildParser(t)
	specific, err := parser.GetStream(ctx2, "specific", []string{"tool_calls"})
	if err != nil {
		t.Fatalf("GetStream(specific) failed: %v", err)
	}
	specificItems := collectAny(t, specific, 3*time.Second)
	if len(specificItems) == 0 {
		t.Fatalf("expected specific(tool_calls) items")
	}
	if msg, ok := specificItems[0].(types.ResponseMessage); !ok || msg.Event != types.ResponseEventToolCalls {
		t.Fatalf("unexpected specific item: %#v", specificItems[0])
	}

	ctx3, cancel3 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel3()
	parser = buildParser(t)
	instant, err := parser.GetStream(ctx3, "instant", nil)
	if err != nil {
		t.Fatalf("GetStream(instant) failed: %v", err)
	}
	instantItems := collectAny(t, instant, 3*time.Second)
	if len(instantItems) == 0 {
		t.Fatalf("expected instant items")
	}
	foundAnswerPath := false
	for _, item := range instantItems {
		evt, ok := item.(types.StreamingData)
		if !ok {
			continue
		}
		if evt.Path == "answer" {
			foundAnswerPath = true
			break
		}
	}
	if !foundAnswerPath {
		t.Fatalf("expected instant answer path event, got %#v", instantItems)
	}
}
