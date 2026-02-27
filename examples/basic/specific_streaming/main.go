package main

import (
	"fmt"
	"os"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func streamingWithFunctionCalling() {
	fmt.Println("Example 1: Get Streaming Result with Function Calling")
	fmt.Println()
	fmt.Println("-----")
	fmt.Println()

	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url": "http://localhost:11434/v1",
		"model":    "qwen2.5:7b",
	})

	agent := agentlyApp.CreateAgent("specific-function-calling")
	stream, err := agent.
		Input("What's the weather like today in New York?").
		Options(map[string]any{
			"tools": []any{
				map[string]any{
					"type": "function",
					"function": map[string]any{
						"name":        "get_weather",
						"description": "Get current temperature for provided coordinates in celsius.",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"latitude":  map[string]any{"type": "number"},
								"longitude": map[string]any{"type": "number"},
							},
							"required":             []any{"latitude", "longitude"},
							"additionalProperties": false,
						},
						"strict": true,
					},
				},
			},
		}).
		GetGenerator("specific", []string{"delta", "tool_calls"})
	if err != nil {
		panic(err)
	}

	for item := range stream {
		message, ok := item.(types.ResponseMessage)
		if !ok {
			continue
		}
		if message.Event == types.ResponseEventDelta {
			fmt.Print(message.Data)
		}
		if message.Event == types.ResponseEventToolCalls {
			fmt.Printf("\n<tool_calls>\n%#v\n</tool_calls>\n", message.Data)
		}
	}
	fmt.Println()
}

func streamingWithReasoningFromDeepSeek() {
	fmt.Println("Example 2: Get Streaming with Reasoning from DeepSeek")
	fmt.Println()
	fmt.Println("-----")
	fmt.Println()

	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if baseURL == "" || apiKey == "" {
		fmt.Println("Skip: DEEPSEEK_BASE_URL / DEEPSEEK_API_KEY not set")
		return
	}

	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url": baseURL,
		"model":    "deepseek-reasoner",
		"auth":     apiKey,
		"request_options": map[string]any{
			"thinking": map[string]any{"type": "enabled"},
		},
	})

	agent := agentlyApp.CreateAgent("specific-deepseek-reasoning")
	stream, err := agent.Input("What's DeepSeek? Response in English.").GetGenerator("specific", []string{"reasoning_delta", "delta"})
	if err != nil {
		panic(err)
	}

	reasoningDone := false
	fmt.Println("[Thinking]:")
	for item := range stream {
		message, ok := item.(types.ResponseMessage)
		if !ok {
			continue
		}
		if message.Event == types.ResponseEventReasoning {
			fmt.Print(message.Data)
		}
		if message.Event == types.ResponseEventDelta {
			if !reasoningDone {
				reasoningDone = true
				fmt.Println("\n\n----\n\n[Reply]:")
			}
			fmt.Print(message.Data)
		}
	}
	fmt.Println()
}

func main() {
	streamingWithFunctionCalling()
	fmt.Println()
	streamingWithReasoningFromDeepSeek()
}
