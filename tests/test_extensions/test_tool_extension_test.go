package extensions_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AgentEra/Agently-Go/agently/builtins/agent_extensions"
	mr "github.com/AgentEra/Agently-Go/agently/builtins/plugins/model_requester"
	pg "github.com/AgentEra/Agently-Go/agently/builtins/plugins/prompt_generator"
	rp "github.com/AgentEra/Agently-Go/agently/builtins/plugins/response_parser"
	tm "github.com/AgentEra/Agently-Go/agently/builtins/plugins/tool_manager"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type toolJudgementRequester struct {
	prompt      *core.Prompt
	requestCall *atomic.Int32
}

func (r *toolJudgementRequester) GenerateRequestData() (types.RequestData, error) {
	return types.RequestData{}, nil
}

func (r *toolJudgementRequester) RequestModel(_ context.Context, _ types.RequestData) (<-chan types.ResponseMessage, error) {
	out := make(chan types.ResponseMessage)
	close(out)
	return out, nil
}

func (r *toolJudgementRequester) BroadcastResponse(_ context.Context, _ <-chan types.ResponseMessage) (<-chan types.ResponseMessage, error) {
	out := make(chan types.ResponseMessage, 16)
	go func() {
		defer close(out)
		call := int(r.requestCall.Add(1))
		switch call {
		case 1:
			done := `{"use_tool":true,"tool_command":{"purpose":"calc","tool_name":"sum","tool_kwargs":{"a":3,"b":4}}}`
			out <- types.ResponseMessage{Event: types.ResponseEventDelta, Data: done}
			out <- types.ResponseMessage{Event: types.ResponseEventDone, Data: done}
			out <- types.ResponseMessage{Event: types.ResponseEventOriginalDone, Data: map[string]any{"done": true}}
		default:
			answer := "tool-not-used"
			if actionResults, ok := r.prompt.Get("action_results", nil, true).(map[string]any); ok {
				if value, exists := actionResults["calc"]; exists && fmt.Sprint(value) == "7" {
					answer = "tool-used"
				}
			}
			done := fmt.Sprintf(`{"answer":"%s"}`, answer)
			out <- types.ResponseMessage{Event: types.ResponseEventDelta, Data: done}
			out <- types.ResponseMessage{Event: types.ResponseEventDone, Data: done}
			out <- types.ResponseMessage{Event: types.ResponseEventOriginalDone, Data: map[string]any{"done": true}}
		}
	}()
	return out, nil
}

func newToolJudgementPluginManager(counter *atomic.Int32) *core.PluginManager {
	settings := core.NewDefaultSettings(nil)
	manager := core.NewPluginManager(settings, nil, "tool-judgement-plugin-manager")

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
		Name:            mr.PluginName,
		DefaultSettings: mr.DefaultSettings,
		Creator: core.ModelRequesterCreator(func(prompt *core.Prompt, _ *utils.Settings) core.ModelRequester {
			return &toolJudgementRequester{prompt: prompt, requestCall: counter}
		}),
	}, true)
	return manager
}

func TestToolExtensionRequestPrefixClosure(t *testing.T) {
	callCounter := &atomic.Int32{}
	manager := newToolJudgementPluginManager(callCounter)
	agent := agentextensions.NewAgent(manager, core.NewDefaultSettings(nil), "tool-closure")

	if err := agent.RegisterTool(types.ToolInfo{
		Name:   "sum",
		Desc:   "sum two ints",
		Kwargs: map[string]any{"a": "number", "b": "number"},
	}, func(kwargs map[string]any) (any, error) {
		return int(toFloat64(kwargs["a"])) + int(toFloat64(kwargs["b"])), nil
	}); err != nil {
		t.Fatalf("register tool failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	agent.Input("3+4=?")
	agent.Output(map[string]any{"answer": "string"})
	data, err := agent.GetData(ctx, core.GetDataOptions{Type: "parsed"})
	if err != nil {
		t.Fatalf("tool closure request failed: %v", err)
	}
	parsed, ok := data.(map[string]any)
	if !ok {
		t.Fatalf("expected parsed map, got %T", data)
	}
	if parsed["answer"] != "tool-used" {
		t.Fatalf("expected tool-used answer, got %#v", parsed)
	}
	if callCounter.Load() < 2 {
		t.Fatalf("expected tool judgement + main request, got %d calls", callCounter.Load())
	}
}

func toFloat64(value any) float64 {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case float64:
		return typed
	case float32:
		return float64(typed)
	default:
		return 0
	}
}
