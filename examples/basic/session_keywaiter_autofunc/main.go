package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
)

const waitTimeout = 90 * time.Second

func configureMain() *agently.Main {
	baseURL := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434/v1"
	}
	model := strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
	if model == "" {
		model = "qwen2.5:7b"
	}

	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url":   baseURL,
		"model":      model,
		"model_type": "chat",
	})
	agentlyApp.SetSettings("debug", false)
	fmt.Printf("using provider: base_url=%s model=%s\n", baseURL, model)
	return agentlyApp
}

func keyWaiterGetSingle(agentName string, agentlyApp *agently.Main) error {
	fmt.Println("\n=== Key Waiter: Get Single Key ===")
	agent := agentlyApp.CreateAgent(agentName + "-single")
	agent.Input("34643523 + 52131231 = ?").Output(map[string]any{
		"thinking": "string",
		"result":   "number",
		"reply":    "string",
	})

	reply, err := agent.GetKeyResult("thinking", core.WithTimeout(waitTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("thinking=%#v\n", reply)
	return nil
}

func keyWaiterMulti(agentName string, agentlyApp *agently.Main) error {
	fmt.Println("\n=== Key Waiter: Wait Keys Stream ===")
	agent := agentlyApp.CreateAgent(agentName + "-multi")
	agent.Input("34643523 + 52131231 = ?").Output(map[string]any{
		"thinking": "string",
		"result":   "number",
		"reply":    "string",
	})

	stream, err := agent.WaitKeys([]string{"thinking", "reply"}, core.WithTimeout(waitTimeout))
	if err != nil {
		return err
	}
	for item := range stream {
		fmt.Printf("event=%v data=%#v\n", item[0], item[1])
	}
	return nil
}

func keyWaiterHandlers(agentName string, agentlyApp *agently.Main) error {
	fmt.Println("\n=== Key Waiter: OnKey + StartWaiter ===")
	agent := agentlyApp.CreateAgent(agentName + "-handlers")
	agent.Input("34643523 + 52131231 = ?").Output(map[string]any{
		"thinking": "string",
		"result":   "number",
		"reply":    "string",
	})

	agent.
		OnKey("thinking", func(result any) any {
			fmt.Printf("thinking: %v\n", result)
			return result
		}).
		OnKey("result", func(result any) any {
			fmt.Printf("result: %v\n", result)
			return result
		}).
		OnKey("reply", func(result any) any {
			fmt.Printf("reply: %v\n", result)
			return result
		})

	results, err := agent.StartWaiter(core.WithTimeout(waitTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("handled events=%d\n", len(results))
	return nil
}

func autoFuncDemo(agentName string, agentlyApp *agently.Main) error {
	fmt.Println("\n=== AutoFunc Demo ===")
	agent := agentlyApp.CreateAgent(agentName + "-autofunc")
	if err := agent.RegisterTool(types.ToolInfo{
		Name: "add",
		Desc: "add two integers",
		Kwargs: map[string]any{
			"a": "number",
			"b": "number",
		},
	}, func(kwargs map[string]any) (any, error) {
		a, _ := kwargs["a"].(float64)
		b, _ := kwargs["b"].(float64)
		return a + b, nil
	}); err != nil {
		return err
	}
	if err := agent.UseTools([]string{"add"}); err != nil {
		return err
	}

	calculate := agent.AutoFunc(
		"Return calculation result of {formula}. Use tool if needed.",
		map[string]any{
			"thinking": "string",
			"result":   "number",
			"reply":    "string",
		},
	)
	data, err := calculate(context.Background(), map[string]any{
		"formula": "3333+6666=?",
	})
	if err != nil {
		return err
	}
	fmt.Printf("auto_func_result=%#v\n", data)
	return nil
}

func sessionQuickDemo(agentName string, agentlyApp *agently.Main) error {
	fmt.Println("\n=== Session Quick Demo ===")
	agent := agentlyApp.CreateAgent(agentName + "-session")
	agent.ActivateSession("demo-session")
	agent.SetChatHistory([]types.ChatMessage{{Role: "user", Content: "hello"}})
	agent.AddChatHistory([]types.ChatMessage{{Role: "assistant", Content: "hi"}})

	reply, err := agent.Input("What did we just say?").GetText(core.WithTimeout(waitTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("session_reply=%s\n", reply)
	agent.DeactivateSession()
	return nil
}

func main() {
	agentlyApp := configureMain()
	agentName := "session-keywaiter-autofunc"

	if err := keyWaiterGetSingle(agentName, agentlyApp); err != nil {
		panic(err)
	}
	if err := keyWaiterMulti(agentName, agentlyApp); err != nil {
		panic(err)
	}
	if err := keyWaiterHandlers(agentName, agentlyApp); err != nil {
		panic(err)
	}
	if err := autoFuncDemo(agentName, agentlyApp); err != nil {
		panic(err)
	}
	if err := sessionQuickDemo(agentName, agentlyApp); err != nil {
		panic(err)
	}
}
