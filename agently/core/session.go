package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type AnalysisHandler func(fullContext []types.ChatMessage, contextWindow []types.ChatMessage, memo any, sessionSettings *utils.RuntimeDataNamespace) (string, error)
type ExecutionHandler func(fullContext []types.ChatMessage, contextWindow []types.ChatMessage, memo any, sessionSettings *utils.RuntimeDataNamespace) ([]types.ChatMessage, []types.ChatMessage, any, error)

type Session struct {
	id             string
	autoResize     bool
	settings       *utils.Settings
	sessionSetting *utils.RuntimeDataNamespace

	analysisHandler  AnalysisHandler
	executionHandler map[string]ExecutionHandler

	fullContext   []types.ChatMessage
	contextWindow []types.ChatMessage
	memo          any
	mu            sync.RWMutex
}

func NewSession(id string, autoResize bool, settings *utils.Settings) *Session {
	if id == "" {
		id = randomID()
	}
	if settings == nil {
		settings = NewDefaultSettings(nil)
	}
	s := &Session{
		id:               id,
		autoResize:       autoResize,
		settings:         utils.NewSettings("Session-Settings", map[string]any{}, settings),
		sessionSetting:   utils.NewSettings("Session-Settings-NS", map[string]any{}, settings).Namespace("session").RuntimeDataNamespace,
		executionHandler: map[string]ExecutionHandler{},
		fullContext:      []types.ChatMessage{},
		contextWindow:    []types.ChatMessage{},
	}
	s.sessionSetting.SetDefault("max_length", nil, true)
	s.analysisHandler = s.defaultAnalysisHandler
	s.executionHandler["simple_cut"] = s.simpleCutExecutionHandler
	return s
}

func (s *Session) ID() string { return s.id }

func (s *Session) defaultAnalysisHandler(fullContext []types.ChatMessage, contextWindow []types.ChatMessage, _ any, sessionSettings *utils.RuntimeDataNamespace) (string, error) {
	maxLength, _ := sessionSettings.Get("max_length", nil, true).(int)
	if maxLength > 0 && s.calculateContextLength(contextWindow) > maxLength {
		return "simple_cut", nil
	}
	return "", nil
}

func (s *Session) simpleCutExecutionHandler(_ []types.ChatMessage, contextWindow []types.ChatMessage, _ any, sessionSettings *utils.RuntimeDataNamespace) ([]types.ChatMessage, []types.ChatMessage, any, error) {
	maxLength, _ := sessionSettings.Get("max_length", nil, true).(int)
	if maxLength <= 0 {
		return nil, nil, nil, nil
	}
	newWindow := make([]types.ChatMessage, 0)
	total := 0
	for i := len(contextWindow) - 1; i >= 0; i-- {
		message := contextWindow[i]
		size := len(fmt.Sprint(message))
		if total+size > maxLength {
			break
		}
		total += size
		newWindow = append(newWindow, message)
	}
	for i, j := 0, len(newWindow)-1; i < j; i, j = i+1, j-1 {
		newWindow[i], newWindow[j] = newWindow[j], newWindow[i]
	}
	return nil, newWindow, nil, nil
}

func (s *Session) calculateContextLength(contextWindow []types.ChatMessage) int {
	length := 0
	for _, message := range contextWindow {
		length += len(fmt.Sprint(message))
	}
	return length
}

func (s *Session) RegisterAnalysisHandler(handler AnalysisHandler) *Session {
	s.analysisHandler = handler
	return s
}

func (s *Session) RegisterExecutionHandler(strategyName string, handler ExecutionHandler) *Session {
	s.executionHandler[strategyName] = handler
	return s
}

func (s *Session) ResetChatHistory() *Session {
	s.mu.Lock()
	s.fullContext = []types.ChatMessage{}
	s.contextWindow = []types.ChatMessage{}
	s.mu.Unlock()
	if s.autoResize {
		s.Resize()
	}
	return s
}

func (s *Session) CleanContextWindow() *Session {
	s.mu.Lock()
	s.contextWindow = []types.ChatMessage{}
	s.mu.Unlock()
	if s.autoResize {
		s.Resize()
	}
	return s
}

func (s *Session) SetChatHistory(chatHistory []types.ChatMessage) *Session {
	normalized := normalizeChatMessages(chatHistory)
	s.mu.Lock()
	s.fullContext = append([]types.ChatMessage{}, normalized...)
	s.contextWindow = append([]types.ChatMessage{}, normalized...)
	s.mu.Unlock()
	if s.autoResize {
		s.Resize()
	}
	return s
}

func (s *Session) AddChatHistory(chatHistory []types.ChatMessage) *Session {
	normalized := normalizeChatMessages(chatHistory)
	s.mu.Lock()
	s.fullContext = append(s.fullContext, normalized...)
	s.contextWindow = append(s.contextWindow, normalized...)
	s.mu.Unlock()
	if s.autoResize {
		s.Resize()
	}
	return s
}

