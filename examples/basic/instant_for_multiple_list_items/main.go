package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func main() {
	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url":   "http://localhost:11434/v1",
		"model":      "qwen2.5:7b",
		"model_type": "chat",
	})
	agentlyApp.SetSettings("debug", false)

	agent := agentlyApp.CreateAgent("instant-for-multiple-list-items")
	instantGenerator, err := agent.
		Input("How to develop an independent game?").
		Output(map[string]any{
			"steps": []any{"string"},
		}).
		GetGenerator("instant")
	if err != nil {
		panic(err)
	}

	for item := range instantGenerator {
		data, ok := item.(types.StreamingData)
		if !ok {
			continue
		}
		if data.WildcardPath == "steps[*]" {
			fmt.Println(data.Path, data.Indexes, data.Value, data.FullData)
		}
	}
}
