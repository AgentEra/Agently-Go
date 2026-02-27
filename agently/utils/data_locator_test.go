package utils

import "testing"

func TestLocatePathInData(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name": "alice",
			"items": []any{
				map[string]any{"id": 1},
				map[string]any{"id": 2},
			},
		},
	}
	if got := LocatePathInData(data, "user.name", "dot", nil); got != "alice" {
		t.Fatalf("expected alice got %#v", got)
	}
	if got := LocatePathInData(data, "/user/items/1/id", "slash", nil); got != 2 {
		t.Fatalf("expected 2 got %#v", got)
	}
	wild := LocatePathInData(data, "user.items[*].id", "dot", nil)
	arr, ok := wild.([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("expected wildcard array got %#v", wild)
	}
}

func TestLocateAllJSON(t *testing.T) {
	input := `before {"a":1} middle {"b":2}`
	all := LocateAllJSON(input)
	if len(all) != 2 {
		t.Fatalf("expected 2 json blocks got %d %#v", len(all), all)
	}
}
