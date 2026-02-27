package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
)

func main() {
	agentlyApp := agently.NewAgently()
	prompt := agentlyApp.CreatePrompt("text-messages-schema")
	prompt.Set("input", "hello")
	prompt.Set("output", map[string]any{"answer": map[string]any{"$type": "str"}})

	text, _ := prompt.ToText()
	messages, _ := prompt.ToMessages(core.WithStrictRoleOrders(true))
	schema, _ := prompt.ToOutputModelSchema()

	fmt.Println(text)
	fmt.Printf("messages=%#v\n", messages)
	fmt.Printf("schema=%#v\n", schema)
}
