package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
)

const formTimeout = 90 * time.Second

type formField struct {
	Key      string
	Question string
	Desc     string
}

var formFields = []formField{
	{Key: "full_name", Question: "What is your full name?", Desc: "customer full name"},
	{Key: "email", Question: "What email should we use for updates?", Desc: "customer email"},
	{Key: "product", Question: "Which product would you like to pre-order?", Desc: "product name"},
	{Key: "quantity", Question: "How many units do you want to reserve?", Desc: "order quantity"},
	{Key: "delivery_city", Question: "Which city should we deliver to?", Desc: "delivery city"},
}

var productSuggestions = []string{
	"Agently Smart Watch",
	"Agently Home Hub",
	"Agently Travel Charger",
}

const roleSettings = "You are a friendly customer experience assistant at a startup. " +
	"You introduce yourself, explain the purpose of the interview, and respond with empathy."

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

func resetForm(formData map[string]string) {
	for key := range formData {
		delete(formData, key)
	}
}

func nextPendingField(formData map[string]string, skipped map[string]struct{}) *formField {
	for i := range formFields {
		field := formFields[i]
		if _, done := formData[field.Key]; done {
			continue
		}
		if _, skip := skipped[field.Key]; skip {
			continue
		}
		return &field
	}
	return nil
}

func getStringByPath(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func generateQuestion(agent *agently.Main, field formField, formData map[string]string, hint string) (string, error) {
	req := agent.CreateAgent("preorder-question").CreateTempRequest()
	result, err := req.
		Input("Generate the next interview question.").
		Instruct("Ask a friendly, concise question. Avoid repeating collected details.").
		Info(map[string]any{
			"field_key":              field.Key,
			"field_desc":             field.Desc,
			"field_default_question": field.Question,
			"current_form_data":      formData,
			"hint":                   hint,
		}).
		Output(map[string]any{"question": "string"}).
		GetData(
			core.GetDataOptions{
				Type:               "parsed",
				EnsureKeys:         []string{"question"},
				MaxRetries:         2,
				RaiseEnsureFailure: true,
			},
			core.WithTimeout(formTimeout),
		)
	if err != nil {
		return "", err
	}
	typed, _ := result.(map[string]any)
	question := getStringByPath(typed, "question")
	if question == "" {
		question = field.Question
	}
	return question, nil
}

func generateStartupMessage(agentlyApp *agently.Main) (string, error) {
	req := agentlyApp.CreateAgent("preorder-startup").CreateTempRequest()
	result, err := req.
		Input("Generate a short welcome message.").
		Instruct("Introduce yourself and explain you are collecting pre-order details. Keep it warm and under 2 sentences.").
		Output(map[string]any{"message": "string"}).
		GetData(
			core.GetDataOptions{
				Type:               "parsed",
				EnsureKeys:         []string{"message"},
				MaxRetries:         2,
				RaiseEnsureFailure: true,
			},
			core.WithTimeout(formTimeout),
		)
	if err != nil {
		return "", err
	}
	typed, _ := result.(map[string]any)
	return getStringByPath(typed, "message"), nil
}

func parseAnswer(agentlyApp *agently.Main, field formField, userAnswer string) (string, error) {
	req := agentlyApp.CreateAgent("preorder-parse").CreateTempRequest()
	result, err := req.
		Input(userAnswer).
		Instruct("Extract the user answer for the requested field. Return plain text only.").
		Info(map[string]any{
			"field_key":  field.Key,
			"field_desc": field.Desc,
		}).
		Output(map[string]any{
			field.Key: "string",
		}).
		GetData(
			core.GetDataOptions{
				Type:               "parsed",
				EnsureKeys:         []string{field.Key},
				MaxRetries:         2,
				RaiseEnsureFailure: true,
			},
			core.WithTimeout(formTimeout),
		)
	if err != nil {
		return "", err
	}
	typed, _ := result.(map[string]any)
	return getStringByPath(typed, field.Key), nil
}

func classifyIntent(agentlyApp *agently.Main, field formField, userAnswer string) (string, error) {
	req := agentlyApp.CreateAgent("preorder-intent").CreateTempRequest()
	result, err := req.
		Input(userAnswer).
		Instruct("Classify user intent. Use one of: answer, unknown, refuse, exit, ask_suggestion.").
		Info(map[string]any{
			"field_key":       field.Key,
			"field_desc":      field.Desc,
			"allowed_intents": []string{"answer", "unknown", "refuse", "exit", "ask_suggestion"},
		}).
		Output(map[string]any{"intent": "string"}).
		GetData(
			core.GetDataOptions{
				Type:               "parsed",
				EnsureKeys:         []string{"intent"},
				MaxRetries:         2,
				RaiseEnsureFailure: true,
			},
			core.WithTimeout(formTimeout),
		)
	if err != nil {
		return "", err
	}
	typed, _ := result.(map[string]any)
	intent := strings.ToLower(getStringByPath(typed, "intent"))
	switch intent {
	case "answer", "unknown", "refuse", "exit", "ask_suggestion":
		return intent, nil
	default:
		return "answer", nil
	}
}

func validateValue(fieldKey string, value string) (bool, string) {
	if strings.TrimSpace(value) == "" {
		return false, "It looks empty."
	}
	if fieldKey == "email" {
		if !strings.Contains(value, "@") || !strings.Contains(value, ".") {
			return false, "Please provide a valid email address."
		}
	}
	if fieldKey == "quantity" {
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return false, "Please provide a positive number."
		}
	}
	return true, ""
}

