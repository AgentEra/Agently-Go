package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func main() {
	agentlyApp := agently.NewAgently()
	agent := agentlyApp.CreateAgent("tool-agent")

	err := agent.RegisterTool(types.ToolInfo{
		Name:   "sum",
		Desc:   "sum two integers",
		Kwargs: map[string]any{"a": "number", "b": "number"},
	}, func(kwargs map[string]any) (any, error) {
		return kwargs["a"], nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("registered=%d\n", len(agent.Tool().Manager().GetToolList(nil)))
}
