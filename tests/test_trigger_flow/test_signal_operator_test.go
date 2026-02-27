package triggerflow_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/AgentEra/Agently-Go/agently/triggerflow"
)

func TestStartToEndChain(t *testing.T) {
	flow := triggerflow.New(nil, "start-end")
	flow.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		if v, ok := data.Value.(int); ok {
			return v + 1, nil
		}
		return data.Value, nil
	}), false, "inc").End()

	result, err := flow.Start(1, triggerflow.WithRunTimeout(3*time.Second))
	if err != nil {
		t.Fatalf("start chain failed: %v", err)
	}
	if result != 2 {
		t.Fatalf("expected result=2, got %#v", result)
	}
}

func TestWhenSignalsAndEmitLoop(t *testing.T) {
	t.Run("and_runtime_event", func(t *testing.T) {
		flow := triggerflow.New(nil, "when-and")
		flow.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			_ = data.SetRuntimeData("ready", "yes", true)
			_ = data.Emit("TRIGGER_A", "A", triggerflow.TriggerTypeEvent)
			return nil, nil
		}), false, "emit-signals")

		flow.When(map[triggerflow.TriggerType][]string{
			triggerflow.TriggerTypeRuntimeData: {"ready"},
			triggerflow.TriggerTypeEvent:       {"TRIGGER_A"},
		}, "and").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "capture-and")

		result, err := flow.Start(nil, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("and mode failed: %v", err)
		}
		values, ok := result.(map[triggerflow.TriggerType]map[string]any)
		if !ok {
			t.Fatalf("expected and result map, got %T (%#v)", result, result)
		}
		if values[triggerflow.TriggerTypeRuntimeData]["ready"] != "yes" {
			t.Fatalf("unexpected runtime_data capture: %#v", values)
		}
		if values[triggerflow.TriggerTypeEvent]["TRIGGER_A"] != "A" {
			t.Fatalf("unexpected event capture: %#v", values)
		}
	})

	t.Run("or", func(t *testing.T) {
		flow := triggerflow.New(nil, "when-or")
		flow.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			return nil, data.Emit("TRIGGER_B", 7, triggerflow.TriggerTypeEvent)
		}), false, "emit-b")
		flow.When(map[triggerflow.TriggerType][]string{triggerflow.TriggerTypeEvent: {"TRIGGER_A", "TRIGGER_B"}}, "or").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "capture-or")

		result, err := flow.Start(nil, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("or mode failed: %v", err)
		}
		payload, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected or payload map, got %T (%#v)", result, result)
		}
		if payload["trigger_event"] != "TRIGGER_B" {
			t.Fatalf("unexpected trigger event in or payload: %#v", payload)
		}
		if payload["value"] != 7 {
			t.Fatalf("unexpected or value: %#v", payload)
		}
	})

	t.Run("simple_or", func(t *testing.T) {
		flow := triggerflow.New(nil, "when-simple-or")
		flow.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			return nil, data.Emit("TRIGGER_X", "simple", triggerflow.TriggerTypeEvent)
		}), false, "emit-simple")
		flow.When(map[triggerflow.TriggerType][]string{triggerflow.TriggerTypeEvent: {"TRIGGER_X", "TRIGGER_Y"}}, "simple_or").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "capture-simple-or")

		result, err := flow.Start(nil, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("simple_or mode failed: %v", err)
		}
		if result != "simple" {
			t.Fatalf("unexpected simple_or result: %#v", result)
		}
	})

	t.Run("flow_data_signal", func(t *testing.T) {
		flow := triggerflow.New(nil, "flow-data-signal")
		flow.When(map[triggerflow.TriggerType][]string{triggerflow.TriggerTypeFlowData: {"flag"}}, "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "flow-capture")

		exec := flow.CreateExecution()
		if err := flow.SetFlowData("flag", "ready", true); err != nil {
			t.Fatalf("set flow data failed: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		result, err := exec.GetResult(ctx)
		if err != nil {
			t.Fatalf("flow-data signal get result failed: %v", err)
		}
		if result != "ready" {
			t.Fatalf("unexpected flow-data signal result: %#v", result)
		}
	})

	t.Run("emit_loop", func(t *testing.T) {
		flow := triggerflow.New(nil, "emit-loop")
		flow.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			return nil, data.Emit("PING", 1, triggerflow.TriggerTypeEvent)
		}), false, "kickoff")

		flow.When("PING", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			n, _ := data.Value.(int)
			if n < 3 {
				return n, data.Emit("PING", n+1, triggerflow.TriggerTypeEvent)
			}
			data.SetResult(n)
			return n, nil
		}), false, "loop")

		result, err := flow.Start(nil, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("emit loop failed: %v", err)
		}
		if result != 3 {
			t.Fatalf("unexpected emit loop result: %#v", result)
		}
	})
}

