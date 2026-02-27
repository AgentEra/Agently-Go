package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type Prompt struct {
	*utils.RuntimeData
	settings        *utils.Settings
	pluginManager   *PluginManager
	promptGenerator PromptGenerator
	parentPrompt    *Prompt
	orderMu         sync.RWMutex
	keyOrder        []string
}

func NewPrompt(pluginManager *PluginManager, parentSettings *utils.Settings, promptDict map[string]any, parentPrompt *Prompt, name string) *Prompt {
	if name == "" {
		name = "Prompt"
	}
	if parentSettings == nil {
		parentSettings = NewDefaultSettings(nil)
	}
	p := &Prompt{
		RuntimeData:   utils.NewRuntimeData(name, promptDict, nil),
		settings:      utils.NewSettings(name+"-Settings", map[string]any{}, parentSettings),
		pluginManager: pluginManager,
		parentPrompt:  parentPrompt,
	}
	if parentPrompt != nil {
		p.RuntimeData.SetParent(parentPrompt.RuntimeData)
	}
	for key := range promptDict {
		p.trackTopLevelKey(key)
	}
	if pluginManager != nil {
		if spec, err := pluginManager.GetActivatedPlugin(PluginTypePromptGenerator); err == nil {
			if creator, ok := spec.Creator.(PromptGeneratorCreator); ok {
				p.promptGenerator = creator(p, p.settings)
			}
		}
	}
	return p
}

func (p *Prompt) Settings() *utils.Settings { return p.settings }

func (p *Prompt) Set(key string, value any, options ...any) {
	config := resolvePromptSetOptions(options...)
	mappings := config.Mappings
	if s, ok := value.(string); ok {
		value = strings.TrimSpace(s)
	}
	if mappings != nil {
		keyAny := utils.DataFormatterSubstitutePlaceholder(key, mappings, nil)
		value = utils.DataFormatterSubstitutePlaceholder(value, mappings, nil)
		mappedKey := fmt.Sprint(keyAny)
		p.trackTopLevelKey(mappedKey)
		p.RuntimeData.Set(mappedKey, value)
		return
	}
	p.trackTopLevelKey(key)
	p.RuntimeData.Set(key, value)
}

func (p *Prompt) Update(data map[string]any, options ...any) {
	config := resolvePromptSetOptions(options...)
	mappings := config.Mappings
	if mappings != nil {
		mapped := utils.DataFormatterSubstitutePlaceholder(data, mappings, nil)
		if m, ok := mapped.(map[string]any); ok {
			for key := range m {
				p.trackTopLevelKey(key)
			}
			p.RuntimeData.Update(m)
			return
		}
	}
	for key := range data {
		p.trackTopLevelKey(key)
	}
	p.RuntimeData.Update(data)
}

func (p *Prompt) Append(key string, value any, options ...any) {
	config := resolvePromptSetOptions(options...)
	mappings := config.Mappings
	if s, ok := value.(string); ok {
		value = strings.TrimSpace(s)
	}
	if mappings != nil {
		keyAny := utils.DataFormatterSubstitutePlaceholder(key, mappings, nil)
		value = utils.DataFormatterSubstitutePlaceholder(value, mappings, nil)
		mappedKey := fmt.Sprint(keyAny)
		p.trackTopLevelKey(mappedKey)
		p.RuntimeData.Append(mappedKey, value)
		return
	}
	p.trackTopLevelKey(key)
	p.RuntimeData.Append(key, value)
}

func (p *Prompt) ToPromptObject() (types.PromptObject, error) {
	if p.promptGenerator == nil {
		return types.PromptObject{}, errors.New("prompt generator plugin is not available")
	}
	return p.promptGenerator.ToPromptObject(), nil
}

func (p *Prompt) ToText(options ...any) (string, error) {
	config := ParsePromptTextOptions(options...)
	if p.promptGenerator == nil {
		data := p.Get("", map[string]any{}, true)
		b, _ := yaml.Marshal(data)
		return string(b), nil
	}
	return p.promptGenerator.ToText(config.RoleMapping)
}

func (p *Prompt) ToMessages(options ...any) ([]map[string]any, error) {
	config := ParsePromptMessageOptions(options...)
	if p.promptGenerator == nil {
		text, _ := p.ToText()
		return []map[string]any{{"role": "user", "content": text}}, nil
	}
	return p.promptGenerator.ToMessages(config)
}

func (p *Prompt) ToOutputModelSchema() (any, error) {
	if p.promptGenerator == nil {
		return p.Get("output", nil, true), nil
	}
	return p.promptGenerator.ToOutputModelSchema()
}

func (p *Prompt) ToSerializablePromptData(options ...any) (map[string]any, error) {
	config := ParseInheritOptions(options...)
	if p.promptGenerator == nil {
		data, _ := p.Get("", map[string]any{}, config.Inherit).(map[string]any)
		return data, nil
	}
	return p.promptGenerator.ToSerializablePromptData(config.Inherit)
}

func (p *Prompt) ToJSONPrompt(options ...any) (string, error) {
	config := ParseInheritOptions(options...)
	if p.promptGenerator != nil {
		return p.promptGenerator.ToJSONPrompt(config.Inherit)
	}
	data, _ := p.ToSerializablePromptData(config.Inherit)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *Prompt) ToYAMLPrompt(options ...any) (string, error) {
	config := ParseInheritOptions(options...)
	if p.promptGenerator != nil {
		return p.promptGenerator.ToYAMLPrompt(config.Inherit)
	}
	data, _ := p.ToSerializablePromptData(config.Inherit)
	b, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *Prompt) OrderedTopLevelKeys(options ...any) []string {
	config := ParseInheritOptions(options...)
	p.orderMu.RLock()
	local := append([]string(nil), p.keyOrder...)
	p.orderMu.RUnlock()
	if !config.Inherit || p.parentPrompt == nil {
		return local
	}
	parentKeys := p.parentPrompt.OrderedTopLevelKeys(true)
	seen := map[string]bool{}
	out := make([]string, 0, len(parentKeys)+len(local))
	for _, key := range parentKeys {
		if !seen[key] {
			out = append(out, key)
			seen[key] = true
		}
	}
	for _, key := range local {
		if !seen[key] {
			out = append(out, key)
			seen[key] = true
		}
	}
	return out
}

func (p *Prompt) trackTopLevelKey(path string) {
	key := topLevelPromptKey(path)
	if key == "" {
		return
	}
	p.orderMu.Lock()
	defer p.orderMu.Unlock()
	for _, existing := range p.keyOrder {
		if existing == key {
			return
		}
	}
	p.keyOrder = append(p.keyOrder, key)
}

func topLevelPromptKey(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "."); idx >= 0 {
		return path[:idx]
	}
	return path
}
