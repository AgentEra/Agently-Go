package main

import (
	"fmt"
	"time"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/triggerflow"
)

const runTimeout = 8 * time.Second

func step11_01Basics() error {
	fmt.Println("\n=== 11-01 basics ===")
	flow := triggerflow.New(nil, "tf-11-01")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return fmt.Sprintf("Hello, %v", data.Value), nil
	})).End()
	result, err := flow.Start("Agently", triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("result=%#v\n", result)
	return nil
}

func step11_02Branching() error {
	fmt.Println("\n=== 11-02 branching ===")
	flow := triggerflow.New(nil, "tf-11-02")

	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		_ = data.SetRuntimeData("flag", "ready", triggerflow.EmitSignal())
		return "runtime done", nil
	})).To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		_ = data.SetRuntimeData("phase", "ready", triggerflow.EmitSignal())
		return "runtime phase done", nil
	})).End()

	flow.When(map[triggerflow.TriggerType][]string{
		triggerflow.TriggerTypeRuntimeData: {"flag", "phase"},
	}, "and").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("[when both] %#v\n", data.Value)
		return data.Value, nil
	}))

	flow.When(map[triggerflow.TriggerType][]string{
		triggerflow.TriggerTypeRuntimeData: {"flag", "other"},
	}, "simple_or").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("[when or] %#v\n", data.Value)
		return data.Value, nil
	}))

	_, err := flow.Start(nil, triggerflow.NoWaitForResult(), triggerflow.WithRunTimeout(runTimeout))
	return err
}

func step11_03Concurrency() error {
	fmt.Println("\n=== 11-03 concurrency ===")
	flow := triggerflow.New(nil, "tf-11-03-batch")
	flow.Batch([]any{
		triggerflow.NamedChunk{
			Name: "a",
			Handler: func(data *triggerflow.EventData) (any, error) {
				time.Sleep(100 * time.Millisecond)
				return fmt.Sprintf("echo: %v", data.Value), nil
			},
		},
		triggerflow.NamedChunk{
			Name: "b",
			Handler: func(data *triggerflow.EventData) (any, error) {
				time.Sleep(100 * time.Millisecond)
				return fmt.Sprintf("echo: %v", data.Value), nil
			},
		},
		triggerflow.NamedChunk{
			Name: "c",
			Handler: func(data *triggerflow.EventData) (any, error) {
				time.Sleep(100 * time.Millisecond)
				return fmt.Sprintf("echo: %v", data.Value), nil
			},
		},
	}, triggerflow.WithBatchConcurrency(2)).End()

	result, err := flow.Start("hello", triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("batch result=%#v\n", result)

	flow2 := triggerflow.New(nil, "tf-11-03-foreach")
	flow2.To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) {
		return []any{1, 2, 3, 4}, nil
	})).ForEach(2).To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		time.Sleep(80 * time.Millisecond)
		return fmt.Sprintf("item=%v", data.Value), nil
	})).EndForEach().End()

	result2, err := flow2.Start(nil, triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("for_each result=%#v\n", result2)
	return nil
}

func step11_04DataFlow() error {
	fmt.Println("\n=== 11-04 data flow ===")
	flow := triggerflow.New(nil, "tf-11-04")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		_ = data.SetRuntimeData("user_id", "u-001", triggerflow.EmitSignal())
		return "runtime ok", nil
	})).Collect("done", "r1", "filled_then_empty").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("[collect runtime] %#v\n", data.Value)
		return data.Value, nil
	}))

	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		_ = data.SetRuntimeData("env", "prod", triggerflow.EmitSignal())
		return "runtime context ok", nil
	})).Collect("done", "r2", "filled_then_empty").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("[collect runtime context] %#v\n", data.Value)
		return data.Value, nil
	})).End()

	flow.When(map[triggerflow.TriggerType][]string{
		triggerflow.TriggerTypeRuntimeData: {"user_id"},
	}, "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("[when runtime] %#v\n", data.Value)
		return data.Value, nil
	}))

	_, err := flow.Start(nil, triggerflow.NoWaitForResult(), triggerflow.WithRunTimeout(runTimeout))
	return err
}

func step11_05Blueprint() error {
	fmt.Println("\n=== 11-05 blueprint ===")
	flow := triggerflow.New(nil, "tf-11-05")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return fmt.Sprintf("%v", data.Value), nil
	})).To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return fmt.Sprintf("%s", fmt.Sprintf("%v", data.Value)), nil
	})).End()

	blueprint := flow.SaveBluePrint()
	flow2 := triggerflow.New(blueprint, "tf-11-05-loaded")
	result, err := flow2.Start("agently", triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("result=%#v\n", result)
	return nil
}

func step11_06RuntimeEventData() error {
	fmt.Println("\n=== 11-06 runtime event data ===")
	flow := triggerflow.New(nil, "tf-11-06")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		_ = data.SetRuntimeData("seen_event", data.TriggerEvent, triggerflow.EmitSignal())
		if err := data.Emit("CustomEvent", map[string]any{
			"from":  data.TriggerEvent,
			"value": data.Value,
		}, triggerflow.TriggerTypeEvent); err != nil {
			return nil, err
		}
		return map[string]any{
			"event":   data.TriggerEvent,
			"type":    data.TriggerType,
			"value":   data.Value,
			"runtime": data.GetRuntimeData("seen_event", nil),
			"layer":   data.LayerMark(),
		}, nil
	})).End()
	flow.When("CustomEvent", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("[custom] %#v\n", data.Value)
		return data.Value, nil
	}))
	result, err := flow.Start("hello", triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("result=%#v\n", result)
	return nil
}

