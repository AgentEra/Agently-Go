package agentextensions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type ToolExtension struct {
	agent *core.BaseAgent
	tool  *core.Tool
}

func NewToolExtension(agent *core.BaseAgent) *ToolExtension {
	tool, _ := core.NewTool(agent.PluginManager(), agent.Settings())
	ext := &ToolExtension{agent: agent, tool: tool}
	agent.ExtensionHandlers().AppendRequestPrefix(ext.requestPrefix)
	return ext
}

func (e *ToolExtension) Tool() *core.Tool { return e.tool }

func (e *ToolExtension) RegisterTool(info types.ToolInfo, fn any) error {
	if e.tool == nil || e.tool.Manager() == nil {
		return errors.New("tool manager not configured")
	}
	if err := e.tool.Manager().Register(info, fn); err != nil {
		return err
	}
	return e.tool.Manager().Tag([]string{info.Name}, []string{"agent-" + e.agent.Name()})
}

func (e *ToolExtension) UseTools(toolNames []string) error {
	if e.tool == nil || e.tool.Manager() == nil {
		return errors.New("tool manager not configured")
	}
	return e.tool.Manager().Tag(toolNames, []string{"agent-" + e.agent.Name()})
}

func (e *ToolExtension) requestPrefix(ctx context.Context, prompt *core.Prompt, _ *utils.Settings) error {
	if e.tool == nil || e.tool.Manager() == nil {
		return nil
	}
	toolList := e.tool.Manager().GetToolList([]string{"agent-" + e.agent.Name()})
	if len(toolList) == 0 {
		return nil
	}
	entries := make([]any, 0, len(toolList))
	for _, info := range toolList {
		entries = append(entries, map[string]any{
			"name":    info.Name,
			"desc":    info.Desc,
			"kwargs":  info.Kwargs,
			"returns": info.Returns,
		})
	}
	prompt.Set("tools", entries)
	if err := e.tryRunToolJudgementAndAppendResult(ctx, prompt); err != nil {
		return err
	}
	return nil
}

func (e *ToolExtension) tryRunToolJudgementAndAppendResult(ctx context.Context, prompt *core.Prompt) error {
	input := prompt.Get("input", nil, true)
	if input == nil {
		return nil
	}

	judgeReq := core.NewModelRequest(e.agent.PluginManager(), e.agent.Name()+"-tool-judge", e.agent.Settings(), nil, nil)
	judgeReq.SetPrompt("input", input)
	if extraInstruction := prompt.Get("instruct", nil, true); extraInstruction != nil {
		judgeReq.SetPrompt("extra instruction", extraInstruction)
	}
	judgeReq.SetPrompt("tools", prompt.Get("tools", []any{}, true))
	judgeReq.SetPrompt("instruct", "Judge if you need to use tool in {tools} to collect information for responding {input}?")
	judgeReq.SetPrompt("output", map[string]any{
		"use_tool": map[string]any{
			"type": "bool",
		},
		"tool_command": map[string]any{
			"purpose":     map[string]any{"type": "string"},
			"tool_name":   map[string]any{"type": "string"},
			"tool_kwargs": map[string]any{"type": "object"},
		},
	})

	judgeData, err := judgeReq.GetData(ctx, core.GetDataOptions{
		Type:       "parsed",
		MaxRetries: 1,
	})
	if err != nil {
		return nil
	}
	judgeResult, ok := judgeData.(map[string]any)
	if !ok {
		return nil
	}
	useTool, _ := judgeResult["use_tool"].(bool)
	if !useTool {
		return nil
	}

	command, _ := judgeResult["tool_command"].(map[string]any)
	toolName := strings.TrimSpace(fmt.Sprint(command["tool_name"]))
	if toolName == "" || toolName == "<nil>" {
		return nil
	}
	purpose := strings.TrimSpace(fmt.Sprint(command["purpose"]))
	if purpose == "" || purpose == "<nil>" {
		purpose = toolName
	}
	kwargs := normalizeKwargs(command["tool_kwargs"])
	toolResult, callErr := e.tool.Manager().CallTool(ctx, toolName, kwargs)
	if callErr != nil {
		toolResult = map[string]any{"error": callErr.Error()}
	}
	toolLog := map[string]any{
		"tool_name": toolName,
		"kwargs":    kwargs,
		"purpose":   purpose,
		"result":    toolResult,
	}
	if core.IsToolLogsEnabled(e.agent.Settings()) {
		_ = core.EmitSystemMessage(e.agent.Settings(), types.SystemEventTool, toolLog)
	}

	prompt.Set("action_results", map[string]any{purpose: toolResult})
	prompt.Set(
		"extra_instruction",
		"NOTICE: MUST QUOTE KEY INFO OR MARK SOURCE (PREFER URL INCLUDED) FROM {action_results} IN REPLY IF YOU USE {action_results} TO IMPROVE REPLY!",
	)
	return nil
}

func normalizeKwargs(raw any) map[string]any {
	switch typed := raw.(type) {
	case map[string]any:
		return typed
	case map[string]string:
		out := map[string]any{}
		for k, v := range typed {
			out[k] = v
		}
		return out
	default:
		return map[string]any{}
	}
}
