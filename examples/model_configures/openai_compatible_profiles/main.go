package main

import (
	"context"
	"fmt"
	"os"
	"time"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
)

func main() {
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434/v1"
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen2.5:7b"
	}

	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url":   baseURL,
		"model":      model,
		"model_type": "chat",
	})

	req := agentlyApp.CreateRequest("openai-compatible-profiles")
	req.Input("Introduce Agently-Go in one concise sentence.")
	req.Output(map[string]any{"answer": "string"})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	data, err := req.GetData(ctx, core.GetDataOptions{Type: "parsed", EnsureKeys: []string{"answer"}, MaxRetries: 2})
	if err != nil {
		panic(err)
	}
	fmt.Printf("result=%#v\n", data)
}
