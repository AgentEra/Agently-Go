package common

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	agently "github.com/AgentEra/Agently-Go/agently"
)

const (
	defaultOllamaBaseURL = "http://127.0.0.1:11434/v1"
	defaultOllamaModel   = "qwen2.5:7b"
)

func OllamaConfigFromEnv() (string, string) {
	baseURL := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	model := strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
	if model == "" {
		model = defaultOllamaModel
	}
	return baseURL, model
}

func ApplyOllamaDefaults(agentlyApp *agently.Main) (string, string) {
	baseURL, model := OllamaConfigFromEnv()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url":   baseURL,
		"model":      model,
		"model_type": "chat",
		"stream":     true,
		"request_options": map[string]any{
			"temperature": 0.1,
		},
		"timeout": map[string]any{
			"read": 180.0,
		},
	})
	return baseURL, model
}

func ConsumeStream(ctx context.Context, stream <-chan any, onItem func(item any) bool) error {
	for {
		select {
		case item, ok := <-stream:
			if !ok {
				return nil
			}
			if onItem != nil {
				if keep := onItem(item); !keep {
					return nil
				}
			}
		case <-ctx.Done():
			return fmt.Errorf("stream timed out: %w", ctx.Err())
		}
	}
}

func TimeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}
