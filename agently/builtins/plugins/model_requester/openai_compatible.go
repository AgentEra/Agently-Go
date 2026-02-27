package modelrequester

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type OpenAICompatible struct {
	prompt   *core.Prompt
	settings *utils.Settings

	pluginSettings *utils.RuntimeDataNamespace
	modelType      string
}

const PluginName = "OpenAICompatible"

var DefaultSettings = map[string]any{
	"$mappings": map[string]any{
		"path_mappings": map[string]any{
			"OpenAICompatible": "plugins.ModelRequester.OpenAICompatible",
			"OpenAI":           "plugins.ModelRequester.OpenAICompatible",
			"OAIClient":        "plugins.ModelRequester.OpenAICompatible",
		},
	},
	"model_type": "chat",
	"model":      nil,
	"default_model": map[string]any{
		"chat":        "gpt-4.1",
		"completions": "gpt-3.5-turbo-instruct",
		"embeddings":  "text-embedding-ada-002",
	},
	"client_options":  map[string]any{},
	"headers":         map[string]any{},
	"proxy":           nil,
	"request_options": map[string]any{},
	"base_url":        "https://api.openai.com/v1",
	"full_url":        nil,
	"path_mapping": map[string]any{
		"chat":        "/chat/completions",
		"completions": "/completions",
		"embeddings":  "/embeddings",
	},
	"auth":                           nil,
	"stream":                         true,
	"rich_content":                   false,
	"strict_role_orders":             true,
	"yield_extra_content_separately": true,
	"content_mapping_style":          "dot",
	"content_mapping": map[string]any{
		"id":            "id",
		"role":          "choices[0].delta.role",
		"reasoning":     "choices[0].delta.reasoning_content",
		"delta":         "choices[0].delta.content",
		"tool_calls":    "choices[0].delta.tool_calls",
		"done":          nil,
		"usage":         "usage",
		"finish_reason": "choices[0].finish_reason",
		"extra_delta": map[string]any{
			"function_call": "choices[0].delta.function_call",
		},
		"extra_done": nil,
	},
	"timeout": map[string]any{
		"connect": 30.0,
		"read":    600.0,
		"write":   30.0,
		"pool":    30.0,
	},
}

func New(prompt *core.Prompt, settings *utils.Settings) core.ModelRequester {
	ns := settings.Namespace("plugins.ModelRequester.OpenAICompatible").RuntimeDataNamespace
	modelType := fmt.Sprint(ns.Get("model_type", "chat", true))
	if modelType == "" {
		modelType = "chat"
	}
	if prompt.Get("attachment", nil, true) != nil {
		ns.Set("rich_content", true)
	}
	return &OpenAICompatible{prompt: prompt, settings: settings, pluginSettings: ns, modelType: modelType}
}

