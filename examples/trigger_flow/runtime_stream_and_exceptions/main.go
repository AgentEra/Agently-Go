package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/AgentEra/Agently-Go/agently/triggerflow"
)

func runtimeStreamDemo() error {
	fmt.Println("=== runtime_stream.py ===")
	flow := triggerflow.New(nil, "runtime-stream-demo")

	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		time.Sleep(300 * time.Millisecond)
		if err := data.PutIntoStream(fmt.Sprintf("Hello, %v", data.Value)); err != nil {
			return nil, err
		}
		return data.Value, nil
	})).
		To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			time.Sleep(300 * time.Millisecond)
			if err := data.PutIntoStream(fmt.Sprintf("Bye, %v", data.Value)); err != nil {
				return nil, err
			}
			return data.Value, nil
		})).
		To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
			return data.Value, data.StopStream()
		})).
		End()

	execution := flow.CreateExecution()
	stream, err := execution.GetRuntimeStream("Agently", triggerflow.WithRunTimeout(4*time.Second))
	if err != nil {
		return err
	}
	for item := range stream {
		fmt.Println(item)
	}
	return nil
}

func exceptionCaptureDemo() {
	fmt.Println("\n=== exception_capture.py ===")
	flow := triggerflow.New(nil, "exception-capture")
	flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		_ = data
		return nil, errors.New("Test Exception")
	})).End()

	if _, err := flow.Start(nil, triggerflow.WithRunTimeout(2*time.Second)); err != nil {
		fmt.Println("Captured:", err)
		return
	}
	fmt.Println("No exception captured.")
}

func main() {
	if err := runtimeStreamDemo(); err != nil {
		panic(err)
	}
	exceptionCaptureDemo()
}
