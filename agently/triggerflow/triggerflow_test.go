package triggerflow

import (
	"context"
	"testing"
	"time"
)

func TestTriggerFlowSignalDrivenStart(t *testing.T) {
	flow := New(nil, "test")
	flow.When("START", "").To(Handler(func(data *EventData) (any, error) {
		if v, ok := data.Value.(int); ok {
			return v + 1, nil
		}
		return data.Value, nil
	}), WithToName("inc")).End()

	result, err := flow.Start(1, WithRunTimeout(3*time.Second))
	if err != nil {
		t.Fatalf("start error: %v", err)
	}
	if result != 2 {
		t.Fatalf("expected 2 got %#v", result)
	}
}

func TestTriggerFlowFlowDataSignal(t *testing.T) {
	flow := New(nil, "flow-data")
	flow.When(map[TriggerType][]string{TriggerTypeFlowData: []string{"flag"}}, "").To(Handler(func(data *EventData) (any, error) {
		data.execution.SetResult(data.Value)
		return data.Value, nil
	}), WithToName("capture"))

	exec := flow.CreateExecution()
	if err := flow.SetFlowData("flag", "ok", true); err != nil {
		t.Fatalf("set flow data error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := exec.GetResult(ctx)
	if err != nil {
		t.Fatalf("get result error: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected ok got %#v", result)
	}
}
