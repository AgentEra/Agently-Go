package types

import (
	"fmt"
	"strings"
)

type PromptSlot string

const (
	PromptSystem       PromptSlot = "system"
	PromptDeveloper    PromptSlot = "developer"
	PromptChatHistory  PromptSlot = "chat_history"
	PromptInfo         PromptSlot = "info"
	PromptTools        PromptSlot = "tools"
	PromptActionResult PromptSlot = "action_results"
	PromptInstruct     PromptSlot = "instruct"
	PromptExamples     PromptSlot = "examples"
	PromptInput        PromptSlot = "input"
	PromptAttachment   PromptSlot = "attachment"
	PromptOutput       PromptSlot = "output"
	PromptOutputFormat PromptSlot = "output_format"
	PromptOptions      PromptSlot = "options"
)

type OutputFormat string

const (
	OutputMarkdown OutputFormat = "markdown"
	OutputText     OutputFormat = "text"
	OutputJSON     OutputFormat = "json"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type ToolMeta struct {
	Name    string         `json:"name"`
	Desc    string         `json:"desc"`
	Kwargs  map[string]any `json:"kwargs"`
	Returns any            `json:"returns,omitempty"`
}

// FieldSpec is the Go replacement of python tuple(type, desc, default).
type FieldSpec struct {
	TypeName string `json:"type"`
	Desc     string `json:"desc,omitempty"`
	Default  any    `json:"default,omitempty"`
}

func (f FieldSpec) String() string {
	if f.Desc == "" {
		return f.TypeName
	}
	return fmt.Sprintf("%s (%s)", f.TypeName, f.Desc)
}

func NewFieldSpec(typeName, desc string) FieldSpec {
	return FieldSpec{TypeName: strings.TrimSpace(typeName), Desc: strings.TrimSpace(desc)}
}
