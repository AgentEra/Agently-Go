package main

import (
	"fmt"
	"time"

	"github.com/AgentEra/Agently-Go/agently/triggerflow"
)

const flowTimeout = 8 * time.Second

func batchDemo() error {
	fmt.Println("=== batch.py ===")
	flow := triggerflow.New(nil, "batch-demo")

	echo := func(data *triggerflow.EventData) (any, error) {
		wait := 80 * time.Millisecond
		fmt.Printf("wait %v: %v\n", wait, data.Value)
		time.Sleep(wait)
		return fmt.Sprintf("wait %v: %v", wait, data.Value), nil
	}

	flow.Batch([]any{
		triggerflow.NamedChunk{Name: "echo_1", Handler: echo},
		triggerflow.NamedChunk{Name: "echo_2", Handler: echo},
		triggerflow.NamedChunk{Name: "echo_3", Handler: echo},
		triggerflow.NamedChunk{Name: "echo_4", Handler: echo},
	}, triggerflow.WithBatchConcurrency(2)).End()

	result, err := flow.Start("Agently", triggerflow.WithRunTimeout(flowTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("result=%#v\n", result)
	return nil
}

func forEachDemo() error {
	fmt.Println("\n=== for_each.py ===")
	handle := func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("START HANDLING: %#v\n", data.Value)
		time.Sleep(60 * time.Millisecond)
		fmt.Printf("FINISH HANDLING: %#v\n", data.Value)
		return data.Value, nil
	}

	flow := triggerflow.New(nil, "for-each-demo")
	flow.ForEach(2).
		To(triggerflow.Handler(handle)).
		EndForEach().
		To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			return data.Value, nil
		})).
		End()

	input := []any{1, 2, "a", "b", []any{1, 2, 3}, map[string]any{"say": "hello world"}}
	result, err := flow.Start(input, triggerflow.WithRunTimeout(flowTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("result=%#v\n", result)
	return nil
}

func matchCaseDemo() error {
	fmt.Println("\n=== match_case.py ===")
	flow1 := triggerflow.New(nil, "match-case-1")
	flow1.To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) {
		return 3, nil
	})).
		Match("").
		Case(1).
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return "It is One!", nil })).
		Case(triggerflow.Condition(func(data *triggerflow.EventData) bool { return data.Value == 2 })).
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return "It is Two!", nil })).
		CaseElse().
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return "I don't know!", nil })).
		EndMatch().
		To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			fmt.Println(data.Value)
			return data.Value, nil
		})).
		End()

	if _, err := flow1.Start(nil, triggerflow.WithRunTimeout(flowTimeout)); err != nil {
		return err
	}

	flow2 := triggerflow.New(nil, "match-case-2")
	flow2.To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) {
		return []any{1, "2", []any{"Agently"}}, nil
	})).
		ForEach(0).
		Match("").
		Case(triggerflow.Condition(func(data *triggerflow.EventData) bool { return data.Value == 1 })).
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return "OK", nil })).
		CaseElse().
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return "Not OK", nil })).
		EndMatch().
		EndForEach().
		To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			fmt.Printf("%#v\n", data.Value)
			return data.Value, nil
		})).
		End()

	if _, err := flow2.Start(nil, triggerflow.WithRunTimeout(flowTimeout)); err != nil {
		return err
	}

	flow3 := triggerflow.New(nil, "if-condition")
	flow3.To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) {
		return 1, nil
	})).
		IfCondition(1).
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return 2, nil })).
		IfCondition(1).
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return "1.OK 2.OK", nil })).
		ElseCondition().
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return "1.OK 2.Not OK", nil })).
		EndCondition().
		ElifCondition(2).
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return "Emm...", nil })).
		ElseCondition().
		To(triggerflow.Handler(func(*triggerflow.EventData) (any, error) { return "Not OK", nil })).
		EndCondition().
		To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			fmt.Println(data.Value)
			return data.Value, nil
		})).
		End()

	_, err := flow3.Start(nil, triggerflow.WithRunTimeout(flowTimeout))
	return err
}

func main() {
	if err := batchDemo(); err != nil {
		panic(err)
	}
	if err := forEachDemo(); err != nil {
		panic(err)
	}
	if err := matchCaseDemo(); err != nil {
		panic(err)
	}
}
