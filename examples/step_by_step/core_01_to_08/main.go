package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
)

const timeout = 90 * time.Second

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
	return agentlyApp
}

func step01Settings(agentlyApp *agently.Main) {
	fmt.Println("\n=== 01-settings ===")
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url": "http://127.0.0.1:11434/v1",
		"model":    "qwen2.5:7b",
	})

	agent := agentlyApp.CreateAgent("step-01")
	agent.SetSettings("OpenAICompatible", map[string]any{
		"model": "qwen3:latest",
	})
	agent.SetSettings("debug", true)

	modelRequesterSettings, _ := agent.Settings().Get("plugins.ModelRequester.OpenAICompatible", map[string]any{}, true).(map[string]any)
	fmt.Printf("base_url=%v\n", modelRequesterSettings["base_url"])
	fmt.Printf("model=%v\n", modelRequesterSettings["model"])
}

func step02PromptMethods(agentlyApp *agently.Main) error {
	fmt.Println("\n=== 02-prompt_methods ===")
	agent := agentlyApp.CreateAgent("step-02")

	agent.SetAgentPrompt("system", "You are a useful assistant.")
	agent.SetRequestPrompt("input", "Hello")

	reply, err := agent.GetText(core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Printf("[basic_prompt_methods] %s\n", reply)

	response := agent.Input("hi").GetResponse()
	data, err := response.Result.GetData(core.GetDataOptions{Type: "parsed"}, core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	meta, err := response.Result.GetMeta(core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Printf("[response data] %#v\n", data)
	fmt.Printf("[response meta] %#v\n", meta)

	request := agentlyApp.CreateRequest("step-02-request-only")
	requestResult, err := request.SetPrompt("input", "Hi").GetData(core.GetDataOptions{Type: "parsed"}, core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Printf("[request instance] %#v\n", requestResult)

	quickResult, err := agent.
		Role("You're a useful assistant named ${assistant_name}.", core.Always(), core.WithMappings(map[string]any{
			"assistant_name": "Alice Agently",
		})).
		Input("What's your name?").
		GetData(core.GetDataOptions{Type: "parsed"}, core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Printf("[quick prompt methods] %#v\n", quickResult)
	return nil
}

func step03OutputFormatControl(agentlyApp *agently.Main) error {
	fmt.Println("\n=== 03-output_format_control ===")
	agent := agentlyApp.CreateAgent("step-03")
	result, err := agent.
		Input("Please explain recursion").
		Output(map[string]any{
			"thinking":    "string",
			"explanation": "string",
			"example_codes": []any{
				"string",
			},
			"practices": []any{
				map[string]any{
					"question": "string",
					"answer":   "string",
				},
			},
		}).
		GetData(
			core.GetDataOptions{
				Type:               "parsed",
				EnsureKeys:         []string{"practices[*].question", "practices[*].answer"},
				KeyStyle:           "dot",
				MaxRetries:         1,
				RaiseEnsureFailure: false,
			},
			core.WithTimeout(timeout),
		)
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", result)
	return nil
}

func step04ConfigurePrompt(agentlyApp *agently.Main) error {
	fmt.Println("\n=== 04-configure_prompt ===")
	agent := agentlyApp.CreateAgent("step-04")

	yamlPrompt := `
.agent:
  system: You are an Agently enhanced agent.
.request:
  input: Say hello.
  output:
    reply:
      $type: str
`
	if err := agent.LoadYAMLPrompt(yamlPrompt); err != nil {
		return err
	}
	result, err := agent.
		SetRequestPrompt("input", "Explain recursion in one paragraph.").
		GetData(core.GetDataOptions{Type: "parsed"}, core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Printf("[yaml result] %#v\n", result)

	jsonPrompt, err := agent.GetJSONPrompt()
	if err != nil {
		return err
	}
	fmt.Printf("[json prompt] %s\n", jsonPrompt)
	return nil
}

func step05ResponseResult(agentlyApp *agently.Main) error {
	fmt.Println("\n=== 05-response_result ===")
	agent := agentlyApp.CreateAgent("step-05")
	response := agent.
		Input("Please explain recursion with a short example.").
		Output(map[string]any{
			"definition": "string",
			"example":    "string",
		}).
		GetResponse()

	text, err := response.Result.GetText(core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	data, err := response.Result.GetData(core.GetDataOptions{Type: "parsed"}, core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	dataObject, err := response.Result.GetDataObject(core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	meta, err := response.Result.GetMeta(core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Printf("[text] %s\n", text)
	fmt.Printf("[data] %#v\n", data)
	fmt.Printf("[data_object] %#v\n", dataObject)
	fmt.Printf("[meta] %#v\n", meta)

	stream := agent.
		Input("List 3 recursion tips.").
		Output(map[string]any{"tips": []any{"string"}})
	gen, err := stream.GetGenerator("delta", core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Print("[delta] ")
	for item := range gen {
		fmt.Print(item)
	}
	fmt.Println()
	return nil
}

func step06Streaming(agentlyApp *agently.Main) error {
	fmt.Println("\n=== 06-streaming ===")
	agent := agentlyApp.CreateAgent("step-06")

	delta, err := agent.Input("Give me a short speech about recursion.").GetGenerator("delta", core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Print("[delta] ")
	for item := range delta {
		fmt.Print(item)
	}
	fmt.Println()

	instant, err := agent.
		Input("Explain recursion with a short definition and two tips.").
		Output(map[string]any{
			"definition": "string",
			"tips":       []any{"string"},
		}).
		GetGenerator("instant", core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Println("[instant]")
	for item := range instant {
		data, ok := item.(types.StreamingData)
		if !ok {
			continue
		}
		if data.Delta == "" {
			continue
		}
		fmt.Printf("path=%s wildcard=%s delta=%q\n", data.Path, data.WildcardPath, data.Delta)
	}

	specific, err := agent.
		Input("Tell me a short story about recursion.").
		GetGenerator("specific", core.WithSpecific("reasoning_delta", "delta", "tool_calls"), core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Println("[specific]")
	for item := range specific {
		message, ok := item.(types.ResponseMessage)
		if !ok {
			continue
		}
		fmt.Printf("event=%s data=%v\n", message.Event, message.Data)
	}
	return nil
}

func step07Tools(agentlyApp *agently.Main) error {
	fmt.Println("\n=== 07-tools ===")
	agent := agentlyApp.CreateAgent("step-07")
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
	result, err := agent.Input("Calculate 345 + 678 with tools.").GetData(core.GetDataOptions{Type: "parsed"}, core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", result)
	return nil
}

func step08ChatHistory(agentlyApp *agently.Main) error {
	fmt.Println("\n=== 08-chat_history ===")
	agent := agentlyApp.CreateAgent("step-08")
	agent.SetChatHistory([]types.ChatMessage{
		{Role: "user", Content: "Hi, who are you?"},
		{Role: "assistant", Content: "I'm an Agently assistant."},
	})
	result, err := agent.Input("What did I ask you before?").GetText(core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Printf("[result] %s\n", result)

	agent.AddChatHistory([]types.ChatMessage{
		{Role: "user", Content: result},
	})
	followUp, err := agent.Input("Summarize my last message in one sentence.").GetText(core.WithTimeout(timeout))
	if err != nil {
		return err
	}
	fmt.Printf("[follow_up] %s\n", followUp)
	agent.ResetChatHistory()
	return nil
}

func main() {
	agentlyApp := configureMain()

	step01Settings(agentlyApp)
	if err := step02PromptMethods(agentlyApp); err != nil {
		panic(err)
	}
	if err := step03OutputFormatControl(agentlyApp); err != nil {
		panic(err)
	}
	if err := step04ConfigurePrompt(agentlyApp); err != nil {
		panic(err)
	}
	if err := step05ResponseResult(agentlyApp); err != nil {
		panic(err)
	}
	if err := step06Streaming(agentlyApp); err != nil {
		panic(err)
	}
	if err := step07Tools(agentlyApp); err != nil {
		panic(err)
	}
	if err := step08ChatHistory(agentlyApp); err != nil {
		panic(err)
	}
}
