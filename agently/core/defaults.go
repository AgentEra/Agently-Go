package core

import "github.com/AgentEra/Agently-Go/agently/utils"

var DefaultSettings = map[string]any{
	"prompt": map[string]any{
		"add_current_time": false,
		"role_mapping": map[string]any{
			"system":    "system",
			"developer": "developer",
			"assistant": "assistant",
			"user":      "user",
			"_":         "assistant",
		},
		"prompt_title_mapping": map[string]any{
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
		},
	},
	"session": map[string]any{
		"max_length": nil,
		"input_keys": nil,
		"reply_keys": nil,
	},
	"response": map[string]any{
		"streaming_parse":            false,
		"streaming_parse_path_style": "dot",
	},
	"runtime": map[string]any{
		"default_timeout_seconds": 120,
		"raise_error":             true,
		"raise_critical":          true,
		"show_model_logs":         false,
		"show_tool_logs":          false,
		"show_trigger_flow_logs":  false,
	},
	"plugins": map[string]any{
		"ToolManager": map[string]any{"activate": "AgentlyToolManager"},
	},
}

func NewDefaultSettings(parent *utils.Settings) *utils.Settings {
	return utils.NewSettings("global_settings", DefaultSettings, parent)
}
