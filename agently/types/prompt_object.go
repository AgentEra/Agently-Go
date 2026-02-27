package types

import "fmt"

type AttachmentContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data any    `json:"data,omitempty"`
}

type PromptObject struct {
	System       any
	Developer    any
	ChatHistory  []ChatMessage
	Info         any
	Tools        []ToolMeta
	ActionResult any
	Instruct     any
	Examples     any
	Input        any
	Attachment   []AttachmentContent
	Output       any
	OutputFormat OutputFormat
	Options      map[string]any
	Extra        map[string]any
}

func PromptObjectFromMap(data map[string]any) PromptObject {
	obj := PromptObject{Options: map[string]any{}, Extra: map[string]any{}}
	if data == nil {
		obj.OutputFormat = OutputMarkdown
		return obj
	}
	obj.System = data["system"]
	obj.Developer = data["developer"]
	obj.Info = data["info"]
	obj.ActionResult = data["action_results"]
	obj.Instruct = data["instruct"]
	obj.Examples = data["examples"]
	obj.Input = data["input"]
	obj.Output = data["output"]

	if opts, ok := data["options"].(map[string]any); ok {
		obj.Options = opts
	}

	if tools, ok := data["tools"].([]any); ok {
		obj.Tools = make([]ToolMeta, 0, len(tools))
		for _, t := range tools {
			if m, ok := t.(map[string]any); ok {
				obj.Tools = append(obj.Tools, ToolMeta{
					Name:    fmt.Sprint(m["name"]),
					Desc:    fmt.Sprint(m["desc"]),
					Kwargs:  asMap(m["kwargs"]),
					Returns: m["returns"],
				})
			}
		}
	}

	switch history := data["chat_history"].(type) {
	case []ChatMessage:
		obj.ChatHistory = append(obj.ChatHistory, history...)
	case []map[string]any:
		obj.ChatHistory = make([]ChatMessage, 0, len(history))
		for _, item := range history {
			obj.ChatHistory = append(obj.ChatHistory, ChatMessage{Role: fmt.Sprint(item["role"]), Content: item["content"]})
		}
	case []any:
		obj.ChatHistory = make([]ChatMessage, 0, len(history))
		for _, item := range history {
			switch msg := item.(type) {
			case map[string]any:
				obj.ChatHistory = append(obj.ChatHistory, ChatMessage{Role: fmt.Sprint(msg["role"]), Content: msg["content"]})
			case ChatMessage:
				obj.ChatHistory = append(obj.ChatHistory, msg)
			case map[string]string:
				obj.ChatHistory = append(obj.ChatHistory, ChatMessage{Role: msg["role"], Content: msg["content"]})
			}
		}
	}

	if attachment, ok := data["attachment"].([]any); ok {
		obj.Attachment = make([]AttachmentContent, 0, len(attachment))
		for _, item := range attachment {
			switch a := item.(type) {
			case map[string]any:
				t := fmt.Sprint(a["type"])
				if t == "text" {
					obj.Attachment = append(obj.Attachment, AttachmentContent{
						Type: "text",
						Text: fmt.Sprint(a["text"]),
					})
				} else {
					obj.Attachment = append(obj.Attachment, AttachmentContent{
						Type: t,
						Data: a[t],
					})
				}
			case string:
				obj.Attachment = append(obj.Attachment, AttachmentContent{Type: "text", Text: a})
			}
		}
	}

	if format, ok := data["output_format"].(string); ok && format != "" {
		obj.OutputFormat = OutputFormat(format)
	} else {
		if obj.Output != nil {
			switch obj.Output.(type) {
			case map[string]any, []any:
				obj.OutputFormat = OutputJSON
			default:
				obj.OutputFormat = OutputMarkdown
			}
		} else {
			obj.OutputFormat = OutputMarkdown
		}
	}

	for k, v := range data {
		switch k {
		case "system", "developer", "chat_history", "info", "tools", "action_results", "instruct", "examples", "input", "attachment", "output", "output_format", "options":
		default:
			obj.Extra[k] = v
		}
	}

	return obj
}

func asMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
