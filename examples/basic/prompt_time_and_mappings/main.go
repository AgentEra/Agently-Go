package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
)

func addCurrentTimeDemo() {
	agentlyApp := agently.NewAgently()
	agent := agentlyApp.CreateAgent("add-current-time")
	agent.Input("hello")

	text, err := agent.GetPromptText()
	if err != nil {
		panic(err)
	}
	fmt.Println("Default:\n", text)

	agent.SetSettings("prompt.add_current_time", false)
	textWithoutTime, err := agent.GetPromptText()
	if err != nil {
		panic(err)
	}
	fmt.Println("Turn off:\n", textWithoutTime)
}

func promptMappingsDemo() {
	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url":   "http://localhost:11434/v1",
		"model":      "qwen2.5:7b",
		"model_type": "chat",
	})

	userInput := "How are you today?"
	role := "A teacher for kids that 3 years old."

	agent := agentlyApp.CreateAgent("prompt-mappings")
	result, err := agent.
		Input("Acting as ${role} to response: ${user_input}", map[string]any{
			"user_input": userInput,
			"role":       role,
		}).
		Output(map[string]any{
			"reply": "string",
		}).
		Start()
	if err != nil {
		panic(err)
	}
	fmt.Printf("reply=%#v\n", result)
}

func main() {
	addCurrentTimeDemo()
	fmt.Println()
	fmt.Println("-----")
	fmt.Println()
	promptMappingsDemo()
}
