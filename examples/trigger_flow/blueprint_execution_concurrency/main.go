package main

import (
	"fmt"
	"time"

	"github.com/AgentEra/Agently-Go/agently/triggerflow"
)

func blueprintAndExecutionDemo() error {
	fmt.Println("=== blue_print_and_execution.py ===")
	blueprint := triggerflow.NewBluePrint("MyBluePrint")

	blueprint.AddEventHandler("test", func(data *triggerflow.EventData) (any, error) {
		fmt.Println("event:test", data.Value)
		return data.Value, nil
	}, "")
	blueprint.AddFlowDataHandler("test", func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("execution:%s flow_data:test %v\n", data.ExecutionID, data.Value)
		return data.Value, nil
	}, "")
	blueprint.AddRuntimeDataHandler("test", func(data *triggerflow.EventData) (any, error) {
		fmt.Printf("execution:%s runtime_data:test %v\n", data.ExecutionID, data.Value)
		return data.Value, nil
	}, "")

	flow := triggerflow.New(blueprint, "blueprint-demo")
	execution1 := flow.CreateExecution()
	if err := execution1.Emit("test", "hello"); err != nil {
		return err
	}
	if err := execution1.SetFlowData("test", "world", triggerflow.EmitSignal()); err != nil {
		return err
	}
	if err := execution1.SetRuntimeData("test", "Agently", triggerflow.EmitSignal()); err != nil {
		return err
	}

	execution2 := flow.CreateExecution()
	if err := execution2.Emit("test", "hello again"); err != nil {
		return err
	}
	if err := execution2.SetFlowData("test", "world again", triggerflow.EmitSignal()); err != nil {
		return err
	}
	if err := execution2.SetRuntimeData("test", "Agently again", triggerflow.EmitSignal()); err != nil {
		return err
	}

	if err := flow.SetFlowData("test", "all change", triggerflow.EmitSignal()); err != nil {
		return err
	}
	flow.RemoveExecution(execution2)
	if err := flow.SetFlowData("test", "only execution_1", triggerflow.EmitSignal()); err != nil {
		return err
	}
	return nil
}

func saveAndLoadBlueprintDemo() error {
	fmt.Println("\n=== save/load blueprint ===")
	flow := triggerflow.New(nil, "source-flow")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		return fmt.Sprintf("%v-ok", data.Value), nil
	})).End()

	blueprint := flow.SaveBluePrint()
	clone := triggerflow.New(blueprint, "clone-flow")
	result, err := clone.Start("value", triggerflow.WithRunTimeout(3*time.Second))
	if err != nil {
		return err
	}
	fmt.Printf("clone result=%#v\n", result)
	return nil
}

func main() {
	if err := blueprintAndExecutionDemo(); err != nil {
		panic(err)
	}
	if err := saveAndLoadBlueprintDemo(); err != nil {
		panic(err)
	}
}