func (m *OpenAICompatible) GenerateRequestData() (types.RequestData, error) {
	requestData := types.RequestData{
		ClientOptions: map[string]any{},
		Headers:       http.Header{},
		Data:          map[string]any{},
		RequestOpts:   map[string]any{},
		RequestURL:    "",
	}

	switch m.modelType {
	case "chat":
		messages, err := m.prompt.ToMessages(
			core.WithRichContent(m.pluginSettings.Get("rich_content", false, true) == true),
			core.WithStrictRoleOrders(m.pluginSettings.Get("strict_role_orders", true, true) != false),
		)
		if err != nil {
			return requestData, err
		}
		requestData.Data["messages"] = messages
	case "completions":
		text, err := m.prompt.ToText()
		if err != nil {
			return requestData, err
		}
		requestData.Data["prompt"] = text
	case "embeddings":
		input := utils.DataFormatterSanitize(m.prompt.Get("input", nil, true), false)
		requestData.Data["input"] = input
	default:
		return requestData, fmt.Errorf("unsupported model_type: %s", m.modelType)
	}

	headers := utils.DataFormatterToStrKeyDict(m.pluginSettings.Get("headers", map[string]any{}, true), "str", "", map[string]any{})
	requestData.Headers = http.Header{}
	for k, v := range headers {
		requestData.Headers.Set(k, fmt.Sprint(v))
	}
	requestData.Headers.Set("Connection", "close")

	clientOptions := utils.DataFormatterToStrKeyDict(m.pluginSettings.Get("client_options", map[string]any{}, true), "", "", map[string]any{})
	requestData.ClientOptions = clientOptions

	requestOptions := utils.DataFormatterToStrKeyDict(m.pluginSettings.Get("request_options", map[string]any{}, true), "serializable", "", map[string]any{})
	if promptOptions, ok := m.prompt.Get("options", map[string]any{}, true).(map[string]any); ok {
		for k, v := range promptOptions {
			requestOptions[k] = v
		}
	}

	model := m.pluginSettings.Get("model", nil, true)
	if model == nil || fmt.Sprint(model) == "" {
		defaults := utils.DataFormatterToStrKeyDict(m.pluginSettings.Get("default_model", map[string]any{}, true), "", m.modelType, map[string]any{})
		model = defaults[m.modelType]
	}
	requestOptions["model"] = model

	isStream := true
	if streamValue := m.pluginSettings.Get("stream", nil, true); streamValue != nil {
		if b, ok := streamValue.(bool); ok {
			isStream = b
		}
	}
	if m.modelType == "embeddings" {
		isStream = false
	}
	requestOptions["stream"] = isStream
	requestData.RequestOpts = requestOptions
	requestData.Stream = isStream

	fullURL := fmt.Sprint(m.pluginSettings.Get("full_url", "", true))
	if strings.TrimSpace(fullURL) != "" && fullURL != "<nil>" {
		requestData.RequestURL = fullURL
	} else {
		baseURL := strings.TrimRight(fmt.Sprint(m.pluginSettings.Get("base_url", "https://api.openai.com/v1", true)), "/")
		pathMap := utils.DataFormatterToStrKeyDict(m.pluginSettings.Get("path_mapping", map[string]any{}, true), "str", "", map[string]any{})
		path := fmt.Sprint(pathMap[m.modelType])
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		if path == "/<nil>" || path == "/" {
			switch m.modelType {
			case "chat":
				path = "/chat/completions"
			case "completions":
				path = "/completions"
			case "embeddings":
				path = "/embeddings"
			}
		}
		requestData.RequestURL = baseURL + path
	}

	return requestData, nil
}

func (m *OpenAICompatible) RequestModel(ctx context.Context, requestData types.RequestData) (<-chan types.ResponseMessage, error) {
	out := make(chan types.ResponseMessage, 128)
	go func() {
		defer close(out)

		payload := map[string]any{}
		for k, v := range requestData.Data {
			payload[k] = v
		}
		for k, v := range requestData.RequestOpts {
			payload[k] = v
		}

		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestData.RequestURL, bytes.NewReader(body))
		if err != nil {
			out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
			return
		}
		req.Header = requestData.Headers.Clone()
		req.Header.Set("Content-Type", "application/json")

		apiKey := fmt.Sprint(m.pluginSettings.Get("api_key", "", true))
		if auth, ok := m.pluginSettings.Get("auth", nil, true).(map[string]any); ok {
			if key, ok := auth["api_key"]; ok && fmt.Sprint(key) != "" {
				apiKey = fmt.Sprint(key)
			}
		}
		if strings.TrimSpace(apiKey) != "" && apiKey != "<nil>" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		timeout := 120 * time.Second
		if tmap, ok := m.pluginSettings.Get("timeout", map[string]any{}, true).(map[string]any); ok {
			if read, ok := tmap["read"].(float64); ok && read > 0 {
				timeout = time.Duration(read * float64(time.Second))
			}
		}
		client := &http.Client{Timeout: timeout}
		resp, err := client.Do(req)
		if err != nil {
			out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			b, _ := io.ReadAll(resp.Body)
			out <- types.ResponseMessage{Event: types.ResponseEventError, Data: fmt.Errorf("status code %d: %s", resp.StatusCode, string(b))}
			return
		}

		if requestData.Stream && (m.modelType == "chat" || m.modelType == "completions") {
			scanner := bufio.NewScanner(resp.Body)
			scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "data:") {
					line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				}
				if line == "[DONE]" {
					out <- types.ResponseMessage{Event: types.ResponseEventOriginalDone, Data: "[DONE]"}
					continue
				}
				out <- types.ResponseMessage{Event: types.ResponseEventOriginalDelta, Data: line}
			}
			if err := scanner.Err(); err != nil {
				out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
			}
			return
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
			return
		}
		out <- types.ResponseMessage{Event: types.ResponseEventOriginalDelta, Data: string(b)}
		out <- types.ResponseMessage{Event: types.ResponseEventOriginalDone, Data: "[DONE]"}
	}()
	return out, nil
}

