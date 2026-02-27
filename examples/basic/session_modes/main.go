package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/builtins/agent_extensions"
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

func basicSessionOnOff(agent *agentextensions.Agent) {
	fmt.Println("\n=== Example 1: Session On / Off ===")
	agent.ActivateSession("demo_on_off")

	fmt.Println("[Ask to remember]")
	agent.Input("Remember this: I need to buy eggs tomorrow.").StreamingPrint()

	fmt.Println("[Session ON]")
	agent.Input("What should I buy tomorrow?").StreamingPrint()

	agent.DeactivateSession()
	fmt.Println("[Session OFF]")
	agent.Input("What should I buy tomorrow?").StreamingPrint()
}

func sessionIsolationByID(agent *agentextensions.Agent) {
	fmt.Println("\n=== Example 2: Session Isolation by ID ===")
	agent.ActivateSession("trip_a")
	agent.Input("Remember this trip note: destination is Tokyo.").StreamingPrint()

	agent.ActivateSession("trip_b")
	agent.Input("Remember this trip note: destination is Paris.").StreamingPrint()

	fmt.Println("[Check trip_b]")
	agent.Input("What is my destination?").StreamingPrint()

	agent.ActivateSession("trip_a")
	fmt.Println("[Check trip_a]")
	agent.Input("What is my destination?").StreamingPrint()
}

func sessionRecordWithKeys(agent *agentextensions.Agent) {
	fmt.Println("\n=== Example 3: Record With input_keys/reply_keys ===")
	agent.ActivateSession("demo_key_record")
	agent.SetSettings("session.input_keys", []any{"info.task", "info.style", "input.lang"})
	agent.SetSettings("session.reply_keys", []any{"summary", "keywords"})

	result, err := agent.
		Info(map[string]any{"task": "Summarize Agently in one sentence.", "style": "technical"}).
		Input(map[string]any{"lang": "en", "extra": "ignore_me"}).
		Output(map[string]any{
			"summary":  "string",
			"keywords": []any{"string"},
			"extra":    "string",
		}).
		GetData(core.GetDataOptions{Type: "parsed"})
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Parsed Result]\n%#v\n", result)
	fmt.Printf("[Chat History]\n%#v\n", agent.AgentPrompt().Get("chat_history", nil, true))

	agent.SetSettings("session.input_keys", nil)
	agent.SetSettings("session.reply_keys", nil)
}

func main() {
	agentlyApp := buildMain()
	agent := agentlyApp.CreateAgent("session")

	// basicSessionOnOff(agent)
	// sessionIsolationByID(agent)
	sessionRecordWithKeys(agent)
}
