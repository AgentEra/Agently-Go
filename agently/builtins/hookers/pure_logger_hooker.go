package hookers

import (
	"fmt"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type PureLoggerHooker struct {
	logger *utils.AgentlyLogger
}

func NewPureLoggerHooker(logger *utils.AgentlyLogger) *PureLoggerHooker {
	return &PureLoggerHooker{logger: logger}
}

func (h *PureLoggerHooker) Name() string { return "PureLoggerHooker" }

func (h *PureLoggerHooker) Events() []types.EventName {
	return []types.EventName{types.EventNameMessage, types.EventNameLog}
}

func (h *PureLoggerHooker) OnRegister()   {}
func (h *PureLoggerHooker) OnUnregister() {}

func (h *PureLoggerHooker) Handler(message types.EventMessage) {
	if h == nil || h.logger == nil {
		return
	}
	status := statusPrefix(message.Status)
	content := fmt.Sprintf("%s[%s] %v", status, message.ModuleName, message.Content)
	switch message.Level {
	case types.LevelDebug:
		h.logger.Debug(content)
	case types.LevelWarning:
		h.logger.Warn(content)
	case types.LevelError, types.LevelCritical:
		h.logger.Error(content)
	default:
		h.logger.Info(content)
	}
}

func statusPrefix(status string) string {
	switch status {
	case "INIT":
		return ">> "
	case "DOING":
		return ".. "
	case "PENDING":
		return ".. "
	case "SUCCESS":
		return "OK "
	case "FAILED":
		return "XX "
	case "UNKNOWN":
		return "?? "
	case "":
		return ""
	default:
		return status + " "
	}
}

var _ core.EventHooker = (*PureLoggerHooker)(nil)