func (s *Session) AnalyzeContext() (string, error) {
	s.mu.RLock()
	fullCopy := append([]types.ChatMessage{}, s.fullContext...)
	windowCopy := append([]types.ChatMessage{}, s.contextWindow...)
	memo := s.memo
	s.mu.RUnlock()
	return s.analysisHandler(fullCopy, windowCopy, memo, s.sessionSetting)
}

func (s *Session) ExecuteStrategy(strategyName string) error {
	handler, ok := s.executionHandler[strategyName]
	if !ok {
		return nil
	}
	s.mu.RLock()
	fullCopy := append([]types.ChatMessage{}, s.fullContext...)
	windowCopy := append([]types.ChatMessage{}, s.contextWindow...)
	memoCopy := s.memo
	s.mu.RUnlock()
	newFull, newWindow, newMemo, err := handler(fullCopy, windowCopy, memoCopy, s.sessionSetting)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if newFull != nil {
		s.fullContext = normalizeChatMessages(newFull)
	}
	if newWindow != nil {
		s.contextWindow = normalizeChatMessages(newWindow)
	}
	if newMemo != nil {
		s.memo = newMemo
	}
	s.mu.Unlock()
	return nil
}

func (s *Session) Resize() error {
	strategy, err := s.AnalyzeContext()
	if err != nil {
		return err
	}
	if strings.TrimSpace(strategy) != "" {
		return s.ExecuteStrategy(strategy)
	}
	return nil
}

func (s *Session) FullContext() []types.ChatMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]types.ChatMessage, len(s.fullContext))
	copy(out, s.fullContext)
	return out
}

func (s *Session) ContextWindow() []types.ChatMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]types.ChatMessage, len(s.contextWindow))
	copy(out, s.contextWindow)
	return out
}

func (s *Session) Memo() any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.memo
}

func (s *Session) ToSerializableData() map[string]any {
	return map[string]any{
		"id":               s.id,
		"auto_resize":      s.autoResize,
		"full_context":     s.FullContext(),
		"context_window":   s.ContextWindow(),
		"memo":             s.Memo(),
		"session_settings": s.sessionSetting.Data(true),
	}
}

func (s *Session) ToJSON() (string, error) {
	b, err := json.MarshalIndent(s.ToSerializableData(), "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Session) ToYAML() (string, error) {
	b, err := yaml.Marshal(s.ToSerializableData())
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Session) LoadSerializableData(data map[string]any) error {
	if data == nil {
		return fmt.Errorf("session data is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if id := strings.TrimSpace(fmt.Sprint(data["id"])); id != "" && id != "<nil>" {
		s.id = id
	}

	if autoResize, ok := data["auto_resize"].(bool); ok {
		s.autoResize = autoResize
	}

	if fullRaw, ok := data["full_context"].([]any); ok {
		s.fullContext = normalizeChatMessages(parseChatMessages(fullRaw))
	} else if fullRawTyped, ok := data["full_context"].([]types.ChatMessage); ok {
		s.fullContext = normalizeChatMessages(fullRawTyped)
	}

	if windowRaw, ok := data["context_window"].([]any); ok {
		s.contextWindow = normalizeChatMessages(parseChatMessages(windowRaw))
	} else if windowRawTyped, ok := data["context_window"].([]types.ChatMessage); ok {
		s.contextWindow = normalizeChatMessages(windowRawTyped)
	}

	s.memo = data["memo"]

	if sessionSettings, ok := data["session_settings"].(map[string]any); ok {
		s.sessionSetting.Update(sessionSettings)
	}
	return nil
}

func (s *Session) LoadJSON(content string) error {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return err
	}
	return s.LoadSerializableData(payload)
}

func (s *Session) LoadYAML(content string) error {
	payload := map[string]any{}
	if err := yaml.Unmarshal([]byte(content), &payload); err != nil {
		return err
	}
	return s.LoadSerializableData(payload)
}

func normalizeChatMessages(in []types.ChatMessage) []types.ChatMessage {
	out := make([]types.ChatMessage, 0, len(in))
	for _, msg := range in {
		if msg.Role == "" {
			msg.Role = "user"
		}
		out = append(out, msg)
	}
	return out
}

func parseChatMessages(raw []any) []types.ChatMessage {
	out := make([]types.ChatMessage, 0, len(raw))
	for _, item := range raw {
		switch typed := item.(type) {
		case types.ChatMessage:
			out = append(out, typed)
		case map[string]any:
			out = append(out, types.ChatMessage{
				Role:    fmt.Sprint(typed["role"]),
				Content: typed["content"],
			})
		}
	}
	return out
}

func randomID() string {
	return fmt.Sprintf("%d", nowUnixNano())
}

var nowUnixNano = func() int64 {
	return time.Now().UnixNano()
}
