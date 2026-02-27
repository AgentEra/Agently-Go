package utils

import (
	"fmt"

	"github.com/AgentEra/Agently-Go/agently/types"
)

// Messenger routes messages via an injected emitter (typically EventCenter.Emit).
type Messenger struct {
	emitter    func(types.EventName, types.EventMessage) error
	moduleName string
	baseMeta   map[string]any
}

func NewMessenger(moduleName string, emitter func(types.EventName, types.EventMessage) error, baseMeta map[string]any) *Messenger {
	if baseMeta == nil {
		baseMeta = map[string]any{}
	}
	return &Messenger{moduleName: moduleName, emitter: emitter, baseMeta: baseMeta}
}

func (m *Messenger) UpdateBaseMeta(update map[string]any) {
	if m == nil || update == nil {
		return
	}
	for k, v := range update {
		m.baseMeta[k] = v
	}
}

func (m *Messenger) Message(content any, event types.EventName, status string, level types.MessageLevel, meta map[string]any) error {
	if m == nil {
		return nil
	}
	if event == "" {
		event = types.EventNameMessage
	}
	if level == "" {
		level = types.LevelInfo
	}
	finalMeta := map[string]any{}
	for k, v := range m.baseMeta {
		finalMeta[k] = v
	}
	for k, v := range meta {
		finalMeta[k] = v
	}
	msg := types.NewEventMessage(event, m.moduleName, content)
	msg.Status = status
	msg.Level = level
	msg.Meta = finalMeta
	if m.emitter == nil {
		return nil
	}
	return m.emitter(event, msg)
}

func (m *Messenger) Debug(content any) error {
	return m.Message(content, types.EventNameLog, "", types.LevelDebug, nil)
}
func (m *Messenger) Info(content any) error {
	return m.Message(content, types.EventNameLog, "", types.LevelInfo, nil)
}
func (m *Messenger) Warning(content any) error {
	return m.Message(content, types.EventNameLog, "", types.LevelWarning, nil)
}

func (m *Messenger) Error(err any) error {
	message := err
	if e, ok := err.(error); ok {
		message = e.Error()
	}
	emitErr := m.Message(message, types.EventNameLog, "", types.LevelError, nil)
	if emitErr != nil {
		return emitErr
	}
	if e, ok := err.(error); ok {
		return e
	}
	return fmt.Errorf("%v", err)
}

func (m *Messenger) ToConsole(content any, status string, tableName string, rowID any) error {
	meta := map[string]any{}
	if tableName != "" {
		meta["table_name"] = tableName
	}
	if rowID != nil {
		meta["row_id"] = rowID
	}
	return m.Message(content, types.EventNameConsole, status, types.LevelInfo, meta)
}

func (m *Messenger) ToData(content any, status string, meta map[string]any) error {
	return m.Message(content, types.EventNameData, status, types.LevelInfo, meta)
}
