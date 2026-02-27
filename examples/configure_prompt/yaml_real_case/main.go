package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
)

func main() {
	agentlyApp := agently.NewAgently()
	agent := agentlyApp.CreateAgent("yaml-real-case")

	content := `.alias:
  set_request_prompt:
    .args:
      - instruct
      - Reply politely.
.agent:
  system: You are ${role}
.request:
  input: Ask ${topic}`
	mappings := map[string]any{"role": "assistant", "topic": "TriggerFlow"}
	if err := agent.LoadYAMLPrompt(content, mappings); err != nil {
		panic(err)
	}

	text, err := agent.GetPromptText()
	if err != nil {
		panic(err)
	}
	fmt.Println(text)
}
