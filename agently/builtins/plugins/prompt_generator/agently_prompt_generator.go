package promptgenerator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type AgentlyPromptGenerator struct {
	prompt   *core.Prompt
	settings *utils.Settings
}

const PluginName = "AgentlyPromptGenerator"

var DefaultSettings = map[string]any{
	"$global": map[string]any{
		"prompt": map[string]any{
			"add_current_time": false,
		},
	},
}

func New(prompt *core.Prompt, settings *utils.Settings) core.PromptGenerator {
	return &AgentlyPromptGenerator{prompt: prompt, settings: settings}
}

func (g *AgentlyPromptGenerator) ToPromptObject() types.PromptObject {
	data, _ := g.prompt.Get("", map[string]any{}, true).(map[string]any)
	return types.PromptObjectFromMap(data)
}

func (g *AgentlyPromptGenerator) checkPromptAllEmpty(obj types.PromptObject) error {
	if obj.Input == nil && obj.Info == nil && obj.Instruct == nil && obj.Output == nil && len(obj.Attachment) == 0 && len(obj.Extra) == 0 {
		return fmt.Errorf("Prompt requires at least one of 'input', 'info', 'instruct', 'output', 'attachment' or customize extra prompt keys to be provided.")
	}
	return nil
}

func (g *AgentlyPromptGenerator) ToText(roleMapping map[string]string) (string, error) {
	obj := g.ToPromptObject()
	if err := g.checkPromptAllEmpty(obj); err != nil {
		return "", err
	}

	roles := g.getRoleMapping(roleMapping)
	titles := g.getPromptTitleMapping()
	lines := make([]string, 0)
	lines = append(lines, roles["user"]+":")

	if g.settings.Get("prompt.add_current_time", false, true) == true {
		lines = append(lines, fmt.Sprintf("[current time]: %s", utils.CurrentTime()), "")
	}

	if obj.System != nil {
		lines = append(lines, g.generateYAMLPromptList(titles["system"], obj.System)...)
	}
	if obj.Developer != nil {
		lines = append(lines, g.generateYAMLPromptList(titles["developer"], obj.Developer)...)
	}

	if len(obj.ChatHistory) > 0 {
		lines = append(lines, fmt.Sprintf("[%s]:", titles["chat_history"]))
		for _, msg := range obj.ChatHistory {
			role := msg.Role
			if mapped, ok := roles[role]; ok {
				role = mapped
			} else if mapped, ok := roles["_"]; ok {
				role = mapped
			}
			for _, textLine := range historyContentToTextLines(msg.Content) {
				lines = append(lines, fmt.Sprintf("[%s]:%s", role, textLine))
			}
		}
		lines = append(lines, "")
	}

	lines = append(lines, g.generateMainPrompt(obj)...)
	lines = append(lines, roles["assistant"]+":")
	return strings.Join(lines, "\n"), nil
}

