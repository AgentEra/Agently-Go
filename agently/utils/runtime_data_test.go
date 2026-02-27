package utils

import "testing"

func TestRuntimeDataInheritanceAndDotPath(t *testing.T) {
	parent := NewRuntimeData("parent", map[string]any{
		"a": map[string]any{"b": 1, "c": []any{"x"}},
	}, nil)
	child := NewRuntimeData("child", map[string]any{
		"a": map[string]any{"c": []any{"y"}},
	}, parent)

	if got := child.Get("a.b", nil, true); got != 1 {
		t.Fatalf("expected inherited a.b=1, got %#v", got)
	}
	list, ok := child.Get("a.c", nil, true).([]any)
	if !ok {
		t.Fatalf("expected list, got %T", child.Get("a.c", nil, true))
	}
	if len(list) != 2 {
		t.Fatalf("expected merged list len=2, got %d: %#v", len(list), list)
	}

	child.Set("a.d", 2)
	if got := child.Get("a.d", nil, true); got != 2 {
		t.Fatalf("expected a.d=2, got %#v", got)
	}
}

func TestRuntimeDataAppendExtend(t *testing.T) {
	r := NewRuntimeData("r", map[string]any{}, nil)
	r.Append("x", "a")
	r.Append("x", "b")
	r.Extend("x", []any{"c", "d"})
	list, ok := r.Get("x", nil, false).([]any)
	if !ok {
		t.Fatalf("expected list, got %T", r.Get("x", nil, false))
	}
	if len(list) != 4 {
		t.Fatalf("expected len=4 got %d %#v", len(list), list)
	}
}
