package agentextensions

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
	"gopkg.in/yaml.v3"
)

type SessionExtension struct {
	agent    *core.BaseAgent
	sessions map[string]*core.Session
	active   *core.Session
	mu       sync.RWMutex
}

func NewSessionExtension(agent *core.BaseAgent) *SessionExtension {
	ext := &SessionExtension{
		agent:    agent,
		sessions: map[string]*core.Session{},
	}
	agent.Settings().SetDefault("session.input_keys", nil, true)
	agent.Settings().SetDefault("session.reply_keys", nil, true)

	agent.ExtensionHandlers().AppendRequestPrefix(ext.sessionRequestPrefix)
	agent.ExtensionHandlers().AppendFinally(ext.sessionFinally)
	return ext
}

func (e *SessionExtension) ActiveSession() *core.Session {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.active
}

func (e *SessionExtension) ActivateSession(sessionID string) *SessionExtension {
	e.mu.Lock()
	defer e.mu.Unlock()
	if sessionID == "" {
		sessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	if existing, ok := e.sessions[sessionID]; ok {
		e.active = existing
	} else {
		s := core.NewSession(sessionID, true, e.agent.Settings())
		e.sessions[sessionID] = s
		e.active = s
	}
	e.refillAgentChatHistoryWithSessionLocked()
	return e
}

func (e *SessionExtension) DeactivateSession() *SessionExtension {
	e.mu.Lock()
	e.active = nil
	e.mu.Unlock()
	e.agent.AgentPrompt().Delete("chat_history")
	e.agent.AgentPrompt().Set("chat_history", []any{})
	return e
}

func (e *SessionExtension) ResetChatHistory() *SessionExtension {
	e.mu.RLock()
	active := e.active
	e.mu.RUnlock()
	if active == nil {
		e.agent.ResetChatHistory()
		return e
	}
	active.ResetChatHistory()
	e.mu.Lock()
	e.refillAgentChatHistoryWithSessionLocked()
	e.mu.Unlock()
	return e
}

func (e *SessionExtension) SetChatHistory(chatHistory []types.ChatMessage) *SessionExtension {
	e.mu.RLock()
	active := e.active
	e.mu.RUnlock()
	if active == nil {
		e.agent.SetChatHistory(toAnyChat(chatHistory))
		return e
	}
	active.SetChatHistory(chatHistory)
	e.mu.Lock()
	e.refillAgentChatHistoryWithSessionLocked()
	e.mu.Unlock()
	return e
}

func (e *SessionExtension) AddChatHistory(chatHistory []types.ChatMessage) *SessionExtension {
	e.mu.RLock()
	active := e.active
	e.mu.RUnlock()
	if active == nil {
		for _, item := range chatHistory {
			e.agent.AddChatHistory(map[string]any{"role": item.Role, "content": item.Content})
		}
		return e
	}
	active.AddChatHistory(chatHistory)
	e.mu.Lock()
	e.refillAgentChatHistoryWithSessionLocked()
	e.mu.Unlock()
	return e
}

func (e *SessionExtension) CleanContextWindow() *SessionExtension {
	e.mu.RLock()
	active := e.active
	e.mu.RUnlock()
	if active == nil {
		e.agent.AgentPrompt().Delete("chat_history")
		return e
	}
	active.CleanContextWindow()
	e.mu.Lock()
	e.refillAgentChatHistoryWithSessionLocked()
	e.mu.Unlock()
	return e
}

func (e *SessionExtension) refillAgentChatHistoryWithSessionLocked() {
	if e.active == nil {
		return
	}
	e.agent.AgentPrompt().Delete("chat_history")
	e.agent.AgentPrompt().Set("chat_history", toAnyChat(e.active.ContextWindow()))
}

func (e *SessionExtension) sessionRequestPrefix(_ context.Context, prompt *core.Prompt, _ *utils.Settings) error {
	e.mu.RLock()
	active := e.active
	e.mu.RUnlock()
	if active == nil {
		return nil
	}
	prompt.Delete("chat_history")
	prompt.Set("chat_history", toAnyChat(active.ContextWindow()))
	if memo := active.Memo(); memo != nil {
		prompt.Set("CHAT SESSION MEMO", memo)
	}
	return nil
}

func (e *SessionExtension) sessionFinally(ctx context.Context, result *core.ModelResponseResult, _ *utils.Settings) error {
	e.mu.RLock()
	active := e.active
	e.mu.RUnlock()
	if active == nil {
		return nil
	}

	inputKeys, inputAll := normalizeSessionKeys(e.agent.Settings().Get("session.input_keys", nil, true))
	replyKeys, replyAll := normalizeSessionKeys(e.agent.Settings().Get("session.reply_keys", nil, true))

	userContent := ""
	requestPrompt := result.Prompt()
	if requestPrompt != nil {
		if inputAll {
			if promptText, err := requestPrompt.ToText(); err == nil {
				userContent = promptText
			}
		} else {
			requestData, _ := requestPrompt.ToSerializablePromptData(true)
			userItems := make([][2]any, 0)
			for _, key := range inputKeys {
				if found, value := e.extractInputValue(requestData, key); found {
					userItems = append(userItems, [2]any{key, value})
				}
			}
			if len(userItems) > 0 {
				userContent = formatKeyedContent(userItems)
			}
		}
	}

	assistantContent := ""
	if parsedData, err := result.PeekData("parsed", ctx); err == nil {
		if replyAll {
			assistantContent = formatValue(parsedData)
		} else {
			replyItems := make([][2]any, 0)
			for _, key := range replyKeys {
				if found, value := extractByPath(parsedData, key); found {
					replyItems = append(replyItems, [2]any{key, value})
				}
			}
			if len(replyItems) > 0 {
				assistantContent = formatKeyedContent(replyItems)
			}
		}
	}

	if strings.TrimSpace(userContent) != "" {
		active.AddChatHistory([]types.ChatMessage{{Role: "user", Content: userContent}})
	}
	if strings.TrimSpace(assistantContent) != "" {
		active.AddChatHistory([]types.ChatMessage{{Role: "assistant", Content: assistantContent}})
	}

	e.mu.Lock()
	e.refillAgentChatHistoryWithSessionLocked()
	e.mu.Unlock()
	return nil
}

func normalizeSessionKeys(value any) ([]string, bool) {
	if value == nil {
		return nil, true
	}
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return []string{}, false
		}
		return []string{trimmed}, false
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out, false
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			trimmed := strings.TrimSpace(fmt.Sprint(item))
			if trimmed != "" && trimmed != "<nil>" {
				out = append(out, trimmed)
			}
		}
		return out, false
	default:
		return []string{}, false
	}
}