func normalizeYesNo(userInput string) string {
	normalized := strings.TrimSpace(strings.ToLower(userInput))
	switch normalized {
	case "yes", "y", "sure", "ok", "okay":
		return "yes"
	case "no", "n", "stop", "exit", "quit":
		return "no"
	default:
		return "unknown"
	}
}

func wantsResetOrExit(userInput string) string {
	normalized := strings.TrimSpace(strings.ToLower(userInput))
	if normalized == "reset" {
		return "reset"
	}
	switch normalized {
	case "bye", "exit", "quit", "stop":
		return "exit"
	default:
		return ""
	}
}

func readLine(scanner *bufio.Scanner, label string) string {
	fmt.Print(label)
	if !scanner.Scan() {
		return "exit"
	}
	return strings.TrimSpace(scanner.Text())
}

func interviewWithResetDemo(agentlyApp *agently.Main) error {
	agent := agentlyApp.CreateAgent("preorder-interview")
	agent.Role(roleSettings, core.Always())

	formData := map[string]string{}
	skipped := map[string]struct{}{}
	attempts := map[string]int{}
	scanner := bufio.NewScanner(os.Stdin)

	startupMessage, err := generateStartupMessage(agentlyApp)
	if err != nil {
		return err
	}
	fmt.Printf("[assistant] %s\n", startupMessage)
	fmt.Println("Type 'reset' anytime to start over.")
	fmt.Println()

	for {
		pending := nextPendingField(formData, skipped)
		if pending == nil {
			if len(skipped) > 0 {
				keys := make([]string, 0, len(skipped))
				for key := range skipped {
					keys = append(keys, key)
				}
				fmt.Printf("\n[assistant] We still need: %s. Continue? (yes/no)\n", strings.Join(keys, ", "))
				cont := readLine(scanner, "[user] ")
				switch wantsResetOrExit(cont) {
				case "reset":
					resetForm(formData)
					skipped = map[string]struct{}{}
					attempts = map[string]int{}
					agent.ResetChatHistory()
					fmt.Println("[system] Form reset. Let's start again.")
					fmt.Println()
					continue
				case "exit":
					fmt.Println("\n[assistant] No problem. See you next time!")
					return nil
				}
				if normalizeYesNo(cont) != "yes" {
					fmt.Println("\n[assistant] No problem. See you next time!")
					return nil
				}
				skipped = map[string]struct{}{}
				continue
			}
			break
		}

		question, err := generateQuestion(agentlyApp, *pending, formData, "")
		if err != nil {
			return err
		}
		fmt.Printf("[assistant] %s\n", question)
		userInput := readLine(scanner, "[user] ")

		switch wantsResetOrExit(userInput) {
		case "reset":
			resetForm(formData)
			skipped = map[string]struct{}{}
			attempts = map[string]int{}
			agent.ResetChatHistory()
			fmt.Println("[system] Form reset. Let's start again.")
			fmt.Println()
			continue
		case "exit":
			fmt.Println("\n[assistant] Thanks for stopping by. Goodbye!")
			return nil
		}

		intent, err := classifyIntent(agentlyApp, *pending, userInput)
		if err != nil {
			return err
		}
		if intent == "exit" {
			fmt.Println("\n[assistant] Thanks for stopping by. Goodbye!")
			return nil
		}
		if intent == "ask_suggestion" && pending.Key == "product" {
			fmt.Printf("[assistant] Here are some options: %s. Which one fits?\n", strings.Join(productSuggestions, ", "))
			userInput = readLine(scanner, "[user] ")
			switch wantsResetOrExit(userInput) {
			case "reset":
				resetForm(formData)
				skipped = map[string]struct{}{}
				attempts = map[string]int{}
				agent.ResetChatHistory()
				fmt.Println("[system] Form reset. Let's start again.")
				fmt.Println()
				continue
			case "exit":
				fmt.Println("\n[assistant] Thanks for stopping by. Goodbye!")
				return nil
			}
		}

		if intent == "unknown" || intent == "refuse" {
			attempts[pending.Key]++
			if attempts[pending.Key] >= 2 {
				skipped[pending.Key] = struct{}{}
				fmt.Println("[assistant] Thanks for sharing. We'll skip this for now.")
				fmt.Println()
				continue
			}
			clarification, err := generateQuestion(agentlyApp, *pending, formData, "The user was unsure or refused. Ask gently if they can share it.")
			if err != nil {
				return err
			}
			fmt.Printf("[assistant] %s\n", clarification)
			userInput = readLine(scanner, "[user] ")
		}

		value, err := parseAnswer(agentlyApp, *pending, userInput)
		if err != nil {
			return err
		}
		valid, reason := validateValue(pending.Key, value)
		if !valid {
			attempts[pending.Key]++
			if attempts[pending.Key] >= 2 {
				skipped[pending.Key] = struct{}{}
				fmt.Println("[assistant] No worries. We'll skip this for now.")
				fmt.Println()
				continue
			}
			retryQuestion, err := generateQuestion(agentlyApp, *pending, formData, reason)
			if err != nil {
				return err
			}
			fmt.Printf("[assistant] %s\n", retryQuestion)
			retryInput := readLine(scanner, "[user] ")
			value, err = parseAnswer(agentlyApp, *pending, retryInput)
			if err != nil {
				return err
			}
			valid, _ = validateValue(pending.Key, value)
			if !valid {
				skipped[pending.Key] = struct{}{}
				fmt.Println("[assistant] Thanks for trying. We'll skip this for now.")
				fmt.Println()
				continue
			}
			userInput = retryInput
		}

		formData[pending.Key] = value
		agent.AddChatHistory([]types.ChatMessage{{Role: "assistant", Content: question}})
		agent.AddChatHistory([]types.ChatMessage{{Role: "user", Content: userInput}})
	}

	fmt.Println("\n[assistant] Thanks! Here is the pre-order info collected:")
	for _, field := range formFields {
		if value, ok := formData[field.Key]; ok {
			fmt.Printf("- %s: %s\n", field.Key, value)
		}
	}
	if len(skipped) > 0 {
		keys := make([]string, 0, len(skipped))
		for key := range skipped {
			keys = append(keys, key)
		}
		fmt.Printf("\n[assistant] Missing: %s\n", strings.Join(keys, ", "))
	}
	return nil
}

func main() {
	agentlyApp := configureMain()
	if err := interviewWithResetDemo(agentlyApp); err != nil {
		panic(err)
	}
}
