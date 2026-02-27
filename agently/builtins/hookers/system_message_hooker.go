package hookers

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type SystemMessageHooker struct {
	logger *utils.AgentlyLogger

	mu           sync.Mutex
	currentTable string
	currentRow   string
	currentStage string
	streaming    bool
}

func NewSystemMessageHooker(logger *utils.AgentlyLogger) *SystemMessageHooker {
	return &SystemMessageHooker{logger: logger}
}

func (h *SystemMessageHooker) Name() string { return "SystemMessageHooker" }

func (h *SystemMessageHooker) Events() []types.EventName {
	return []types.EventName{types.EventNameSystem}
}

func (h *SystemMessageHooker) OnRegister()   {}
func (h *SystemMessageHooker) OnUnregister() {}

func (h *SystemMessageHooker) Handler(message types.EventMessage) {
	content, ok := message.Content.(map[string]any)
	if !ok {
		return
	}
	settings, _ := content["settings"].(*utils.Settings)
	messageType := types.SystemEvent(fmt.Sprint(content["type"]))
	data := content["data"]

	switch messageType {
	case types.SystemEventModelRequest:
		h.handleModelMessage(settings, data)
	case types.SystemEventTool:
		h.handleToolMessage(settings, data)
	case types.SystemEventTriggerFlow:
		h.handleTriggerFlowMessage(settings, data)
	}
}

func (h *SystemMessageHooker) handleModelMessage(settings *utils.Settings, data any) {
	if !core.IsModelLogsEnabled(settings) {
		return
	}
	payload, ok := data.(map[string]any)
	if !ok {
		return
	}
	agentName := fmt.Sprint(payload["agent_name"])
	responseID := fmt.Sprint(payload["response_id"])

	content, _ := payload["content"].(map[string]any)
	stage := fmt.Sprint(content["stage"])
	detail := fmt.Sprint(content["detail"])
	delta, _ := content["delta"].(bool)

	if delta {
		h.mu.Lock()
		defer h.mu.Unlock()

		if h.currentTable == agentName && h.currentRow == responseID && h.currentStage == stage {
			fmt.Print(colorText(detail, "gray", false, false))
			return
		}

		header := colorText(
			fmt.Sprintf("[Agent-%s] - [Request-%s]", agentName, responseID),
			"blue",
			true,
			false,
		)
		stageLabel := colorText("Stage:", "cyan", true, false)
		stageValue := colorText(stage, "yellow", false, true)
		detailLabel := colorText("Detail:\n", "cyan", true, false)
		fmt.Printf("%s\n%s %s\n%s%s", header, stageLabel, stageValue, detailLabel, colorText(detail, "green", false, false))

		h.currentTable = agentName
		h.currentRow = responseID
		h.currentStage = stage
		h.streaming = true
		return
	}

	h.mu.Lock()
	if h.streaming {
		fmt.Println()
		h.streaming = false
	}
	h.mu.Unlock()

	header := colorText(
		fmt.Sprintf("[Agent-%s] - [Response-%s]", agentName, responseID),
		"blue",
		true,
		false,
	)
	stageLabel := colorText("Stage:", "cyan", true, false)
	stageValue := colorText(stage, "yellow", false, true)
	detailLabel := colorText("Detail:\n", "cyan", true, false)
	h.logger.Info(fmt.Sprintf("%s\n%s %s\n%s%s", header, stageLabel, stageValue, detailLabel, colorText(detail, "gray", false, false)))
}

func (h *SystemMessageHooker) handleToolMessage(settings *utils.Settings, data any) {
	if !core.IsToolLogsEnabled(settings) {
		return
	}
	title := colorText("[Tool Using Result]:", "blue", true, false)
	body := colorText(formatLogData(data), "gray", false, false)
	h.logger.Info(fmt.Sprintf("%s\n%s", title, body))
}

func (h *SystemMessageHooker) handleTriggerFlowMessage(settings *utils.Settings, data any) {
	if !core.IsTriggerFlowLogsEnabled(settings) {
		return
	}
	h.logger.Info(colorText(fmt.Sprintf("[TriggerFlow] %s", formatLogData(data)), "yellow", true, false))
}

func formatLogData(data any) string {
	switch typed := data.(type) {
	case nil:
		return "<nil>"
	case string:
		return typed
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines := make([]string, 0, len(keys))
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("%s: %v", key, typed[key]))
		}
		return strings.Join(lines, "\n")
	default:
		return fmt.Sprintf("%v", data)
	}
}

func colorText(text string, color string, bold bool, underline bool) string {
	colors := map[string]int{
		"black":   30,
		"red":     31,
		"green":   32,
		"yellow":  33,
		"blue":    34,
		"magenta": 35,
		"cyan":    36,
		"white":   37,
		"gray":    90,
	}
	codes := make([]string, 0, 3)
	if bold {
		codes = append(codes, "1")
	}
	if underline {
		codes = append(codes, "4")
	}
	if code, ok := colors[color]; ok {
		codes = append(codes, fmt.Sprint(code))
	}
	if len(codes) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

var _ core.EventHooker = (*SystemMessageHooker)(nil)
