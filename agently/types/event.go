package types

import "time"

type EventName string

const (
	EventNameMessage EventName = "message"
	EventNameError   EventName = "error"
	EventNameData    EventName = "data"
	EventNameLog     EventName = "log"
	EventNameConsole EventName = "console"
	EventNameSystem  EventName = "AGENTLY_SYS"
)

type SystemEvent string

const (
	SystemEventModelRequest SystemEvent = "MODEL_REQUEST"
	SystemEventTool         SystemEvent = "TOOL"
	SystemEventTriggerFlow  SystemEvent = "TRIGGER_FLOW"
)

type MessageLevel string

const (
	LevelDebug    MessageLevel = "DEBUG"
	LevelInfo     MessageLevel = "INFO"
	LevelWarning  MessageLevel = "WARNING"
	LevelError    MessageLevel = "ERROR"
	LevelCritical MessageLevel = "CRITICAL"
)

type EventMessage struct {
	Event      EventName      `json:"event"`
	Status     string         `json:"status,omitempty"`
	ModuleName string         `json:"module_name,omitempty"`
	Content    any            `json:"content"`
	Exception  error          `json:"-"`
	Level      MessageLevel   `json:"level"`
	Meta       map[string]any `json:"meta,omitempty"`
	Timestamp  int64          `json:"timestamp"`
}

func NewEventMessage(event EventName, moduleName string, content any) EventMessage {
	return EventMessage{
		Event:      event,
		ModuleName: moduleName,
		Content:    content,
		Level:      LevelInfo,
		Meta:       map[string]any{},
		Timestamp:  time.Now().UnixMilli(),
	}
}
