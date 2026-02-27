package utils_test

import (
	"context"
	"testing"
	"time"

	"github.com/AgentEra/Agently-Go/agently/utils"
)

func TestRuntimeDataAndSettingsSemantic(t *testing.T) {
	parent := utils.NewRuntimeData("parent", map[string]any{"a": map[string]any{"x": 1}}, nil)
	child := utils.NewRuntimeData("child", map[string]any{"a": map[string]any{"y": 2}}, parent)

	if got := child.Get("a.x", nil, true); got != 1 {
		t.Fatalf("expected inherited value a.x=1, got %#v", got)
	}
	if got := child.Get("a.y", nil, true); got != 2 {
		t.Fatalf("expected child value a.y=2, got %#v", got)
	}

	settings := utils.NewSettings("settings", map[string]any{}, nil)
	settings.UpdateMappings(map[string]any{
		"path_mappings": map[string]any{
			"OpenAICompatible": "plugins.ModelRequester.OpenAICompatible",
		},
	})
	settings.SetSettings("OpenAICompatible", map[string]any{"model": "qwen2.5:7b"}, false)
	if got := settings.Get("plugins.ModelRequester.OpenAICompatible.model", nil, true); got != "qwen2.5:7b" {
		t.Fatalf("expected path mapping write-through, got %#v", got)
	}
}

func TestFunctionShifterFutureAndAwait(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	future := utils.Future(ctx, func(_ context.Context) (int, error) {
		return 42, nil
	})
	value, err := utils.Await(ctx, future)
	if err != nil {
		t.Fatalf("Await failed: %v", err)
	}
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}
}

func TestGeneratorConsumerReplay(t *testing.T) {
	source := make(chan any, 3)
	source <- "a"
	source <- "b"
	close(source)

	consumer := utils.NewGeneratorConsumer(source)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sub1, err := consumer.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe sub1 failed: %v", err)
	}
	items1 := make([]any, 0)
	for item := range sub1 {
		items1 = append(items1, item)
	}
	if len(items1) != 2 {
		t.Fatalf("expected sub1 replay size 2, got %#v", items1)
	}

	sub2, err := consumer.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe sub2 failed: %v", err)
	}
	items2 := make([]any, 0)
	for item := range sub2 {
		items2 = append(items2, item)
	}
	if len(items2) != 2 {
		t.Fatalf("expected sub2 replay size 2, got %#v", items2)
	}
}

func TestStreamingJSONParserSemantic(t *testing.T) {
	parser := utils.NewStreamingJSONParser(map[string]any{
		"answer": "string",
		"steps":  []any{"string"},
	})

	events1, err := parser.ParseChunk(`{"answer":"hel`)
	if err != nil {
		t.Fatalf("ParseChunk 1 failed: %v", err)
	}
	events2, err := parser.ParseChunk(`lo","steps":["a","b"]}`)
	if err != nil {
		t.Fatalf("ParseChunk 2 failed: %v", err)
	}
	final := parser.Finalize()

	if len(events1)+len(events2)+len(final) == 0 {
		t.Fatalf("expected non-empty streaming parser events")
	}
	foundAnswer := false
	for _, evt := range append(append(events1, events2...), final...) {
		if evt.Path == "answer" {
			foundAnswer = true
			break
		}
	}
	if !foundAnswer {
		t.Fatalf("expected answer path in streaming parser events")
	}
}
