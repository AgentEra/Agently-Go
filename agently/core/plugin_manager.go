package core

import (
	"fmt"
	"sort"

	"github.com/AgentEra/Agently-Go/agently/utils"
)

type PluginSpec struct {
	Name            string
	DefaultSettings map[string]any
	Creator         any
	OnRegister      func()
	OnUnregister    func()
}

// PluginManager manages plugin registry and activation.
type PluginManager struct {
	name     string
	settings *utils.Settings
	plugins  *utils.RuntimeData
}

func NewPluginManager(settings *utils.Settings, parent *PluginManager, name string) *PluginManager {
	if settings == nil {
		settings = NewDefaultSettings(nil)
	}
	if name == "" {
		name = "PluginManager"
	}
	var parentPlugins *utils.RuntimeData
	if parent != nil {
		parentPlugins = parent.plugins
	}
	return &PluginManager{
		name:     name,
		settings: settings,
		plugins:  utils.NewRuntimeData(name+"-plugins", map[string]any{}, parentPlugins),
	}
}

func (m *PluginManager) Name() string { return m.name }

func (m *PluginManager) Settings() *utils.Settings { return m.settings }

func (m *PluginManager) Register(pluginType PluginType, spec PluginSpec, activate bool) error {
	if spec.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if spec.OnRegister != nil {
		spec.OnRegister()
	}

	m.plugins.Update(map[string]any{string(pluginType): map[string]any{spec.Name: spec}})
	if activate {
		m.settings.Set(fmt.Sprintf("plugins.%s.activate", pluginType), spec.Name)
	}

	defaults := deepCopyMap(spec.DefaultSettings)
	if global, ok := defaults["$global"].(map[string]any); ok {
		m.settings.Update(global)
		delete(defaults, "$global")
	}
	if mappings, ok := defaults["$mappings"].(map[string]any); ok {
		m.settings.UpdateMappings(mappings)
		delete(defaults, "$mappings")
	}
	m.settings.Set(fmt.Sprintf("plugins.%s.%s", pluginType, spec.Name), defaults)
	return nil
}

func (m *PluginManager) Unregister(pluginType PluginType, pluginName string) error {
	pluginTypeMap, ok := m.plugins.Get(string(pluginType), nil, true).(map[string]any)
	if !ok {
		return fmt.Errorf("plugin type %s not found", pluginType)
	}
	entry, ok := pluginTypeMap[pluginName]
	if !ok {
		return fmt.Errorf("plugin %s not found in %s", pluginName, pluginType)
	}
	if spec, ok := entry.(PluginSpec); ok && spec.OnUnregister != nil {
		spec.OnUnregister()
	}
	m.plugins.Delete(fmt.Sprintf("%s.%s", pluginType, pluginName))
	return nil
}

func (m *PluginManager) GetPlugin(pluginType PluginType, pluginName string) (PluginSpec, error) {
	entry := m.plugins.Get(fmt.Sprintf("%s.%s", pluginType, pluginName), nil, true)
	if entry == nil {
		return PluginSpec{}, fmt.Errorf("plugin %s.%s not found", pluginType, pluginName)
	}
	spec, ok := entry.(PluginSpec)
	if !ok {
		return PluginSpec{}, fmt.Errorf("plugin %s.%s has invalid entry type %T", pluginType, pluginName, entry)
	}
	return spec, nil
}

func (m *PluginManager) GetActivatedPlugin(pluginType PluginType) (PluginSpec, error) {
	active := m.settings.Get(fmt.Sprintf("plugins.%s.activate", pluginType), "", true)
	if active == nil || fmt.Sprint(active) == "" {
		return PluginSpec{}, fmt.Errorf("plugin type %s has no active plugin", pluginType)
	}
	return m.GetPlugin(pluginType, fmt.Sprint(active))
}

func (m *PluginManager) GetPluginList(pluginType *PluginType) map[string][]string {
	result := map[string][]string{}
	if pluginType != nil {
		items, ok := m.plugins.Get(string(*pluginType), nil, true).(map[string]any)
		if !ok {
			return map[string][]string{string(*pluginType): {}}
		}
		names := make([]string, 0, len(items))
		for name := range items {
			names = append(names, name)
		}
		sort.Strings(names)
		return map[string][]string{string(*pluginType): names}
	}
	all, _ := m.plugins.Get("", map[string]any{}, true).(map[string]any)
	for pType, v := range all {
		items, ok := v.(map[string]any)
		if !ok {
			continue
		}
		names := make([]string, 0, len(items))
		for name := range items {
			names = append(names, name)
		}
		sort.Strings(names)
		result[pType] = names
	}
	return result
}
