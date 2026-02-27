package core

import (
	"fmt"
	"strings"

	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

// ApplyDebugMode keeps compatibility with Python's debug switch:
// debug=true enables model/tool/trigger-flow runtime logs, false disables them.
func ApplyDebugMode(settings *utils.Settings, enabled bool) {
	if settings == nil {
		return
	}
	settings.Set("debug", enabled)
	settings.Set("runtime.show_model_logs", enabled)
	settings.Set("runtime.show_tool_logs", enabled)
	settings.Set("runtime.show_trigger_flow_logs", enabled)
}

func IsModelLogsEnabled(settings *utils.Settings) bool {
	return getBoolSetting(settings, "runtime.show_model_logs", false)
}

func IsToolLogsEnabled(settings *utils.Settings) bool {
	return getBoolSetting(settings, "runtime.show_tool_logs", false)
}

func IsTriggerFlowLogsEnabled(settings *utils.Settings) bool {
	return getBoolSetting(settings, "runtime.show_trigger_flow_logs", false)
}

// BindEventCenter stores event center in settings so runtime components can emit system messages.
func BindEventCenter(settings *utils.Settings, eventCenter *EventCenter) {
	if settings == nil || eventCenter == nil {
		return
	}
	settings.Set("runtime.event_center", eventCenter)
}

func eventCenterFromSettings(settings *utils.Settings) *EventCenter {
	if settings == nil {
		return nil
	}
	center, _ := settings.Get("runtime.event_center", nil, true).(*EventCenter)
	return center
}

// EmitSystemMessage sends structured system messages to the configured EventCenter.
// It is a no-op when no EventCenter is bound into settings.
func EmitSystemMessage(settings *utils.Settings, messageType types.SystemEvent, data any) error {
	center := eventCenterFromSettings(settings)
	if center == nil {
		return nil
	}
	return center.SystemMessage(messageType, data, settings)
}

func getBoolSetting(settings *utils.Settings, key string, fallback bool) bool {
	if settings == nil {
		return fallback
	}
	raw := settings.Get(key, fallback, true)
	switch typed := raw.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		}
	}
	if raw == nil {
		return fallback
	}
	return strings.EqualFold(fmt.Sprint(raw), "true")
}
