package testkit

import (
	"context"
	"sync/atomic"

	pg "github.com/AgentEra/Agently-Go/agently/builtins/plugins/prompt_generator"
	rp "github.com/AgentEra/Agently-Go/agently/builtins/plugins/response_parser"
	tm "github.com/AgentEra/Agently-Go/agently/builtins/plugins/tool_manager"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type ScriptedRequester struct {
	Script       func(call int) []types.ResponseMessage
	RequestCount *atomic.Int32
}

func (s *ScriptedRequester) GenerateRequestData() (types.RequestData, error) {
	return types.RequestData{}, nil
}

func (s *ScriptedRequester) RequestModel(_ context.Context, _ types.RequestData) (<-chan types.ResponseMessage, error) {
	out := make(chan types.ResponseMessage)
	close(out)
	return out, nil
}

func (s *ScriptedRequester) BroadcastResponse(_ context.Context, _ <-chan types.ResponseMessage) (<-chan types.ResponseMessage, error) {
	out := make(chan types.ResponseMessage, 64)
	go func() {
		defer close(out)
		call := int(s.RequestCount.Add(1))
		for _, msg := range s.Script(call) {
			out <- msg
		}
	}()
	return out, nil
}

func NewRegressionPluginManager(script func(call int) []types.ResponseMessage, requestCount *atomic.Int32) *core.PluginManager {
	if requestCount == nil {
		requestCount = &atomic.Int32{}
	}
	settings := core.NewDefaultSettings(nil)
	manager := core.NewPluginManager(settings, nil, "regression-plugin-manager")

	_ = manager.Register(core.PluginTypePromptGenerator, core.PluginSpec{
		Name:            pg.PluginName,
		DefaultSettings: pg.DefaultSettings,
		Creator:         core.PromptGeneratorCreator(pg.New),
	}, true)
	_ = manager.Register(core.PluginTypeResponseParser, core.PluginSpec{
		Name:            rp.PluginName,
		DefaultSettings: rp.DefaultSettings,
		Creator:         core.ResponseParserCreator(rp.New),
	}, true)
	_ = manager.Register(core.PluginTypeToolManager, core.PluginSpec{
		Name:            tm.PluginName,
		DefaultSettings: tm.DefaultSettings,
		Creator:         core.ToolManagerCreator(tm.New),
	}, true)
	_ = manager.Register(core.PluginTypeModelRequester, core.PluginSpec{
		Name: "RegressionScriptedRequester",
		Creator: core.ModelRequesterCreator(func(_ *core.Prompt, _ *utils.Settings) core.ModelRequester {
			return &ScriptedRequester{Script: script, RequestCount: requestCount}
		}),
	}, true)

	return manager
}