func step11_07EmitWhen() error {
	fmt.Println("\n=== 11-07 emit + when ===")
	flow := triggerflow.New(nil, "tf-11-07")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		if err := data.Emit("Plan.Read", map[string]any{"task": "read"}, triggerflow.TriggerTypeEvent); err != nil {
			return nil, err
		}
		if err := data.Emit("Plan.Write", map[string]any{"task": "write"}, triggerflow.TriggerTypeEvent); err != nil {
			return nil, err
		}
		return "plan done", nil
	})).End()

	flow.When("Plan.Read", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		task, _ := data.Value.(map[string]any)
		return fmt.Sprintf("read: %v", task["task"]), nil
	})).Collect("plan", "read", "filled_and_update")

	flow.When("Plan.Write", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		task, _ := data.Value.(map[string]any)
		return fmt.Sprintf("write: %v", task["task"]), nil
	})).Collect("plan", "write", "filled_and_update").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("[collect] %#v\n", data.Value)
		return data.Value, nil
	})).End()

	_, err := flow.Start("go", triggerflow.NoWaitForResult(), triggerflow.WithRunTimeout(runTimeout))
	return err
}

func step11_08LoopFlow() error {
	fmt.Println("\n=== 11-08 loop flow ===")
	flow := triggerflow.New(nil, "tf-11-08")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return nil, data.Emit("Loop", 0, triggerflow.TriggerTypeEvent)
	}))
	flow.When("Loop", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		count, _ := data.Value.(int)
		if count >= 3 {
			return nil, data.Emit("LoopEnd", count, triggerflow.TriggerTypeEvent)
		}
		return count, data.Emit("Loop", count+1, triggerflow.TriggerTypeEvent)
	}))
	flow.When("LoopEnd", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return fmt.Sprintf("done: %v", data.Value), nil
	})).End()
	result, err := flow.Start("start", triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("result=%#v\n", result)
	return nil
}

func step11_09Result() error {
	fmt.Println("\n=== 11-09 set result ===")
	flow := triggerflow.New(nil, "tf-11-09")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return fmt.Sprintf("work(%v)", data.Value), nil
	})).End()

	execution, _, err := flow.StartExecution("task-1", triggerflow.NoWaitForResult(), triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	execution.SetResult("final answer: done")
	result, err := execution.GetResult(core.WithTimeout(runTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("result=%#v\n", result)
	return nil
}

func step11_10RuntimeStream() error {
	fmt.Println("\n=== 11-10 runtime stream ===")
	flow := triggerflow.New(nil, "tf-11-10")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		for i := 0; i < 3; i++ {
			if err := data.PutIntoStream(map[string]any{"step": i + 1, "status": "working"}); err != nil {
				return nil, err
			}
			time.Sleep(50 * time.Millisecond)
		}
		if err := data.StopStream(); err != nil {
			return nil, err
		}
		return "done", nil
	})).End()

	stream, err := flow.GetRuntimeStream("start", triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	for event := range stream {
		fmt.Printf("[stream] %#v\n", event)
	}
	return nil
}

func step11_11SideBranch() error {
	fmt.Println("\n=== 11-11 side branch ===")
	flow := triggerflow.New(nil, "tf-11-11")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return fmt.Sprintf("main: %v", data.Value), nil
	})).
		Separator(true, true, "side branch boundary").
		SideBranch(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			fmt.Printf("[side] %v\n", data.Value)
			return "side done", nil
		}), "side-task").
		To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			fmt.Printf("[main] %v\n", data.Value)
			return data.Value, nil
		})).
		End()

	_, err := flow.Start("hello", triggerflow.WithRunTimeout(runTimeout))
	return err
}

func step11_12DiveDeep() error {
	fmt.Println("\n=== 11-12 dive deep ===")

	flowResult := triggerflow.New(nil, "tf-11-12-result")
	flowResult.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return fmt.Sprintf("work(%v)", data.Value), nil
	})).End()
	result, err := flowResult.Start("task", triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("[start_and_result] %#v\n", result)

	manual := triggerflow.New(nil, "tf-11-12-manual")
	manual.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return data.Value, nil
	})).End()
	execution, _, err := manual.StartExecution("ignored", triggerflow.NoWaitForResult(), triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	execution.SetResult("manual result")
	manualResult, err := execution.GetResult(core.WithTimeout(runTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("[manual result] %#v\n", manualResult)

	streamFlow := triggerflow.New(nil, "tf-11-12-stream")
	streamFlow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		_ = data.PutIntoStream("step-1")
		_ = data.PutIntoStream("step-2")
		_ = data.StopStream()
		return "done", nil
	}))
	stream, err := streamFlow.GetRuntimeStream("start", triggerflow.WithRunTimeout(runTimeout))
	if err != nil {
		return err
	}
	for event := range stream {
		fmt.Printf("[stream] %#v\n", event)
	}

	noResult := triggerflow.New(nil, "tf-11-12-no-result")
	noResult.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return "emitted", data.Emit("Ping", "pong", triggerflow.TriggerTypeEvent)
	}))
	noResult.When("Ping", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("[ping] %#v\n", data.Value)
		return nil, nil
	}))
	if _, err := noResult.Start("x", triggerflow.NoWaitForResult(), triggerflow.WithRunTimeout(runTimeout)); err != nil {
		return err
	}
	return nil
}

func main() {
	steps := []func() error{
		step11_01Basics,
		step11_02Branching,
		step11_03Concurrency,
		step11_04DataFlow,
		step11_05Blueprint,
		step11_06RuntimeEventData,
		step11_07EmitWhen,
		step11_08LoopFlow,
		step11_09Result,
		step11_10RuntimeStream,
		step11_11SideBranch,
		step11_12DiveDeep,
	}
	for i, step := range steps {
		if err := step(); err != nil {
			panic(fmt.Sprintf("step %d failed: %v", i+1, err))
		}
	}
}