func (g *AgentlyPromptGenerator) ToMessages(options core.PromptMessageOptions) ([]map[string]any, error) {
	obj := g.ToPromptObject()
	if err := g.checkPromptAllEmpty(obj); err != nil {
		return nil, err
	}

	roles := g.getRoleMapping(options.RoleMapping)
	titles := g.getPromptTitleMapping()
	messages := make([]map[string]any, 0)

	addCurrentTime := g.settings.Get("prompt.add_current_time", false, true) == true
	currentTimePrefix := ""
	if addCurrentTime {
		currentTimePrefix = fmt.Sprintf("[current time]: %s\n\n", utils.CurrentTime())
	}
	prependCurrentTimeText := func(text string) string {
		if !addCurrentTime {
			return text
		}
		return currentTimePrefix + text
	}
	prependCurrentTimeRich := func(content []map[string]any) []map[string]any {
		if !addCurrentTime {
			return content
		}
		for idx := range content {
			if fmt.Sprint(content[idx]["type"]) == "text" {
				content[idx]["text"] = prependCurrentTimeText(fmt.Sprint(content[idx]["text"]))
				return content
			}
		}
		return append([]map[string]any{{"type": "text", "text": currentTimePrefix}}, content...)
	}

	if obj.System != nil {
		messages = append(messages, map[string]any{"role": roles["system"], "content": g.serializeContent(obj.System)})
	}
	if obj.Developer != nil {
		messages = append(messages, map[string]any{"role": roles["developer"], "content": g.serializeContent(obj.Developer)})
	}

	history := make([]map[string]any, 0)
	lastRole := ""
	for _, msg := range obj.ChatHistory {
		role := msg.Role
		if mapped, ok := roles[role]; ok {
			role = mapped
		} else if mapped, ok := roles["_"]; ok {
			role = mapped
		}
		content := historyContentToRich(msg.Content)
		if options.StrictRoleOrders {
			if len(history) > 0 && role == lastRole {
				previous := history[len(history)-1]["content"].([]map[string]any)
				history[len(history)-1]["content"] = append(previous, content...)
			} else {
				history = append(history, map[string]any{"role": role, "content": content})
			}
		} else {
			history = append(history, map[string]any{"role": role, "content": content})
		}
		lastRole = role
	}
	if options.StrictRoleOrders && len(history) > 0 {
		if fmt.Sprint(history[0]["role"]) != "user" {
			history = append([]map[string]any{{
				"role":    "user",
				"content": []map[string]any{{"type": "text", "text": fmt.Sprintf("[%s]", titles["chat_history"])}},
			}}, history...)
		}
		if fmt.Sprint(history[len(history)-1]["role"]) != "assistant" {
			history = append(history, map[string]any{
				"role":    "assistant",
				"content": []map[string]any{{"type": "text", "text": "[User continue input]"}},
			})
		}
	}

	if options.RichContent {
		for _, item := range history {
			messages = append(messages, map[string]any{"role": item["role"], "content": item["content"]})
		}
	} else {
		for _, item := range history {
			simplified := simplifyHistoryContent(item["content"])
			messages = append(messages, map[string]any{"role": item["role"], "content": simplified})
		}
	}

	onlyInput := obj.Input != nil && obj.Tools == nil && obj.ActionResult == nil && obj.Info == nil && obj.Instruct == nil && obj.Output == nil && len(obj.Extra) == 0 && len(obj.Attachment) == 0
	if onlyInput {
		messages = append(messages, map[string]any{"role": roles["user"], "content": prependCurrentTimeText(g.serializeContent(obj.Input))})
		return messages, nil
	}

	onlyAttachment := len(obj.Attachment) > 0 && obj.Input == nil && obj.Tools == nil && obj.ActionResult == nil && obj.Info == nil && obj.Instruct == nil && obj.Output == nil && len(obj.Extra) == 0
	if onlyAttachment {
		if options.RichContent {
			content := prependCurrentTimeRich(attachmentsToRichContent(obj.Attachment))
			messages = append(messages, map[string]any{"role": roles["user"], "content": content})
			return messages, nil
		}
		for _, att := range obj.Attachment {
			if att.Type == "text" && att.Text != "" {
				messages = append(messages, map[string]any{"role": roles["user"], "content": prependCurrentTimeText(att.Text)})
			}
		}
		return messages, nil
	}

	mainPrompt := strings.Join(g.generateMainPrompt(obj), "\n")
	if options.RichContent {
		content := []map[string]any{{"type": "text", "text": prependCurrentTimeText(mainPrompt)}}
		if len(obj.Attachment) > 0 {
			content = append(content, attachmentsToRichContent(obj.Attachment)...)
		}
		messages = append(messages, map[string]any{"role": roles["user"], "content": content})
		return messages, nil
	}

	for _, att := range obj.Attachment {
		if att.Type == "text" && att.Text != "" {
			messages = append(messages, map[string]any{"role": roles["user"], "content": att.Text})
		}
	}
	messages = append(messages, map[string]any{"role": roles["user"], "content": prependCurrentTimeText(mainPrompt)})
	return messages, nil
}

func (g *AgentlyPromptGenerator) ToOutputModelSchema() (any, error) {
	obj := g.ToPromptObject()
	if obj.Output == nil {
		return nil, fmt.Errorf("output is empty")
	}
	return utils.DataFormatterSanitize(g.toSerializableOutputPrompt(obj.Output), true), nil
}

func (g *AgentlyPromptGenerator) ToSerializablePromptData(inherit bool) (map[string]any, error) {
	data, _ := g.prompt.Get("", map[string]any{}, inherit).(map[string]any)
	if outputPrompt, ok := data["output"]; ok {
		data["output"] = g.toSerializableOutputPrompt(outputPrompt)
	}
	sanitized, _ := utils.DataFormatterSanitize(data, true).(map[string]any)
	return sanitized, nil
}

