package testkit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	defaultOllamaBaseURL = "http://127.0.0.1:11434/v1"
	defaultOllamaModel   = "qwen2.5:7b"
)

func OllamaBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultOllamaBaseURL
}

func OllamaModel() string {
	if value := strings.TrimSpace(os.Getenv("OLLAMA_MODEL")); value != "" {
		return value
	}
	return defaultOllamaModel
}

func RequireOllamaHealth(t *testing.T) {
	t.Helper()

	baseURL := OllamaBaseURL()
	model := OllamaModel()
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(baseURL + "/models")
	if err != nil {
		t.Fatalf("online regression requires reachable OLLAMA service: base_url=%s err=%v", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		t.Fatalf("online regression health check failed: base_url=%s status=%d", baseURL, resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("online regression models decode failed: %v", err)
	}

	for _, entry := range payload.Data {
		if entry.ID == model {
			return
		}
	}

	available := make([]string, 0, len(payload.Data))
	for _, entry := range payload.Data {
		available = append(available, entry.ID)
	}
	t.Fatalf("online regression model unavailable: required=%s available=%v", model, available)
}

func ApplyOllamaDefaults(settingsSetter func(string, any, bool)) {
	settingsSetter("plugins.ModelRequester.OpenAICompatible.base_url", OllamaBaseURL(), false)
	settingsSetter("plugins.ModelRequester.OpenAICompatible.model", OllamaModel(), false)
	settingsSetter("plugins.ModelRequester.OpenAICompatible.stream", true, false)
	settingsSetter("plugins.ModelRequester.OpenAICompatible.request_options.temperature", 0.1, false)
	settingsSetter("plugins.ModelRequester.OpenAICompatible.timeout.read", 180.0, false)
}

func OnlineConfigSummary() string {
	return fmt.Sprintf("base_url=%s model=%s", OllamaBaseURL(), OllamaModel())
}
