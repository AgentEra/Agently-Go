package core

import (
	"fmt"

	"github.com/AgentEra/Agently-Go/agently/utils"
)

type Tool struct {
	settings   *utils.Settings
	plugin     ToolManager
	pluginName string
}

func NewTool(pluginManager *PluginManager, parentSettings *utils.Settings) (*Tool, error) {
	if parentSettings == nil {
		parentSettings = NewDefaultSettings(nil)
	}
	settings := utils.NewSettings("Tool-Settings", map[string]any{}, parentSettings)
	spec, err := pluginManager.GetActivatedPlugin(PluginTypeToolManager)
	if err != nil {
		return nil, err
	}
	creator, ok := spec.Creator.(ToolManagerCreator)
	if !ok {
		return nil, fmt.Errorf("tool manager creator for %s has invalid type %T", spec.Name, spec.Creator)
	}
	return &Tool{settings: settings, plugin: creator(settings), pluginName: spec.Name}, nil
}

func (t *Tool) Settings() *utils.Settings { return t.settings }

func (t *Tool) ManagerName() string { return t.pluginName }

func (t *Tool) Manager() ToolManager { return t.plugin }
