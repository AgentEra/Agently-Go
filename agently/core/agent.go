package core

import (
	"context"
	"fmt"
	"time"

	"github.com/AgentEra/Agently-Go/agently/utils"
)

type BaseAgent struct {
	id   string
	name string

	pluginManager     *PluginManager
	settings          *utils.Settings
	agentPrompt       *Prompt
	extensionHandlers *ExtensionHandlers
	request           *ModelRequest
}

func NewBaseAgent(pluginManager *PluginManager, parentSettings *utils.Settings, name string) *BaseAgent {
	if name == "" {
		name = fmt.Sprintf("agent-%d", time.Now().UnixNano())
	}
	if parentSettings == nil {
		parentSettings = NewDefaultSettings(nil)
	}

	settings := utils.NewSettings("Agent-"+name+"-Settings", map[string]any{}, parentSettings)
	handlers := NewExtensionHandlers(nil)
	agentPrompt := NewPrompt(pluginManager, settings, map[string]any{}, nil, "Agent-"+name+"-Prompt")
	request := NewModelRequest(pluginManager, name, settings, agentPrompt, handlers)

	return &BaseAgent{
		id:                fmt.Sprintf("%d", time.Now().UnixNano()),
		name:              name,
		pluginManager:     pluginManager,
		settings:          settings,
		agentPrompt:       agentPrompt,
		extensionHandlers: handlers,
		request:           request,
	}
}

func (a *BaseAgent) ID() string   { return a.id }
func (a *BaseAgent) Name() string { return a.name }

func (a *BaseAgent) PluginManager() *PluginManager { return a.pluginManager }

func (a *BaseAgent) Settings() *utils.Settings { return a.settings }

func (a *BaseAgent) SetSettings(key string, value any, options ...any) *BaseAgent {
	config := ParseSettingsSetOptions(options...)
	a.settings.SetSettings(key, value, config.AutoLoadEnv)
	return a
}

func (a *BaseAgent) Request() *ModelRequest { return a.request }

func (a *BaseAgent) Prompt() *Prompt { return a.request.Prompt() }

func (a *BaseAgent) AgentPrompt() *Prompt { return a.agentPrompt }

func (a *BaseAgent) ExtensionHandlers() *ExtensionHandlers { return a.extensionHandlers }

func (a *BaseAgent) CreateRequest(name string, options ...any) *ModelRequest {
	config := ParseRequestCreateOptions(options...)
	var parentPrompt *Prompt
	if config.InheritAgentPrompt {
		parentPrompt = a.agentPrompt
	}
	var parentHandlers *ExtensionHandlers
	if config.InheritExtensionHandlers {
		parentHandlers = a.extensionHandlers
	}
	return NewModelRequest(a.pluginManager, name, a.settings, parentPrompt, parentHandlers)
}

func (a *BaseAgent) CreateTempRequest() *ModelRequest {
	return a.CreateRequest(fmt.Sprintf("%s-temp-%d", a.name, time.Now().UnixNano()))
}

func (a *BaseAgent) SetAgentPrompt(key string, value any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	a.agentPrompt.Set(key, value, config.Mappings)
	return a
}

func (a *BaseAgent) SetRequestPrompt(key string, value any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	a.request.Prompt().Set(key, value, config.Mappings)
	return a
}

func (a *BaseAgent) RemoveAgentPrompt(key string) *BaseAgent {
	a.agentPrompt.Set(key, nil)
	return a
}

func (a *BaseAgent) RemoveRequestPrompt(key string) *BaseAgent {
	a.request.Prompt().Set(key, nil)
	return a
}

func (a *BaseAgent) ResetChatHistory() *BaseAgent {
	a.agentPrompt.Delete("chat_history")
	a.agentPrompt.Set("chat_history", []any{})
	return a
}

func (a *BaseAgent) SetChatHistory(chatHistory any) *BaseAgent {
	a.agentPrompt.Set("chat_history", chatHistory)
	return a
}

func (a *BaseAgent) AddChatHistory(chatHistory any) *BaseAgent {
	a.agentPrompt.Append("chat_history", chatHistory)
	return a
}

func (a *BaseAgent) ResetActionResults() *BaseAgent {
	a.agentPrompt.Delete("action_results")
	return a
}