func (g *AgentlyPromptGenerator) ToJSONPrompt(inherit bool) (string, error) {
	data, _ := g.ToSerializablePromptData(inherit)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (g *AgentlyPromptGenerator) ToYAMLPrompt(inherit bool) (string, error) {
	data, _ := g.ToSerializablePromptData(inherit)
	b, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (g *AgentlyPromptGenerator) serializeContent(v any) string {
	sanitized := utils.DataFormatterSanitize(v, false)
	switch t := sanitized.(type) {
	case string:
		return t
	case nil:
		return ""
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(t)
	default:
		b, _ := yaml.Marshal(t)
		return string(b)
	}
}

func (g *AgentlyPromptGenerator) generateMainPrompt(obj types.PromptObject) []string {
	titles := g.getPromptTitleMapping()
	lines := make([]string, 0)

	if len(obj.Tools) > 0 {
		lines = append(lines, fmt.Sprintf("[%s]:", titles["tools"]))
		for _, tool := range obj.Tools {
			lines = append(lines, "[")
			lines = append(lines, fmt.Sprintf("name: %s", tool.Name))
			lines = append(lines, fmt.Sprintf("desc: %s", tool.Desc))
			lines = append(lines, fmt.Sprintf("kwargs: %s", g.generateJSONOutputPrompt(tool.Kwargs, 0)))
			if tool.Returns != nil {
				lines = append(lines, fmt.Sprintf("returns: %s", g.generateJSONOutputPrompt(tool.Returns, 0)))
			}
			lines = append(lines, "]")
		}
	}

	if obj.ActionResult != nil {
		lines = append(lines, g.generateYAMLPromptList(titles["action_results"], obj.ActionResult)...)
	}

	if obj.Info != nil {
		lines = append(lines, fmt.Sprintf("[%s]:", titles["info"]))
		switch info := obj.Info.(type) {
		case map[string]any:
			for _, key := range mapKeysSorted(info) {
				lines = append(lines, fmt.Sprintf("- %s: %v", key, utils.DataFormatterSanitize(info[key], false)))
			}
		case []any:
			for _, item := range info {
				lines = append(lines, fmt.Sprintf("- %v", utils.DataFormatterSanitize(item, false)))
			}
		default:
			lines = append(lines, fmt.Sprint(utils.DataFormatterSanitize(info, false)))
		}
		lines = append(lines, "")
	}

	for _, key := range g.orderedExtraKeys(obj.Extra) {
		lines = append(lines, g.generateYAMLPromptList(key, obj.Extra[key])...)
	}

	if obj.Instruct != nil {
		lines = append(lines, g.generateYAMLPromptList(titles["instruct"], obj.Instruct)...)
	}
	if obj.Examples != nil {
		lines = append(lines, g.generateYAMLPromptList(titles["examples"], obj.Examples)...)
	}
	if obj.Input != nil {
		lines = append(lines, g.generateYAMLPromptList(titles["input"], obj.Input)...)
	}

	if obj.Output != nil {
		switch obj.OutputFormat {
		case types.OutputJSON:
			lines = append(lines,
				fmt.Sprintf("[%s]:", titles["output_requirement"]),
				"Data Format: JSON",
				"Data Structure:",
				g.generateJSONOutputPrompt(obj.Output, 0),
				"",
			)
		case types.OutputMarkdown:
			lines = append(lines,
				fmt.Sprintf("[%s]:", titles["output_requirement"]),
				"Data Format: markdown text",
			)
		case types.OutputText:
			// Do not inject output requirement for text mode.
		}
	}

	lines = append(lines, fmt.Sprintf("[%s]:", titles["output"]))
	return lines
}

func (g *AgentlyPromptGenerator) generateYAMLPromptList(title string, promptPart any) []string {
	content := g.serializeContent(promptPart)
	return []string{fmt.Sprintf("[%s]:", title), content, ""}
}

func (g *AgentlyPromptGenerator) generateJSONOutputPrompt(output any, layer int) string {
	indent := strings.Repeat("  ", layer)
	nextIndent := strings.Repeat("  ", layer+1)

	if m, ok := toStringMap(output); ok {
		if len(m) == 0 {
			return "{}"
		}
		keys := mapKeysSorted(m)
		lines := []string{"{"}
		for idx, key := range keys {
			value := m[key]
			valueStr := g.generateJSONOutputPrompt(value, layer+1)
			descStr := ""
			if tuple, ok := value.(types.OutputTuple); ok && len(tuple) > 1 {
				descStr = " //"
				if desc := strings.TrimSpace(fmt.Sprint(tuple[1])); desc != "" && desc != "<nil>" {
					descStr += " " + desc
				}
			}
			comma := ""
			if idx < len(keys)-1 {
				comma = ","
			}
			lines = append(lines, fmt.Sprintf("%s\"%s\": %s%s%s", nextIndent, key, valueStr, comma, descStr))
		}
		lines = append(lines, fmt.Sprintf("%s}", indent))
		return strings.Join(lines, "\n")
	}

	if list, ok := toAnySlice(output); ok {
		if len(list) == 0 {
			return "[]"
		}
		lines := []string{"["}
		for idx := range list {
			item := list[idx]
			itemStr := g.generateJSONOutputPrompt(item, layer+1)
			descStr := ""
			if tuple, ok := item.(types.OutputTuple); ok && len(tuple) > 1 {
				if desc := strings.TrimSpace(fmt.Sprint(tuple[1])); desc != "" && desc != "<nil>" {
					descStr = " // " + desc
				}
			}
			lines = append(lines, fmt.Sprintf("%s%s,%s", nextIndent, itemStr, descStr))
		}
		lines = append(lines, fmt.Sprintf("%s...", nextIndent))
		lines = append(lines, fmt.Sprintf("%s]", indent))
		return strings.Join(lines, "\n")
	}

	if tuple, ok := output.(types.OutputTuple); ok && len(tuple) >= 1 {
		if m, ok := toStringMap(tuple[0]); ok {
			return g.generateJSONOutputPrompt(m, layer+1)
		}
		if list, ok := toAnySlice(tuple[0]); ok {
			return g.generateJSONOutputPrompt(list, layer+1)
		}
		return fmt.Sprintf("<%v>", utils.DataFormatterSanitize(tuple[0], false))
	}

	return fmt.Sprintf("<%v>", utils.DataFormatterSanitize(output, false))
}

func (g *AgentlyPromptGenerator) toSerializableOutputPrompt(outputPromptPart any) any {
	if tuple, ok := outputPromptPart.(types.OutputTuple); ok {
		switch len(tuple) {
		case 0:
			return []any{}
		case 1:
			return map[string]any{"$type": tuple[0]}
		default:
			descParts := make([]string, 0, len(tuple)-1)
			for _, part := range tuple[1:] {
				if part == nil {
					continue
				}
				text := strings.TrimSpace(fmt.Sprint(part))
				if text == "" || text == "<nil>" {
					continue
				}
				descParts = append(descParts, text)
			}
			if len(descParts) == 0 {
				return map[string]any{"$type": tuple[0]}
			}
			return map[string]any{"$type": tuple[0], "$desc": strings.Join(descParts, ";")}
		}
	}

	if m, ok := toStringMap(outputPromptPart); ok {
		result := map[string]any{}
		for key, value := range m {
			result[key] = g.toSerializableOutputPrompt(value)
		}
		return result
	}

	if list, ok := toAnySlice(outputPromptPart); ok {
		copied := make([]any, 0, len(list))
		copied = append(copied, list...)
		return copied
	}

	return outputPromptPart
}

func historyContentToRich(content any) []map[string]any {
	switch typed := content.(type) {
	case map[string]any:
		if _, ok := typed["type"]; ok {
			return []map[string]any{typed}
		}
		return []map[string]any{{"type": "text", "text": fmt.Sprint(content)}}
	case []map[string]any:
		out := make([]map[string]any, 0, len(typed))
		out = append(out, typed...)
		return out
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			switch one := item.(type) {
			case map[string]any:
				if _, ok := one["type"]; ok {
					out = append(out, one)
				}
			default:
				out = append(out, map[string]any{"type": "text", "text": fmt.Sprint(one)})
			}
		}
		return out
	default:
		return []map[string]any{{"type": "text", "text": fmt.Sprint(content)}}
	}
}

func historyContentToTextLines(content any) []string {
	switch typed := content.(type) {
	case map[string]any:
		if _, ok := typed["type"]; ok {
			return historyContentToTextLines([]any{typed})
		}
		return []string{fmt.Sprint(utils.DataFormatterSanitize(typed, false))}
	case []map[string]any:
		lines := make([]string, 0)
		for _, item := range typed {
			if fmt.Sprint(item["type"]) == "text" {
				lines = append(lines, fmt.Sprint(item["text"]))
			}
		}
		return lines
	case []any:
		lines := make([]string, 0)
		for _, item := range typed {
			switch one := item.(type) {
			case map[string]any:
				if fmt.Sprint(one["type"]) == "text" {
					lines = append(lines, fmt.Sprint(one["text"]))
				}
			default:
				lines = append(lines, fmt.Sprint(utils.DataFormatterSanitize(one, false)))
			}
		}
		return lines
	default:
		return []string{fmt.Sprint(utils.DataFormatterSanitize(content, false))}
	}
}

func simplifyHistoryContent(content any) string {
	switch typed := content.(type) {
	case string:
		return typed
	case []map[string]any:
		parts := make([]string, 0)
		for _, item := range typed {
			if fmt.Sprint(item["type"]) == "text" {
				parts = append(parts, fmt.Sprint(item["text"]))
			}
		}
		return strings.Join(parts, "\n\n")
	case []any:
		parts := make([]string, 0)
		for _, item := range typed {
			switch one := item.(type) {
			case map[string]any:
				if fmt.Sprint(one["type"]) == "text" {
					parts = append(parts, fmt.Sprint(one["text"]))
				}
			default:
				parts = append(parts, fmt.Sprint(one))
			}
		}
		return strings.Join(parts, "\n\n")
	default:
		return fmt.Sprint(content)
	}
}

func attachmentsToRichContent(attachments []types.AttachmentContent) []map[string]any {
	content := make([]map[string]any, 0, len(attachments))
	for _, att := range attachments {
		item := map[string]any{"type": att.Type}
		if att.Type == "text" {
			item["text"] = att.Text
		} else if att.Data != nil {
			item[att.Type] = att.Data
		}
		content = append(content, item)
	}
	return content
}

func toStringMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	}
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return nil, false
	}
	result := map[string]any{}
	iter := rv.MapRange()
	for iter.Next() {
		result[fmt.Sprint(iter.Key().Interface())] = iter.Value().Interface()
	}
	return result, true
}

