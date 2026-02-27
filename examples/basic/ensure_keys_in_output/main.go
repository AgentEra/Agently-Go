package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
)

func main() {
	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url":   "http://localhost:11434/v1",
		"model":      "qwen2.5:7b",
		"model_type": "chat",
	})
	agentlyApp.SetSettings("debug", true)

	agent := agentlyApp.CreateAgent("ensure-keys-in-output")
	dataObject, err := agent.
		Input("How to develop an independent game? You MUST use the key name 'final.results' instead of the key name 'final.steps' IF value of the key 'control' > 1. Now 'control' is 1.").
		Output(map[string]any{
			"control": "number",
			"final": map[string]any{
				"steps": []any{"string"},
			},
			"resources": []any{
				map[string]any{
					"title": "string",
					"link":  "string",
				},
			},
		}).
		GetDataObject(core.GetDataOptions{
			EnsureKeys: []string{"final.steps", "resources[*].title", "resources[*].link"},
			KeyStyle:   "dot",
			MaxRetries: 1,
		})
	if err != nil {
		panic(err)
	}

	fmt.Printf("ensured_data_object=%#v\n", dataObject)
}