func (a *BaseAgent) SetActionResults(actionResults any) *BaseAgent {
	a.agentPrompt.Set("action_results", actionResults)
	return a
}

func (a *BaseAgent) AddActionResults(action string, result any) *BaseAgent {
	a.agentPrompt.Append("action_results", map[string]any{action: result})
	return a
}

func (a *BaseAgent) System(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("system", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("system", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) Rule(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("instruct", []any{"{system.rule} ARE IMPORTANT RULES YOU SHALL FOLLOW!"})
		a.agentPrompt.Set("system.rule", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("instruct", []any{"{system.rule} ARE IMPORTANT RULES YOU SHALL FOLLOW!"})
		a.request.Prompt().Set("system.rule", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) Role(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("instruct", []any{"YOU MUST REACT AND RESPOND AS {system.role}!"})
		a.agentPrompt.Set("system.your_role", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("instruct", []any{"YOU MUST REACT AND RESPOND AS {system.role}!"})
		a.request.Prompt().Set("system.your_role", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) UserInfo(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("instruct", []any{"{system.user_info} IS IMPORTANT INFORMATION ABOUT USER!"})
		a.agentPrompt.Set("system.user_info", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("instruct", []any{"{system.user_info} IS IMPORTANT INFORMATION ABOUT USER!"})
		a.request.Prompt().Set("system.user_info", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) Input(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("input", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("input", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) Info(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("info", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("info", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) Instruct(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("instruct", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("instruct", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) Examples(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("examples", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("examples", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) Output(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("output", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("output", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) Attachment(prompt any, options ...any) *BaseAgent {
	config := resolvePromptSetOptions(options...)
	if config.Always {
		a.agentPrompt.Set("attachment", prompt, config.Mappings)
	} else {
		a.request.Prompt().Set("attachment", prompt, config.Mappings)
	}
	return a
}

func (a *BaseAgent) Options(options map[string]any, opts ...any) *BaseAgent {
	config := resolvePromptSetOptions(opts...)
	if config.Always {
		a.agentPrompt.Set("options", options)
	} else {
		a.request.Prompt().Set("options", options)
	}
	return a
}

func (a *BaseAgent) GetPromptText(_ ...any) (string, error) {
	return a.request.Prompt().ToText()
}

func (a *BaseAgent) GetResponse() *ModelResponse { return a.request.GetResponse() }

func (a *BaseAgent) GetResult() *ModelResponseResult { return a.request.GetResult() }

func (a *BaseAgent) GetMetaWithContext(ctx context.Context) (map[string]any, error) {
	return a.request.GetMetaWithContext(ctx)
}

func (a *BaseAgent) GetMeta(options ...any) (map[string]any, error) {
	return a.request.GetMeta(options...)
}

func (a *BaseAgent) GetTextWithContext(ctx context.Context) (string, error) {
	return a.request.GetTextWithContext(ctx)
}

func (a *BaseAgent) GetText(options ...any) (string, error) {
	return a.request.GetText(options...)
}

func (a *BaseAgent) GetDataWithContext(ctx context.Context, opts GetDataOptions) (any, error) {
	return a.request.GetDataWithContext(ctx, opts)
}

func (a *BaseAgent) GetData(args ...any) (any, error) {
	return a.request.GetData(args...)
}

func (a *BaseAgent) StartWithContext(ctx context.Context) (any, error) {
	return a.GetDataWithContext(ctx, GetDataOptions{Type: "parsed"})
}

func (a *BaseAgent) Start(options ...any) (any, error) {
	return a.GetData(append([]any{GetDataOptions{Type: "parsed"}}, options...)...)
}

func (a *BaseAgent) GetDataObjectWithContext(ctx context.Context, opts GetDataOptions) (any, error) {
	return a.request.GetDataObjectWithContext(ctx, opts)
}

func (a *BaseAgent) GetDataObject(args ...any) (any, error) {
	return a.request.GetDataObject(args...)
}

func (a *BaseAgent) GetGeneratorWithContext(ctx context.Context, streamType string, options ...any) (<-chan any, error) {
	return a.request.GetGeneratorWithContext(ctx, streamType, options...)
}

func (a *BaseAgent) GetGenerator(args ...any) (<-chan any, error) {
	return a.request.GetGenerator(args...)
}