func (m *OpenAICompatible) BroadcastResponse(_ context.Context, source <-chan types.ResponseMessage) (<-chan types.ResponseMessage, error) {
	out := make(chan types.ResponseMessage, 128)
	go func() {
		defer close(out)

		meta := map[string]any{}
		messageRecord := map[string]any{}
		reasoningBuffer := strings.Builder{}
		contentBuffer := strings.Builder{}

		contentMapping, _ := m.pluginSettings.Get("content_mapping", map[string]any{}, true).(map[string]any)
		mappingStyle := fmt.Sprint(m.pluginSettings.Get("content_mapping_style", "dot", true))
		if mappingStyle != "dot" && mappingStyle != "slash" {
			mappingStyle = "dot"
		}

		for msg := range source {
			if msg.Event == types.ResponseEventError {
				out <- msg
				continue
			}

			raw := fmt.Sprint(msg.Data)
			if raw == "[DONE]" {
				doneData := locateByMapping(messageRecord, contentMapping["done"], mappingStyle)
				if doneData == nil {
					doneData = contentBuffer.String()
				}
				out <- types.ResponseMessage{Event: types.ResponseEventDone, Data: doneData}

				reasoningDone := locateByMapping(messageRecord, contentMapping["reasoning"], mappingStyle)
				if reasoningDone == nil {
					reasoningDone = reasoningBuffer.String()
				}
				out <- types.ResponseMessage{Event: types.ResponseEventReasoningDone, Data: reasoningDone}
				out <- types.ResponseMessage{Event: types.ResponseEventOriginalDone, Data: messageRecord}

				if finishReason := locateByMapping(messageRecord, contentMapping["finish_reason"], mappingStyle); finishReason != nil {
					meta["finish_reason"] = finishReason
				}
				if usage := locateByMapping(messageRecord, contentMapping["usage"], mappingStyle); usage != nil {
					meta["usage"] = usage
				}
				out <- types.ResponseMessage{Event: types.ResponseEventMeta, Data: meta}
				continue
			}

			out <- types.ResponseMessage{Event: types.ResponseEventOriginalDelta, Data: raw}

			loaded := map[string]any{}
			if err := json.Unmarshal([]byte(raw), &loaded); err != nil {
				continue
			}
			messageRecord = loaded

			if _, ok := meta["id"]; !ok {
				if id := locateByMapping(loaded, contentMapping["id"], mappingStyle); id != nil {
					meta["id"] = id
				}
			}
			if _, ok := meta["role"]; !ok {
				if role := locateByMapping(loaded, contentMapping["role"], mappingStyle); role != nil {
					meta["role"] = role
				}
			}

			if reasoning := locateByMapping(loaded, contentMapping["reasoning"], mappingStyle); reasoning != nil {
				reasoningText := fmt.Sprint(reasoning)
				reasoningBuffer.WriteString(reasoningText)
				out <- types.ResponseMessage{Event: types.ResponseEventReasoning, Data: reasoningText}
			}
			if delta := locateByMapping(loaded, contentMapping["delta"], mappingStyle); delta != nil {
				deltaText := fmt.Sprint(delta)
				contentBuffer.WriteString(deltaText)
				out <- types.ResponseMessage{Event: types.ResponseEventDelta, Data: deltaText}
			}
			if toolCalls := locateByMapping(loaded, contentMapping["tool_calls"], mappingStyle); toolCalls != nil {
				out <- types.ResponseMessage{Event: types.ResponseEventToolCalls, Data: toolCalls}
			}
			if extraDelta, ok := contentMapping["extra_delta"].(map[string]any); ok {
				for extraKey, extraPath := range extraDelta {
					if value := locateByMapping(loaded, extraPath, mappingStyle); value != nil {
						out <- types.ResponseMessage{Event: types.ResponseEventExtra, Data: map[string]any{extraKey: value}}
					}
				}
			}
		}
	}()
	return out, nil
}

func locateByMapping(data map[string]any, mapping any, style string) any {
	if mapping == nil {
		return nil
	}
	path := strings.TrimSpace(fmt.Sprint(mapping))
	if path == "" || path == "<nil>" {
		return nil
	}
	return utils.LocatePathInData(data, path, style, nil)
}
