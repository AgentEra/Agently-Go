package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
)

func buildMain() *agently.Main {
	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url": "http://127.0.0.1:11434/v1",
		"model":    "qwen2.5:7b",
	})
	return agentlyApp
}

func basicPromptMethods(agentlyApp *agently.Main) {
	fmt.Println("== basic_prompt_methods ==")
	agent := agentlyApp.CreateAgent("basic-prompt-methods")
	agent.SetAgentPrompt("system", "You are a useful assistant.")
	agent.SetRequestPrompt("input", "Hello")

	first, err := agent.Start()
	if err != nil {
		panic(err)
	}
	fmt.Printf("first=%#v\n", first)

	_, err = agent.Start()
	fmt.Printf("second_error=%v\n", err)

	agent.AgentPrompt().Clear()
	agent.Prompt().Clear()
}

func requestInstance(agentlyApp *agently.Main) {
	fmt.Println("\n== request_instance ==")
	request := agentlyApp.CreateRequest("basic-request")
	result, err := request.SetPrompt("input", "Hi").GetData()
	if err != nil {
		panic(err)
	}
	fmt.Printf("request_result=%#v\n", result)
}

func responseReuse(agentlyApp *agently.Main) {
	fmt.Println("\n== response_reuse ==")
	agent := agentlyApp.CreateAgent("response-reuse")
	response := agent.Input("hi").GetResponse()

	resultData, err := response.Result.GetData(core.GetDataOptions{Type: "parsed"})
	if err != nil {
		panic(err)
	}
	resultMeta, err := response.Result.GetMeta()
	if err != nil {
		panic(err)
	}
	fmt.Printf("data=%#v\n", resultData)
	fmt.Printf("meta=%#v\n", resultMeta)
}

func placeholderMappings(agentlyApp *agently.Main) {
	fmt.Println("\n== placeholder_mappings ==")
	agent := agentlyApp.CreateAgent("placeholder-mappings")
	result, err := agent.
		SetRequestPrompt("input", "My question is ${question}", map[string]any{"question": "Who're you?"}).
		SetRequestPrompt(
			"info",
			map[string]any{"${role_settings}": map[string]any{"${name_key}": "${name_value}"}},
			map[string]any{
				"role_settings": "Role Settings",
				"name_key":      "Name",
				"name_value":    "Alice Agently",
			},
		).
		Start()
	if err != nil {
		panic(err)
	}
	fmt.Printf("result=%#v\n", result)
}

func quickPromptMethods(agentlyApp *agently.Main) {
	fmt.Println("\n== quick_prompt_methods ==")
	agent := agentlyApp.CreateAgent("quick-methods")
	result, err := agent.
		Role("You're a useful assistant named ${assistant_name}.", core.Always(), core.WithMappings(map[string]any{"assistant_name": "Alice Agently"})).
		Input("What's your name?").
		Start()
	if err != nil {
		panic(err)
	}
	fmt.Printf("result=%#v\n", result)
}

func main() {
	agentlyApp := buildMain()
	basicPromptMethods(agentlyApp)
	requestInstance(agentlyApp)
	responseReuse(agentlyApp)
	placeholderMappings(agentlyApp)
	quickPromptMethods(agentlyApp)
}
