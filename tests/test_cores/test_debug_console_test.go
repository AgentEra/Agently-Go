package core_test

import (
	"testing"
	"time"

	entry "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func TestDebugSettingMapping(t *testing.T) {
	main := entry.NewAgently()

	main.SetSettings("debug", true)
	if got := main.Settings.Get("runtime.show_model_logs", false, true); got != true {
		t.Fatalf("debug=true should enable runtime.show_model_logs, got %#v", got)
	}
	if got := main.Settings.Get("runtime.show_tool_logs", false, true); got != true {
		t.Fatalf("debug=true should enable runtime.show_tool_logs, got %#v", got)
	}
	if got := main.Settings.Get("runtime.show_trigger_flow_logs", false, true); got != true {
		t.Fatalf("debug=true should enable runtime.show_trigger_flow_logs, got %#v", got)
	}

	main.SetSettings("debug", false)
	if got := main.Settings.Get("runtime.show_model_logs", true, true); got != false {
		t.Fatalf("debug=false should disable runtime.show_model_logs, got %#v", got)
	}
	if got := main.Settings.Get("runtime.show_tool_logs", true, true); got != false {
		t.Fatalf("debug=false should disable runtime.show_tool_logs, got %#v", got)
	}
	if got := main.Settings.Get("runtime.show_trigger_flow_logs", true, true); got != false {
		t.Fatalf("debug=false should disable runtime.show_trigger_flow_logs, got %#v", got)
	}
}

func TestEmitSystemMessageWithBoundEventCenter(t *testing.T) {
	main := entry.NewAgently()
	ch := make(chan types.EventMessage, 1)
	main.EventCenter.RegisterHook(types.EventNameSystem, func(msg types.EventMessage) {
		ch <- msg
	}, "debug-system-capture")

	if err := core.EmitSystemMessage(main.Settings, types.SystemEventTool, map[string]any{"name": "sum"}); err != nil {
		t.Fatalf("EmitSystemMessage failed: %v", err)
	}

	select {
	case msg := <-ch:
		content, ok := msg.Content.(map[string]any)
		if !ok {
			t.Fatalf("unexpected system message content type: %T", msg.Content)
		}
		if content["type"] != types.SystemEventTool {
			t.Fatalf("unexpected system message type: %#v", content["type"])
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("wait system message timeout")
	}
}
