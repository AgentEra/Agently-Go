package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
)

func main() {
	agentlyApp := agently.NewAgently()
	agent := agentlyApp.CreateAgent("configure-roundtrip")

	yamlPrompt := `.agent:
  system: SYS
.request:
  input: IN
  output:
    answer:
      $type: str`
	if err := agent.LoadYAMLPrompt(yamlPrompt); err != nil {
		panic(err)
	}

	jsonPrompt, err := agent.GetJSONPrompt()
	if err != nil {
		panic(err)
	}
	fmt.Println(jsonPrompt)
}
