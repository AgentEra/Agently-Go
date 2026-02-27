package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
)

func main() {
	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("prompt.prompt_title_mapping", map[string]any{
		"system":             "SYSTEM ROLE",
		"input":              "INPUT",
		"output_requirement": "OUTPUT REQUIREMENT",
		"output":             "OUTPUT",
	})

	prompt := agentlyApp.CreatePrompt("title-mapping")
	prompt.Set("system", "sys")
	prompt.Set("input", "hello")
	prompt.Set("output", map[string]any{"answer": map[string]any{"$type": "str"}})
	text, err := prompt.ToText()
	if err != nil {
		panic(err)
	}
	fmt.Println(text)
}
