package main

import (
	"fmt"
	"time"

	"github.com/AgentEra/Agently-Go/agently/triggerflow"
)

const timeout = 5 * time.Second

func basicFlow() error {
	fmt.Println("=== basic_flow.py ===")
	flow := triggerflow.New(nil, "basic-flow")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("Hello, %v\n", data.Value)
		return data.Value, nil
	})).To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("Bye, %v\n", data.Value)
		return data.Value, nil
	})).End()

	execution := flow.CreateExecution()
	_, err := execution.Start("Agently", triggerflow.WithRunTimeout(timeout))
	return err
}

func whenSignals() error {
	fmt.Println("\n=== when.py ===")
	flow := triggerflow.New(nil, "when-signals")

	changeRuntimeData := flow.Chunk(triggerflow.NamedChunk{
		Name: "change_runtime_data",
		Handler: func(data *triggerflow.EventData) (any, error) {
			time.Sleep(40 * time.Millisecond)
			return nil, data.SetRuntimeData("test", "Hello", triggerflow.EmitSignal())
		},
	})
	changeAnotherRuntimeData := flow.Chunk(triggerflow.NamedChunk{
		Name: "change_another_runtime_data",
		Handler: func(data *triggerflow.EventData) (any, error) {
			time.Sleep(80 * time.Millisecond)
			return nil, data.SetRuntimeData("test_2", "Bye", triggerflow.EmitSignal())
		},
	})
	changeFlowData := flow.Chunk(triggerflow.NamedChunk{
		Name: "change_flow_data",
		Handler: func(data *triggerflow.EventData) (any, error) {
			time.Sleep(20 * time.Millisecond)
			return nil, data.SetFlowData("test", "Hello", triggerflow.EmitSignal())
		},
	})
	changeAnotherFlowData := flow.Chunk(triggerflow.NamedChunk{
		Name: "change_another_flow_data",
		Handler: func(data *triggerflow.EventData) (any, error) {
			time.Sleep(60 * time.Millisecond)
			return nil, data.SetFlowData("test_2", "Bye", triggerflow.EmitSignal())
		},
	})

	flow.To(changeRuntimeData).
		To(changeAnotherRuntimeData).
		Collect("branch_end", "1", "filled_then_empty")

	flow.To(changeFlowData).
		To(changeAnotherFlowData).
		Collect("branch_end", "2", "filled_then_empty").
		To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			fmt.Println("All Done")
			return data.Value, nil
		})).
		End()

	flow.When(changeRuntimeData, "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("change runtime data done, but return nil: %#v\n", data.Value)
		return data.Value, nil
	}))

	flow.When(map[triggerflow.TriggerType][]string{
		triggerflow.TriggerTypeRuntimeData: {"test"},
	}, "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("runtime data 'test' changed: %#v\n", data.Value)
		return data.Value, nil
	}))

	flow.When(map[triggerflow.TriggerType][]string{
		triggerflow.TriggerTypeFlowData: {"test"},
	}, "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("flow data 'test' changed: %#v\n", data.Value)
		return data.Value, nil
	}))

	flow.When(map[triggerflow.TriggerType][]string{
		triggerflow.TriggerTypeRuntimeData: {"test"},
		triggerflow.TriggerTypeFlowData:    {"test"},
	}, "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("runtime data 'test' and flow data 'test' both changed: %#v\n", data.Value)
		return data.Value, nil
	}))

	flow.When(map[triggerflow.TriggerType][]string{
		triggerflow.TriggerTypeRuntimeData: {"test", "test_1"},
	}, "or").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("runtime data 'test' or 'test_1' changed: %#v\n", data.Value)
		return data.Value, nil
	}))

	flow.When(map[triggerflow.TriggerType][]string{
		triggerflow.TriggerTypeFlowData: {"test", "test_1"},
	}, "simple_or").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("flow data 'test' or 'test_1' changed: %#v\n", data.Value)
		return data.Value, nil
	}))

	flow.When(map[triggerflow.TriggerType][]string{
		triggerflow.TriggerTypeRuntimeData: {"test", "test_2"},
		triggerflow.TriggerTypeFlowData:    {"test", "test_2"},
	}, "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("all data changed: %#v\n", data.Value)
		return data.Value, nil
	}))

	_, err := flow.Start(nil, triggerflow.NoWaitForResult(), triggerflow.WithRunTimeout(timeout))
	return err
}

func main() {
	if err := basicFlow(); err != nil {
		panic(err)
	}
	if err := whenSignals(); err != nil {
		panic(err)
	}
}
