package triggerflow

import (
	"reflect"
	"sync"
)

type BluePrint struct {
	Name     string
	handlers AllHandlers
	Chunks   map[string]*Chunk
	mu       sync.RWMutex
}

func NewBluePrint(name string) *BluePrint {
	if name == "" {
		name = nextID("blueprint")
	}
	return &BluePrint{
		Name: name,
		handlers: AllHandlers{
			TriggerTypeEvent:       Handlers{},
			TriggerTypeFlowData:    Handlers{},
			TriggerTypeRuntimeData: Handlers{},
		},
		Chunks: map[string]*Chunk{},
	}
}

func (b *BluePrint) AddHandler(triggerType TriggerType, target string, handler Handler, id string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if id == "" {
		id = nextID("handler")
	}
	handlers := b.handlers[triggerType]
	if handlers == nil {
		handlers = Handlers{}
		b.handlers[triggerType] = handlers
	}
	if _, ok := handlers[target]; !ok {
		handlers[target] = map[string]Handler{}
	}
	handlers[target][id] = handler
	return id
}

func (b *BluePrint) RemoveHandler(triggerType TriggerType, target string, handler any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	handlers := b.handlers[triggerType]
	if handlers == nil {
		return
	}
	targetHandlers := handlers[target]
	if targetHandlers == nil {
		return
	}
	switch typed := handler.(type) {
	case string:
		delete(targetHandlers, typed)
	case Handler:
		targetPtr := reflect.ValueOf(typed).Pointer()
		for id, h := range targetHandlers {
			if reflect.ValueOf(h).Pointer() == targetPtr {
				delete(targetHandlers, id)
				break
			}
		}
	}
}

func (b *BluePrint) RemoveAll(triggerType TriggerType, target string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.handlers[triggerType]; ok {
		b.handlers[triggerType][target] = map[string]Handler{}
	}
}

func (b *BluePrint) AddEventHandler(event string, handler Handler, id string) string {
	return b.AddHandler(TriggerTypeEvent, event, handler, id)
}

func (b *BluePrint) AddFlowDataHandler(key string, handler Handler, id string) string {
	return b.AddHandler(TriggerTypeFlowData, key, handler, id)
}

func (b *BluePrint) AddRuntimeDataHandler(key string, handler Handler, id string) string {
	return b.AddHandler(TriggerTypeRuntimeData, key, handler, id)
}

func (b *BluePrint) SnapshotHandlers() AllHandlers {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := AllHandlers{
		TriggerTypeEvent:       Handlers{},
		TriggerTypeFlowData:    Handlers{},
		TriggerTypeRuntimeData: Handlers{},
	}
	for t, handlers := range b.handlers {
		result[t] = Handlers{}
		for target, items := range handlers {
			result[t][target] = map[string]Handler{}
			for id, handler := range items {
				result[t][target][id] = handler
			}
		}
	}
	return result
}

func (b *BluePrint) Copy(name string) *BluePrint {
	if name == "" {
		name = b.Name + "-copy"
	}
	copyBP := NewBluePrint(name)
	copyBP.handlers = b.SnapshotHandlers()
	copyBP.Chunks = map[string]*Chunk{}
	for chunkName, chunk := range b.Chunks {
		copyBP.Chunks[chunkName] = chunk
	}
	return copyBP
}
