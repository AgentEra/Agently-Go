package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
)

func main() {
	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url":   "http://localhost:11434/v1",
		"model":      "qwen2.5:7b",
		"model_type": "chat",
	})
	agentlyApp.SetSettings("debug", false)

	agent := agentlyApp.CreateAgent("sync-generator-streaming")
	stream, err := agent.Input("Give me a long speech").GetGenerator("delta")
	if err != nil {
		panic(err)
	}

	for delta := range stream {
		fmt.Print(delta)
	}
	fmt.Println()
}
