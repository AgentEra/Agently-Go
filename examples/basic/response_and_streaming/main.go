package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
)

const defaultTimeout = 90 * time.Second

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
	fmt.Printf("using provider: base_url=%s model=%s\n", baseURL, model)
	return agentlyApp
}

func differentResponseResults(agentlyApp *agently.Main) error {
	fmt.Println("\n=== Step 05: Different Response Results ===")
	response := agentlyApp.CreateAgent("response-result").
		Input("Please explain recursion with a short example.").
		Output(map[string]any{
			"definition": "string",
			"example":    "string",
		}).
		GetResponse()

	text, err := response.Result.GetText(core.WithTimeout(defaultTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("[text] %s\n", text)

	data, err := response.Result.GetData(
		core.GetDataOptions{Type: "parsed"},
		core.WithTimeout(defaultTimeout),
	)
	if err != nil {
		return err
	}
	fmt.Printf("[data] %#v\n", data)

	dataObject, err := response.Result.GetDataObject(core.WithTimeout(defaultTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("[data_object] %#v\n", dataObject)

	meta, err := response.Result.GetMeta(core.WithTimeout(defaultTimeout))
	if err != nil {
		return err
	}
	fmt.Printf("[meta] %#v\n", meta)

	streamResponse := agentlyApp.CreateAgent("response-result-stream").
		Input("List 3 recursion tips.").
		Output(map[string]any{
			"tips": []any{"string"},
		}).
		GetResponse()

	fmt.Print("[delta] ")
	stream, err := streamResponse.Result.GetGenerator("delta", core.WithTimeout(defaultTimeout))
	if err != nil {
		return err
	}
	for item := range stream {
		fmt.Print(item)
	}
	fmt.Println()
	return nil
}

func concurrentRequests(agentlyApp *agently.Main) error {
	fmt.Println("\n=== Step 05: Concurrent Requests ===")
	prompts := []string{
		"Summarize recursion in one sentence.",
		"Give one example of recursion in Python.",
	}

	results := make([]string, len(prompts))
	errs := make([]error, len(prompts))
	start := time.Now()

	var wg sync.WaitGroup
	for i, prompt := range prompts {
		i := i
		prompt := prompt
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := agentlyApp.CreateAgent(fmt.Sprintf("concurrent-%d", i)).
				Input(prompt).
				GetResponse()
			text, err := resp.Result.GetText(core.WithTimeout(defaultTimeout))
			if err != nil {
				errs[i] = err
				return
			}
			results[i] = text
		}()
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	fmt.Printf("[elapsed] %s\n", time.Since(start))
	for i, item := range results {
		fmt.Printf("[result %d] %s\n", i+1, item)
	}
	return nil
}

func basicDeltaStreaming(agentlyApp *agently.Main) error {
	fmt.Println("\n=== Step 06: Basic Delta Streaming ===")
	stream, err := agentlyApp.CreateAgent("stream-delta").
		Input("Give me a short speech about recursion.").
		GetGenerator("delta", core.WithTimeout(defaultTimeout))
	if err != nil {
		return err
	}
	for item := range stream {
		fmt.Print(item)
	}
	fmt.Println()
	return nil
}

func instantStructuredStreaming(agentlyApp *agently.Main) error {
	fmt.Println("\n=== Step 06: Instant Structured Streaming ===")
	stream, err := agentlyApp.CreateAgent("stream-instant").
		Input("Explain recursion with a short definition and two tips.").
		Output(map[string]any{
			"definition": "string",
			"tips":       []any{"string"},
		}).
		GetGenerator("instant", core.WithTimeout(defaultTimeout))
	if err != nil {
		return err
	}

	currentPath := ""
	for item := range stream {
		data, ok := item.(types.StreamingData)
		if !ok {
			continue
		}
		changed := currentPath != data.Path
		currentPath = data.Path
		if data.WildcardPath == "tips[*]" {
			if changed {
				index := strings.TrimSuffix(strings.TrimPrefix(data.Path, "tips["), "]")
				fmt.Printf("\nTip %s: ", index)
			}
			if data.Delta != "" {
				fmt.Print(data.Delta)
			}
		}
		if data.Path == "definition" {
			if changed {
				fmt.Print("\nDefinition: ")
			}
			if data.Delta != "" {
				fmt.Print(data.Delta)
			}
		}
	}
	fmt.Println()
	return nil
}

func specificEventStreaming(agentlyApp *agently.Main) error {
	fmt.Println("\n=== Step 06: Specific Event Streaming ===")
	stream, err := agentlyApp.CreateAgent("stream-specific").
		Input("Tell me a short story about recursion.").
		GetGenerator(
			"specific",
			core.WithSpecific("reasoning_delta", "delta", "tool_calls"),
			core.WithTimeout(defaultTimeout),
		)
	if err != nil {
		return err
	}

	currentEvent := ""
	for item := range stream {
		message, ok := item.(types.ResponseMessage)
		if !ok {
			continue
		}
		eventName := string(message.Event)
		if eventName == "reasoning_delta" || eventName == "delta" {
			if currentEvent != eventName {
				currentEvent = eventName
				if eventName == "reasoning_delta" {
					fmt.Print("\n[reasoning] ")
				} else {
					fmt.Print("\n[answer] ")
				}
			}
			fmt.Print(message.Data)
			continue
		}
		if eventName == "tool_calls" {
			fmt.Printf("\n[tool_calls] %#v", message.Data)
		}
	}
	fmt.Println()
	return nil
}

func main() {
	agentlyApp := configureMain()

	if err := differentResponseResults(agentlyApp); err != nil {
		panic(err)
	}
	if err := concurrentRequests(agentlyApp); err != nil {
		panic(err)
	}
	if err := basicDeltaStreaming(agentlyApp); err != nil {
		panic(err)
	}
	if err := instantStructuredStreaming(agentlyApp); err != nil {
		panic(err)
	}
	if err := specificEventStreaming(agentlyApp); err != nil {
		panic(err)
	}
}
