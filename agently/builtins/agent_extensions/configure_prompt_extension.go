package agentextensions

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type ConfigurePromptExtension struct {
	agent *core.BaseAgent
}

type orderedEntry struct {
	key   string
	value any
}

type orderedMap []orderedEntry

func NewConfigurePromptExtension(agent *core.BaseAgent) *ConfigurePromptExtension {
	return &ConfigurePromptExtension{agent: agent}
}

func (e *ConfigurePromptExtension) GetJSONPrompt() (string, error) {
	agentData, _ := e.agent.AgentPrompt().ToSerializablePromptData(false)
	requestData, _ := e.agent.Prompt().ToSerializablePromptData(false)
	payload := map[string]any{".agent": agentData, ".request": requestData}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (e *ConfigurePromptExtension) GetYAMLPrompt() (string, error) {
	agentData, _ := e.agent.AgentPrompt().ToSerializablePromptData(false)
	requestData, _ := e.agent.Prompt().ToSerializablePromptData(false)
	payload := map[string]any{".agent": agentData, ".request": requestData}
	b, err := yaml.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (e *ConfigurePromptExtension) LoadYAMLPrompt(pathOrContent string, options ...any) error {
	config := parseConfigurePromptLoadOptions(options...)
	mappings := config.Mappings
	promptKeyPath := config.PromptKeyPath
	raw, err := readPromptRaw(pathOrContent)
	if err != nil {
		return err
	}
	prompt, err := parseOrderedDocument(raw)
	if err != nil {
		return err
	}
	if promptKeyPath != "" {
		picked := locatePathInOrdered(prompt, promptKeyPath)
		if picked == nil {
			return fmt.Errorf("cannot locate prompt_key_path: %s", promptKeyPath)
		}
		prompt = picked
	}
	if len(orderedEntries(prompt)) == 0 {
		return fmt.Errorf("prompt config must be a mapping")
	}
	e.executePromptConfigure(prompt, mappings)
	return nil
}

func (e *ConfigurePromptExtension) LoadJSONPrompt(pathOrContent string, options ...any) error {
	config := parseConfigurePromptLoadOptions(options...)
	mappings := config.Mappings
	promptKeyPath := config.PromptKeyPath
	raw, err := readPromptRaw(pathOrContent)
	if err != nil {
		return err
	}
	prompt, err := parseOrderedDocument(raw)
	if err != nil {
		return err
	}
	if promptKeyPath != "" {
		picked := locatePathInOrdered(prompt, promptKeyPath)
		if picked == nil {
			return fmt.Errorf("cannot locate prompt_key_path: %s", promptKeyPath)
		}
		prompt = picked
	}
	if len(orderedEntries(prompt)) == 0 {
		return fmt.Errorf("prompt config must be a mapping")
	}
	e.executePromptConfigure(prompt, mappings)
	return nil
}

func (e *ConfigurePromptExtension) executePromptConfigure(prompt any, variableMappings map[string]any) {
	for _, entry := range orderedEntries(prompt) {
		promptKey := entry.key
		promptValue := entry.value
		switch promptKey {
		case ".agent":
			agentEntries := orderedEntries(promptValue)
			if len(agentEntries) == 0 {
				e.agent.SetAgentPrompt("system", toPlainValue(promptValue), variableMappings)
				continue
			}
			for _, item := range agentEntries {
				e.setAgentPromptByKey(item.key, item.value, variableMappings)
			}
		case ".request":
			requestEntries := orderedEntries(promptValue)
			if len(requestEntries) == 0 {
				e.agent.SetRequestPrompt("input", toPlainValue(promptValue), variableMappings)
				continue
			}
			for _, item := range requestEntries {
				e.setRequestPromptByKey(item.key, item.value, variableMappings)
			}
		case ".alias":
			e.executeAliases(promptValue, variableMappings)
		default:
			if strings.HasPrefix(promptKey, "$") && !strings.HasPrefix(promptKey, "${") {
				e.setAgentPromptByKey(strings.TrimPrefix(promptKey, "$"), promptValue, variableMappings)
			} else {
				e.setRequestPromptByKey(promptKey, promptValue, variableMappings)
			}
		}
	}
}

func (e *ConfigurePromptExtension) setAgentPromptByKey(key string, value any, variableMappings map[string]any) {
	resolvedKey := resolvePromptKey(key, variableMappings)
	if resolvedKey == "output" {
		mappedValue := applyMappings(toPlainValue(value), variableMappings)
		e.agent.SetAgentPrompt(resolvedKey, e.generateOutputValue(mappedValue))
		return
	}
	mappedValue := applyMappings(toPlainValue(value), variableMappings)
	e.agent.SetAgentPrompt(resolvedKey, mappedValue)
}

func (e *ConfigurePromptExtension) setRequestPromptByKey(key string, value any, variableMappings map[string]any) {
	resolvedKey := resolvePromptKey(key, variableMappings)
	if resolvedKey == "output" {
		mappedValue := applyMappings(toPlainValue(value), variableMappings)
		e.agent.SetRequestPrompt(resolvedKey, e.generateOutputValue(mappedValue))
		return
	}
	mappedValue := applyMappings(toPlainValue(value), variableMappings)
	e.agent.SetRequestPrompt(resolvedKey, mappedValue)
}

func (e *ConfigurePromptExtension) generateOutputValue(outputPromptValue any) any {
	if entries := orderedEntries(outputPromptValue); len(entries) > 0 {
		var outputType any
		var outputDesc any
		hasType := false
		hasDesc := false
		for _, entry := range entries {
			switch entry.key {
			case "$type", ".type":
				outputType = entry.value
				hasType = true
			case "$desc", ".desc":
				outputDesc = entry.value
				hasDesc = true
			}
		}
		if hasType || hasDesc {
			if !hasType {
				outputType = "Any"
			}
			return types.NewOutputTuple(e.generateOutputValue(outputType), outputDesc)
		}
		result := map[string]any{}
		for _, entry := range entries {
			result[entry.key] = e.generateOutputValue(entry.value)
		}
		return result
	}

	switch typed := outputPromptValue.(type) {
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, e.generateOutputValue(item))
		}
		return result
	default:
		return toPlainValue(outputPromptValue)
	}
}

