package agentextensions

import (
	"context"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type Agent struct {
	*core.BaseAgent

	sessionExt         *SessionExtension
	toolExt            *ToolExtension
	configurePromptExt *ConfigurePromptExtension
	keyWaiterExt       *KeyWaiterExtension
	autoFuncExt        *AutoFuncExtension
	streamingExt       *StreamingPrintExtension
}

func NewAgent(pluginManager *core.PluginManager, parentSettings *utils.Settings, name string) *Agent {
	base := core.NewBaseAgent(pluginManager, parentSettings, name)
	a := &Agent{BaseAgent: base}
	a.sessionExt = NewSessionExtension(base)
	a.toolExt = NewToolExtension(base)
	a.configurePromptExt = NewConfigurePromptExtension(base)
	a.keyWaiterExt = NewKeyWaiterExtension(base)
	a.autoFuncExt = NewAutoFuncExtension(base)
	a.streamingExt = NewStreamingPrintExtension(base)
	return a
}

func (a *Agent) ActivateSession(sessionID string) *Agent {
	a.sessionExt.ActivateSession(sessionID)
	return a
}

func (a *Agent) DeactivateSession() *Agent {
	a.sessionExt.DeactivateSession()
	return a
}

func (a *Agent) ResetChatHistory() *Agent {
	a.sessionExt.ResetChatHistory()
	return a
}

func (a *Agent) SetChatHistory(chatHistory []types.ChatMessage) *Agent {
	a.sessionExt.SetChatHistory(chatHistory)
	return a
}

func (a *Agent) AddChatHistory(chatHistory []types.ChatMessage) *Agent {
	a.sessionExt.AddChatHistory(chatHistory)
	return a
}

func (a *Agent) CleanContextWindow() *Agent {
	a.sessionExt.CleanContextWindow()
	return a
}

func (a *Agent) RegisterTool(info types.ToolInfo, fn any) error {
	return a.toolExt.RegisterTool(info, fn)
}

func (a *Agent) UseTools(toolNames []string) error {
	return a.toolExt.UseTools(toolNames)
}

func (a *Agent) Tool() *core.Tool {
	return a.toolExt.Tool()
}

func (a *Agent) GetJSONPrompt() (string, error) {
	return a.configurePromptExt.GetJSONPrompt()
}

func (a *Agent) GetYAMLPrompt() (string, error) {
	return a.configurePromptExt.GetYAMLPrompt()
}

func (a *Agent) LoadYAMLPrompt(pathOrContent string, options ...any) error {
	return a.configurePromptExt.LoadYAMLPrompt(pathOrContent, options...)
}

func (a *Agent) LoadJSONPrompt(pathOrContent string, options ...any) error {
	return a.configurePromptExt.LoadJSONPrompt(pathOrContent, options...)
}

func (a *Agent) GetKeyResultWithContext(ctx context.Context, key string, options ...any) (any, error) {
	return a.keyWaiterExt.GetKeyResultWithContext(ctx, key, options...)
}

func (a *Agent) GetKeyResult(args ...any) (any, error) {
	return a.keyWaiterExt.GetKeyResult(args...)
}

func (a *Agent) WaitKeysWithContext(ctx context.Context, keys []string, options ...any) (<-chan [2]any, error) {
	return a.keyWaiterExt.WaitKeysWithContext(ctx, keys, options...)
}

func (a *Agent) WaitKeys(args ...any) (<-chan [2]any, error) {
	return a.keyWaiterExt.WaitKeys(args...)
}

func (a *Agent) OnKey(key string, handler func(any) any) *Agent {
	a.keyWaiterExt.OnKey(key, handler)
	return a
}

func (a *Agent) StartWaiterWithContext(ctx context.Context, options ...any) ([][3]any, error) {
	return a.keyWaiterExt.StartWaiterWithContext(ctx, options...)
}

func (a *Agent) StartWaiter(options ...any) ([][3]any, error) {
	return a.keyWaiterExt.StartWaiter(options...)
}

func (a *Agent) AutoFunc(instruction string, outputSchema any) func(context.Context, map[string]any) (any, error) {
	return a.autoFuncExt.AutoFunc(instruction, outputSchema)
}

func (a *Agent) StreamingPrintWithContext(ctx context.Context) error {
	return a.streamingExt.StreamingPrintWithContext(ctx)
}

func (a *Agent) StreamingPrint(options ...any) error {
	return a.streamingExt.StreamingPrint(options...)
}