func toAnySlice(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	case types.OutputTuple:
		// OutputTuple has dedicated semantics; do not treat as plain list.
		return nil, false
	}
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil, false
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	result := make([]any, 0, rv.Len())
	for idx := 0; idx < rv.Len(); idx++ {
		result = append(result, rv.Index(idx).Interface())
	}
	return result, true
}

func mapKeysSorted(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (g *AgentlyPromptGenerator) orderedExtraKeys(extra map[string]any) []string {
	if len(extra) == 0 {
		return nil
	}
	ordered := make([]string, 0, len(extra))
	seen := map[string]bool{}
	for _, key := range g.prompt.OrderedTopLevelKeys(true) {
		if isStandardPromptSlot(key) {
			continue
		}
		if _, ok := extra[key]; ok && !seen[key] {
			ordered = append(ordered, key)
			seen[key] = true
		}
	}
	remaining := make([]string, 0)
	for key := range extra {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	ordered = append(ordered, remaining...)
	return ordered
}

func isStandardPromptSlot(key string) bool {
	switch key {
	case "system", "developer", "chat_history", "info", "tools", "action_results", "instruct", "examples", "input", "attachment", "output", "output_format", "options":
		return true
	default:
		return false
	}
}

func (g *AgentlyPromptGenerator) getRoleMapping(overrides map[string]string) map[string]string {
	roleMap := map[string]string{"system": "system", "developer": "developer", "assistant": "assistant", "user": "user", "_": "assistant"}
	if configured, ok := g.settings.Get("prompt.role_mapping", map[string]any{}, true).(map[string]any); ok {
		for k, v := range configured {
			roleMap[k] = fmt.Sprint(v)
		}
	}
	for k, v := range overrides {
		roleMap[k] = v
	}
	return roleMap
}

func (g *AgentlyPromptGenerator) getPromptTitleMapping() map[string]string {
	titles := map[string]string{
		"system":             "SYSTEM",
		"developer":          "DEVELOPER DIRECTIONS",
		"chat_history":       "CHAT HISTORY",
		"info":               "INFO",
		"tools":              "TOOLS",
		"action_results":     "ACTION RESULTS",
		"instruct":           "INSTRUCT",
		"examples":           "EXAMPLES",
		"input":              "INPUT",
		"output":             "OUTPUT",
		"output_requirement": "OUTPUT REQUIREMENT",
	}
	if configured, ok := g.settings.Get("prompt.prompt_title_mapping", map[string]any{}, true).(map[string]any); ok {
		for k, v := range configured {
			titles[k] = fmt.Sprint(v)
		}
	}
	return titles
}
