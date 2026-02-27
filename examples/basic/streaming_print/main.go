package main

import (
	"fmt"
	"strings"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func buildMain() *agently.Main {
	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url": "http://127.0.0.1:11434/v1",
		"model":    "qwen2.5:7b",
	})
	return agentlyApp
}

func basicDeltaStreaming(agentlyApp *agently.Main) {
	fmt.Println("== basic_delta_streaming ==")
	agent := agentlyApp.CreateAgent("streaming-print-delta")
	stream, err := agent.Input("Give me a short speech about recursion.").GetGenerator("delta")
	if err != nil {
		panic(err)
	}
	for delta := range stream {
		fmt.Print(delta)
	}
	fmt.Println()
	fmt.Println()
}

func instantStructuredStreaming(agentlyApp *agently.Main) {
	fmt.Println("== instant_structured_streaming ==")
	agent := agentlyApp.CreateAgent("streaming-print-instant")
	stream, err := agent.
		Input("Explain recursion with a short definition and two tips.").
		Output(map[string]any{
			"definition": "string",
			"tips":       []any{"string"},
		}).
		GetGenerator("instant")
	if err != nil {
		panic(err)
	}

	currentPath := ""
	for item := range stream {
		data, ok := item.(types.StreamingData)
		if !ok {
			continue
		}
		changed := currentPath != data.Path
		currentPath = data.Path
		if data.WildcardPath == "tips[*]" {
			if changed {
				index := strings.TrimSuffix(strings.TrimPrefix(data.Path, "tips["), "]")
				fmt.Printf("\nTip %s: ", index)
			}
			if data.Delta != "" {
				fmt.Print(data.Delta)
			}
		}
		if data.Path == "definition" {
			if changed {
				fmt.Print("\nDefinition: ")
			}
			if data.Delta != "" {
				fmt.Print(data.Delta)
			}
		}
	}
	fmt.Println()
	fmt.Println()
}

func specificEventStreaming(agentlyApp *agently.Main) {
	fmt.Println("== specific_event_streaming ==")
	agent := agentlyApp.CreateAgent("streaming-print-specific")
	stream, err := agent.Input("Tell me a short story about recursion.").GetGenerator("specific", []string{"reasoning_delta", "delta", "tool_calls"})
	if err != nil {
		panic(err)
	}

	currentEvent := ""
	for item := range stream {
		message, ok := item.(types.ResponseMessage)
		if !ok {
			continue
		}
		eventName := string(message.Event)
		if eventName == "reasoning_delta" || eventName == "delta" {
			if currentEvent != eventName {
				currentEvent = eventName
				if eventName == "reasoning_delta" {
					fmt.Print("\n[reasoning] ")
				} else {
					fmt.Print("\n[answer] ")
				}
			}
			fmt.Print(message.Data)
			continue
		}
		if eventName == "tool_calls" {
			fmt.Printf("\n[tool_calls] %#v", message.Data)
		}
	}
	fmt.Println()
	fmt.Println()
}

func main() {
	agentlyApp := buildMain()
	basicDeltaStreaming(agentlyApp)
	instantStructuredStreaming(agentlyApp)
	specificEventStreaming(agentlyApp)
}
