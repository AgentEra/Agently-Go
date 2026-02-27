package core

import (
	"fmt"
	"sync"

	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type EventHook func(types.EventMessage)

type EventHooker interface {
	Name() string
	Events() []types.EventName
	Handler(types.EventMessage)
	OnRegister()
	OnUnregister()
}

type EventCenter struct {
	mu      sync.RWMutex
	hooks   map[types.EventName]map[string]EventHook
	hookers map[string]EventHooker
}

func NewEventCenter() *EventCenter {
	return &EventCenter{
		hooks:   map[types.EventName]map[string]EventHook{},
		hookers: map[string]EventHooker{},
	}
}

func (e *EventCenter) RegisterHook(event types.EventName, callback EventHook, hookName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if hookName == "" {
		hookName = "hook"
	}
	if _, ok := e.hooks[event]; !ok {
		e.hooks[event] = map[string]EventHook{}
	}
	e.hooks[event][hookName] = callback
}

func (e *EventCenter) UnregisterHook(event types.EventName, hookName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.hooks[event]; !ok {
		return
	}
	delete(e.hooks[event], hookName)
}

func (e *EventCenter) RegisterHookerPlugin(hooker EventHooker) {
	if hooker == nil {
		return
	}
	hooker.OnRegister()
	for _, event := range hooker.Events() {
		e.RegisterHook(event, hooker.Handler, hooker.Name())
	}
	e.mu.Lock()
	e.hookers[hooker.Name()] = hooker
	e.mu.Unlock()
}

func (e *EventCenter) UnregisterHookerPlugin(hooker any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var target EventHooker
	switch typed := hooker.(type) {
	case string:
		target = e.hookers[typed]
	case EventHooker:
		target = typed
	}
	if target == nil {
		return
	}
	for _, event := range target.Events() {
		if m, ok := e.hooks[event]; ok {
			delete(m, target.Name())
		}
	}
	target.OnUnregister()
	delete(e.hookers, target.Name())
}

func (e *EventCenter) Emit(event types.EventName, message types.EventMessage) error {
	e.mu.RLock()
	hooks := map[string]EventHook{}
	if h, ok := e.hooks[event]; ok {
		for name, cb := range h {
			hooks[name] = cb
		}
	}
	e.mu.RUnlock()

	if len(hooks) == 0 && event == types.EventNameLog {
		fmt.Println(message.Content)
		return nil
	}

	var wg sync.WaitGroup
	for _, hook := range hooks {
		wg.Add(1)
		go func(cb EventHook) {
			defer wg.Done()
			cb(message)
		}(hook)
	}
	wg.Wait()
	return nil
}

func (e *EventCenter) SystemMessage(messageType types.SystemEvent, data any, settings *utils.Settings) error {
	content := map[string]any{
		"type":     messageType,
		"data":     data,
		"settings": settings,
	}
	msg := types.NewEventMessage(types.EventNameSystem, "Agently", content)
	return e.Emit(types.EventNameSystem, msg)
}

func (e *EventCenter) CreateMessenger(moduleName string, baseMeta map[string]any) *utils.Messenger {
	return utils.NewMessenger(moduleName, e.Emit, baseMeta)
}
