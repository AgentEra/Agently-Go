package utils

import (
	"testing"

	"github.com/AgentEra/Agently-Go/agently/types"
)

func TestStreamingJSONParserParseChunkAndFinalize(t *testing.T) {
	schema := map[string]any{"answer": "string"}
	parser := NewStreamingJSONParser(schema)

	events, err := parser.ParseChunk(`{"answer":"he`)
	if err != nil {
		t.Fatalf("parse chunk error: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("expected delta event from first chunk")
	}

	events2, err := parser.ParseChunk(`llo"}`)
	if err != nil {
		t.Fatalf("parse chunk2 error: %v", err)
	}
	_ = events2

	final := parser.Finalize()
	if len(final) == 0 {
		t.Fatalf("expected finalize done events")
	}
	foundDone := false
	for _, evt := range final {
		if evt.Path == "answer" && evt.EventType == types.StreamEventDone {
			foundDone = true
		}
	}
	if !foundDone {
		t.Fatalf("expected answer done event, got %#v", final)
	}
}