func TestOperatorsBatchForEachMatchCollectSideBranch(t *testing.T) {
	t.Run("batch", func(t *testing.T) {
		flow := triggerflow.New(nil, "batch")
		flow.When("START", "").Batch([]any{
			triggerflow.NamedChunk{Name: "left", Handler: func(data *triggerflow.EventData) (any, error) { return fmt.Sprintf("%v-L", data.Value), nil }},
			triggerflow.NamedChunk{Name: "right", Handler: func(data *triggerflow.EventData) (any, error) { return fmt.Sprintf("%v-R", data.Value), nil }},
		}, false, 2).To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "batch-result")

		result, err := flow.Start("X", triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("batch failed: %v", err)
		}
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected batch result map, got %T (%#v)", result, result)
		}
		if resultMap["left"] == nil || resultMap["right"] == nil {
			t.Fatalf("batch map missing keys: %#v", resultMap)
		}
	})

	t.Run("for_each", func(t *testing.T) {
		flow := triggerflow.New(nil, "for-each")
		flow.When("START", "").ForEach(2).To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			if v, ok := data.Value.(int); ok {
				return v * 2, nil
			}
			return data.Value, nil
		}), false, "double").EndForEach().To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "foreach-result")

		result, err := flow.Start([]any{1, 2, 3}, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("for_each failed: %v", err)
		}
		items, ok := result.([]any)
		if !ok || len(items) == 0 {
			t.Fatalf("unexpected for_each result: %#v", result)
		}
		for _, item := range items {
			if v, ok := item.(int); ok && v%2 != 0 {
				t.Fatalf("for_each item should be transformed to even value, got %#v", items)
			}
		}
	})

	t.Run("match_hit_first", func(t *testing.T) {
		flow := triggerflow.New(nil, "match-first")
		flow.When("START", "").Match("hit_first").
			Case("A").To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) { return "case-a", nil }), false, "case-a").
			Case("B").To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) { return "case-b", nil }), false, "case-b").
			CaseElse().To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) { return "case-else", nil }), false, "case-else").
			EndMatch().To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "match-first-result")

		result, err := flow.Start("B", triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("match hit_first failed: %v", err)
		}
		if result != "case-b" {
			t.Fatalf("unexpected match hit_first result: %#v", result)
		}
	})

	t.Run("match_hit_all", func(t *testing.T) {
		flow := triggerflow.New(nil, "match-all")
		flow.When("START", "").Match("hit_all").
			Case(triggerflow.Condition(func(data *triggerflow.EventData) bool {
				v, _ := data.Value.(int)
				return v >= 2
			})).To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) { return "ge2", nil }), false, "case-ge2").
			Case(triggerflow.Condition(func(data *triggerflow.EventData) bool {
				v, _ := data.Value.(int)
				return v%2 == 0
			})).To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) { return "even", nil }), false, "case-even").
			EndMatch().To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "match-all-result")

		result, err := flow.Start(2, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("match hit_all failed: %v", err)
		}
		items, ok := result.([]any)
		if !ok || len(items) == 0 {
			t.Fatalf("unexpected match hit_all result: %#v", result)
		}
	})

	t.Run("collect", func(t *testing.T) {
		triggerflow.GlobalBlockData.Clear()
		flow := triggerflow.New(nil, "collect")
		start := flow.When("START", "")

		start.To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) {
			return "left", nil
		}), false, "left").Collect("group", "left", "filled_and_update").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "collect-left")

		start.To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) {
			return "right", nil
		}), false, "right").Collect("group", "right", "filled_and_update").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "collect-right")

		result, err := flow.Start(nil, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("collect failed: %v", err)
		}
		values, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("unexpected collect result type: %T (%#v)", result, result)
		}
		if values["left"] == nil || values["right"] == nil {
			t.Fatalf("collect result missing branches: %#v", values)
		}
	})

	t.Run("collect_filled_then_empty", func(t *testing.T) {
		triggerflow.GlobalBlockData.Clear()
		flow := triggerflow.New(nil, "collect-reset")
		start := flow.When("START", "")

		start.To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) { return "v1", nil }), false, "b1").Collect("group_reset", "b1", "filled_then_empty")
		start.To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) { return "v2", nil }), false, "b2").Collect("group_reset", "b2", "filled_then_empty").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "collect-reset-result")

		for i := 0; i < 2; i++ {
			result, err := flow.Start(nil, triggerflow.WithRunTimeout(3*time.Second))
			if err != nil {
				t.Fatalf("collect filled_then_empty run %d failed: %v", i+1, err)
			}
			values, ok := result.(map[string]any)
			if !ok || len(values) == 0 {
				t.Fatalf("collect filled_then_empty run %d mismatch: %#v", i+1, result)
			}
			if values["b1"] == nil && values["b2"] == nil {
				t.Fatalf("collect filled_then_empty run %d missing expected branches: %#v", i+1, values)
			}
		}
	})

	t.Run("side_branch_not_pollute_main_chain", func(t *testing.T) {
		flow := triggerflow.New(nil, "side-branch")
		start := flow.When("START", "")
		start.SideBranch(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			_ = data.SetRuntimeData("side.flag", true)
			return "side", nil
		}), "side")
		start.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult("main")
			return "main", nil
		}), false, "main")

		result, err := flow.Start(nil, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("side_branch flow failed: %v", err)
		}
		if result != "main" {
			t.Fatalf("expected main chain result, got %#v", result)
		}
	})
}