func extractByPath(data any, key string) (bool, any) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, nil
	}
	if m, ok := data.(map[string]any); ok {
		if value, exists := m[key]; exists {
			return true, value
		}
	}
	style := "dot"
	if strings.Contains(key, "/") {
		style = "slash"
	}
	marker := &struct{}{}
	value := utils.LocatePathInData(data, key, style, marker)
	if value == marker {
		return false, nil
	}
	return true, value
}

func (e *SessionExtension) extractInputValue(promptData map[string]any, key string) (bool, any) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, nil
	}
	if key == ".request" {
		return true, promptData
	}
	if strings.HasPrefix(key, ".request.") {
		return extractByPath(promptData, strings.TrimPrefix(key, ".request."))
	}
	if key == ".agent" {
		agentData, _ := e.agent.AgentPrompt().ToSerializablePromptData(true)
		return true, agentData
	}
	if strings.HasPrefix(key, ".agent.") {
		agentData, _ := e.agent.AgentPrompt().ToSerializablePromptData(true)
		return extractByPath(agentData, strings.TrimPrefix(key, ".agent."))
	}
	if found, value := extractByPath(promptData, key); found {
		return true, value
	}
	inputData, ok := promptData["input"].(map[string]any)
	if !ok {
		return false, nil
	}
	if strings.HasPrefix(key, "input.") {
		return extractByPath(inputData, strings.TrimPrefix(key, "input."))
	}
	return extractByPath(inputData, key)
}

func formatValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return fmt.Sprint(typed)
	case nil:
		return "null"
	default:
		if b, err := yaml.Marshal(typed); err == nil {
			return strings.TrimSpace(string(b))
		}
		return fmt.Sprint(typed)
	}
}

func formatKeyedContent(items [][2]any) string {
	lines := make([]string, 0, len(items)*2)
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("[%s]:", fmt.Sprint(item[0])))
		lines = append(lines, formatValue(item[1]))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func toAnyChat(in []types.ChatMessage) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{"role": item.Role, "content": item.Content})
	}
	return out
}
