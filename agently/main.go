package agently

import (
	"fmt"
	"log"
	"strings"

	"github.com/AgentEra/Agently-Go/agently/builtins/agent_extensions"
	"github.com/AgentEra/Agently-Go/agently/builtins/hookers"
	mr "github.com/AgentEra/Agently-Go/agently/builtins/plugins/model_requester"
	pg "github.com/AgentEra/Agently-Go/agently/builtins/plugins/prompt_generator"
	rp "github.com/AgentEra/Agently-Go/agently/builtins/plugins/response_parser"
	tm "github.com/AgentEra/Agently-Go/agently/builtins/plugins/tool_manager"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/triggerflow"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type Main struct {
	Settings      *utils.Settings
	PluginManager *core.PluginManager
	EventCenter   *core.EventCenter
	Logger        *utils.AgentlyLogger
	Tool          *core.Tool
}

func NewAgently() *Main {
	settings := core.NewDefaultSettings(nil)
	pluginManager := core.NewPluginManager(settings, nil, "global_plugin_manager")

	must(pluginManager.Register(core.PluginTypePromptGenerator, core.PluginSpec{
		Name:            pg.PluginName,
		DefaultSettings: pg.DefaultSettings,
		Creator:         core.PromptGeneratorCreator(pg.New),
	}, true))
	must(pluginManager.Register(core.PluginTypeModelRequester, core.PluginSpec{
		Name:            mr.PluginName,
		DefaultSettings: mr.DefaultSettings,
		Creator:         core.ModelRequesterCreator(mr.New),
	}, true))
	must(pluginManager.Register(core.PluginTypeResponseParser, core.PluginSpec{
		Name:            rp.PluginName,
		DefaultSettings: rp.DefaultSettings,
		Creator:         core.ResponseParserCreator(rp.New),
	}, true))
	must(pluginManager.Register(core.PluginTypeToolManager, core.PluginSpec{
		Name:            tm.PluginName,
		DefaultSettings: tm.DefaultSettings,
		Creator:         core.ToolManagerCreator(tm.New),
	}, true))

	eventCenter := core.NewEventCenter()
	logger := utils.NewLogger("Agently", utils.LevelInfo)
	eventCenter.RegisterHookerPlugin(hookers.NewPureLoggerHooker(logger))
	eventCenter.RegisterHookerPlugin(hookers.NewSystemMessageHooker(logger))
	core.BindEventCenter(settings, eventCenter)

	tool, _ := core.NewTool(pluginManager, settings)
	return &Main{
		Settings:      settings,
		PluginManager: pluginManager,
		EventCenter:   eventCenter,
		Logger:        logger,
		Tool:          tool,
	}
}

// NewMain is kept as a backward-compatible alias for NewAgently.
func NewMain() *Main {
	return NewAgently()
}

func (m *Main) SetSettings(key string, value any, options ...any) *Main {
	config := core.ParseSettingsSetOptions(options...)
	if key == "debug" {
		enabled := false
		switch typed := value.(type) {
		case bool:
			enabled = typed
		default:
			enabled = strings.EqualFold(strings.TrimSpace(fmt.Sprint(value)), "true")
		}
		core.ApplyDebugMode(m.Settings, enabled)
		return m
	}
	m.Settings.SetSettings(key, value, config.AutoLoadEnv)
	return m
}

func (m *Main) SetLogLevel(level utils.LogLevel) *Main {
	m.Logger.SetLevel(level)
	return m
}

func (m *Main) CreatePrompt(name string) *core.Prompt {
	return core.NewPrompt(m.PluginManager, m.Settings, map[string]any{}, nil, name)
}

func (m *Main) CreateRequest(name string) *core.ModelRequest {
	return core.NewModelRequest(m.PluginManager, name, m.Settings, nil, nil)
}

func (m *Main) CreateAgent(name string) *agentextensions.Agent {
	return agentextensions.NewAgent(m.PluginManager, m.Settings, name)
}

func (m *Main) CreateTriggerFlow(name string) *triggerflow.TriggerFlow {
	return triggerflow.New(nil, name)
}

func (m *Main) CreateTriggerFlowBluePrint(name string) *triggerflow.BluePrint {
	return triggerflow.NewBluePrint(name)
}

func must(err error) {
	if err != nil {
		log.Fatalf("agently init failed: %v", err)
	}
}

var Agently = NewAgently()