func (e *ConfigurePromptExtension) executeAliases(aliases any, variableMappings map[string]any) {
	for _, aliasEntry := range orderedEntries(aliases) {
		aliasName := aliasEntry.key
		spec := aliasEntry.value
		args := []any{}
		kwargs := map[string]any{}

		if spec == nil {
			e.executeAlias(aliasName, args, kwargs, variableMappings)
			continue
		}

		for _, param := range orderedEntries(spec) {
			if param.key == ".args" {
				if list, ok := param.value.([]any); ok {
					for _, item := range list {
						args = append(args, toPlainValue(item))
					}
				}
				continue
			}
			kwargs[param.key] = toPlainValue(param.value)
		}
		e.executeAlias(aliasName, args, kwargs, variableMappings)
	}
}

func (e *ConfigurePromptExtension) executeAlias(name string, args []any, kwargs map[string]any, variableMappings map[string]any) {
	switch name {
	case "set_request_prompt":
		if len(args) < 2 {
			return
		}
		key := fmt.Sprint(args[0])
		value := args[1]
		mappings := variableMappings
		if len(args) > 2 {
			if localMappings, ok := args[2].(map[string]any); ok {
				mappings = localMappings
			}
		}
		if localMappings, ok := kwargs["mappings"].(map[string]any); ok {
			mappings = localMappings
		}
		e.setRequestPromptByKey(key, value, mappings)
	case "set_agent_prompt":
		if len(args) < 2 {
			return
		}
		key := fmt.Sprint(args[0])
		value := args[1]
		mappings := variableMappings
		if len(args) > 2 {
			if localMappings, ok := args[2].(map[string]any); ok {
				mappings = localMappings
			}
		}
		if localMappings, ok := kwargs["mappings"].(map[string]any); ok {
			mappings = localMappings
		}
		e.setAgentPromptByKey(key, value, mappings)
	}
}

func readPromptRaw(pathOrContent string) ([]byte, error) {
	if stat, err := os.Stat(pathOrContent); err == nil && !stat.IsDir() {
		return os.ReadFile(pathOrContent)
	}
	return []byte(pathOrContent), nil
}

func parseOrderedDocument(raw []byte) (any, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	node := &root
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return orderedMap{}, nil
		}
		node = node.Content[0]
	}
	return decodeYAMLNode(node)
}

func decodeYAMLNode(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.MappingNode:
		entries := make(orderedMap, 0, len(node.Content)/2)
		for idx := 0; idx+1 < len(node.Content); idx += 2 {
			key := node.Content[idx].Value
			value, err := decodeYAMLNode(node.Content[idx+1])
			if err != nil {
				return nil, err
			}
			entries = append(entries, orderedEntry{key: key, value: value})
		}
		return entries, nil
	case yaml.SequenceNode:
		list := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			value, err := decodeYAMLNode(child)
			if err != nil {
				return nil, err
			}
			list = append(list, value)
		}
		return list, nil
	case yaml.AliasNode:
		if node.Alias != nil {
			return decodeYAMLNode(node.Alias)
		}
		return nil, nil
	default:
		var value any
		if err := node.Decode(&value); err != nil {
			return node.Value, nil
		}
		return value, nil
	}
}

func locatePathInOrdered(value any, path string) any {
	path = strings.TrimSpace(path)
	if path == "" {
		return value
	}
	current := value
	parts := strings.Split(path, ".")
	for _, part := range parts {
		switch typed := current.(type) {
		case orderedMap:
			found := false
			for _, entry := range typed {
				if entry.key == part {
					current = entry.value
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		case map[string]any:
			next, ok := typed[part]
			if !ok {
				return nil
			}
			current = next
		default:
			return nil
		}
	}
	return current
}

func orderedEntries(value any) orderedMap {
	switch typed := value.(type) {
	case orderedMap:
		return typed
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		entries := make(orderedMap, 0, len(keys))
		for _, key := range keys {
			entries = append(entries, orderedEntry{key: key, value: typed[key]})
		}
		return entries
	default:
		return nil
	}
}

func toPlainValue(value any) any {
	switch typed := value.(type) {
	case orderedMap:
		result := map[string]any{}
		for _, entry := range typed {
			result[entry.key] = toPlainValue(entry.value)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, toPlainValue(item))
		}
		return result
	default:
		return typed
	}
}

func resolvePromptKey(key string, mappings map[string]any) string {
	if mappings == nil {
		return key
	}
	resolved := utils.DataFormatterSubstitutePlaceholder(key, mappings, nil)
	return fmt.Sprint(resolved)
}

func applyMappings(value any, mappings map[string]any) any {
	if mappings == nil {
		return value
	}
	return utils.DataFormatterSubstitutePlaceholder(value, mappings, nil)
}