func TestRuntimeBlueprintAndSkipExceptions(t *testing.T) {
	t.Run("runtime_stream_put_stop", func(t *testing.T) {
		flow := triggerflow.New(nil, "runtime-stream")
		flow.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			_ = data.PutIntoStream("one")
			_ = data.PutIntoStream("two")
			_ = data.StopStream()
			return "done", nil
		}), false, "stream").End()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		stream, err := flow.GetRuntimeStream(ctx, nil, triggerflow.WithRunTimeout(2*time.Second))
		if err != nil {
			t.Fatalf("GetRuntimeStream failed: %v", err)
		}
		items := readAll(stream)
		if len(items) != 2 || items[0] != "one" || items[1] != "two" {
			t.Fatalf("unexpected runtime stream values: %#v", items)
		}
	})

	t.Run("runtime_stream_timeout", func(t *testing.T) {
		flow := triggerflow.New(nil, "runtime-timeout")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		stream, err := flow.GetRuntimeStream(ctx, nil, triggerflow.WithRunTimeout(120*time.Millisecond))
		if err != nil {
			t.Fatalf("GetRuntimeStream timeout flow failed: %v", err)
		}
		items := readAll(stream)
		if len(items) != 0 {
			t.Fatalf("expected empty stream on timeout flow, got %#v", items)
		}
	})

	t.Run("set_result_override_and_timeout", func(t *testing.T) {
		flow := triggerflow.New(nil, "set-result")
		flow.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult("manual")
			return "default", nil
		}), false, "set-result").End()

		result, err := flow.Start(nil, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("set_result flow failed: %v", err)
		}
		if result != "manual" {
			t.Fatalf("expected explicit set_result override, got %#v", result)
		}

		timeoutFlow := triggerflow.New(nil, "result-timeout")
		exec := timeoutFlow.CreateExecution()
		_, _ = exec.Start(nil, false, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
		defer cancel()
		if _, err := exec.GetResult(ctx); err == nil {
			t.Fatalf("expected GetResult timeout error")
		}
	})

	t.Run("blueprint_copy_execution_isolation_and_flow_data_fanout", func(t *testing.T) {
		source := triggerflow.New(nil, "source")
		source.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			if v, ok := data.Value.(int); ok {
				return v + 1, nil
			}
			return data.Value, nil
		}), false, "inc").End()

		snapshot := source.SaveBluePrint()
		cloned := triggerflow.New(snapshot, "cloned")
		clonedResult, err := cloned.Start(10, triggerflow.WithRunTimeout(3*time.Second))
		if err != nil {
			t.Fatalf("cloned blueprint start failed: %v", err)
		}
		if clonedResult != 11 {
			t.Fatalf("unexpected cloned blueprint result: %#v", clonedResult)
		}

		isolationFlow := triggerflow.New(nil, "runtime-isolation")
		isolationFlow.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			_ = data.SetRuntimeData("value", data.Value)
			data.SetResult(data.GetRuntimeData("value", nil))
			return data.Value, nil
		}), false, "isolation")

		execA := isolationFlow.CreateExecution()
		execB := isolationFlow.CreateExecution()
		if _, err := execA.Start("A", false, 0); err != nil {
			t.Fatalf("execA start failed: %v", err)
		}
		if _, err := execB.Start("B", false, 0); err != nil {
			t.Fatalf("execB start failed: %v", err)
		}
		ctxA, cancelA := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelA()
		ctxB, cancelB := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelB()
		resA, err := execA.GetResult(ctxA)
		if err != nil {
			t.Fatalf("execA result failed: %v", err)
		}
		resB, err := execB.GetResult(ctxB)
		if err != nil {
			t.Fatalf("execB result failed: %v", err)
		}
		if resA != "A" || resB != "B" {
			t.Fatalf("execution runtime isolation mismatch: A=%#v B=%#v", resA, resB)
		}

		fanoutFlow := triggerflow.New(nil, "flow-data-fanout")
		fanoutFlow.When(map[triggerflow.TriggerType][]string{triggerflow.TriggerTypeFlowData: {"shared"}}, "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			data.SetResult(data.Value)
			return data.Value, nil
		}), false, "flow-listener")

		flowExecA := fanoutFlow.CreateExecution()
		flowExecB := fanoutFlow.CreateExecution()
		if err := fanoutFlow.SetFlowData("shared", "fanout", true); err != nil {
			t.Fatalf("set flow data fanout failed: %v", err)
		}
		ctxFA, cancelFA := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelFA()
		ctxFB, cancelFB := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelFB()
		fanA, err := flowExecA.GetResult(ctxFA)
		if err != nil {
			t.Fatalf("flowExecA fanout result failed: %v", err)
		}
		fanB, err := flowExecB.GetResult(ctxFB)
		if err != nil {
			t.Fatalf("flowExecB fanout result failed: %v", err)
		}
		if fanA != "fanout" || fanB != "fanout" {
			t.Fatalf("flow_data fanout mismatch: A=%#v B=%#v", fanA, fanB)
		}
	})

	t.Run("skip_exceptions", func(t *testing.T) {
		strictFlow := triggerflow.New(nil, "strict")
		strictFlow.When("START", "").To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) {
			return nil, errors.New("boom")
		}), false, "err")
		if _, err := strictFlow.Start(nil, triggerflow.NoWaitForResult(), triggerflow.WithRunTimeout(3*time.Second)); err == nil {
			t.Fatalf("expected strict flow to return handler error")
		}

		skipFlow := triggerflow.New(nil, "skip", true)
		skipFlow.When("START", "").To(triggerflow.Handler(func(_ *triggerflow.EventData) (any, error) {
			return nil, errors.New("boom")
		}), false, "err")
		if _, err := skipFlow.Start(nil, triggerflow.NoWaitForResult(), triggerflow.WithRunTimeout(3*time.Second)); err != nil {
			t.Fatalf("expected skip_exceptions flow to ignore handler error, got %v", err)
		}
	})
}

func readAll(ch <-chan any) []any {
	items := make([]any, 0)
	for item := range ch {
		items = append(items, item)
	}
	return items
}
