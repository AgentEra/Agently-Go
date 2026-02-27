package agentextensions

import (
	"context"
	"fmt"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
)

type KeyWaiterExtension struct {
	agent      *core.BaseAgent
	keyHandler map[string][]func(any) any
}

func NewKeyWaiterExtension(agent *core.BaseAgent) *KeyWaiterExtension {
	return &KeyWaiterExtension{agent: agent, keyHandler: map[string][]func(any) any{}}
}

func (e *KeyWaiterExtension) checkKeys(keys []string, mustInPrompt bool) error {
	if output := e.agent.Prompt().Get("output", nil, true); output == nil {
		return fmt.Errorf("cannot wait keys without output prompt")
	}
	if !mustInPrompt {
		return nil
	}
	promptData, _ := e.agent.Prompt().Get("", map[string]any{}, true).(map[string]any)
	for _, key := range keys {
		if _, ok := promptData[key]; !ok {
			return fmt.Errorf("key %s not found in prompt", key)
		}
	}
	return nil
}

func (e *KeyWaiterExtension) getConsumerWithContext(ctx context.Context) (<-chan any, error) {
	response := e.agent.GetResponse()
	return response.Result.GetGeneratorWithContext(ctx, "instant")
}

func (e *KeyWaiterExtension) GetKeyResultWithContext(ctx context.Context, key string, options ...any) (any, error) {
	config := parseKeyWaiterOptions(options...)
	if err := e.checkKeys([]string{key}, config.MustInPrompt); err != nil {
		return nil, err
	}
	stream, err := e.getConsumerWithContext(ctx)
	if err != nil {
		return nil, err
	}
	for item := range stream {
		data, ok := item.(types.StreamingData)
		if ok && data.Path == key && data.IsComplete {
			return data.Value, nil
		}
	}
	return nil, nil
}

func (e *KeyWaiterExtension) GetKeyResult(args ...any) (any, error) {
	invokeRaw, key, keyWaiterRaw := parseGetKeyResultCallArgs(args...)
	ctx, cancel := core.BuildInvokeContext(e.agent.Settings(), invokeRaw...)
	defer cancel()
	return e.GetKeyResultWithContext(ctx, key, keyWaiterRaw...)
}

func (e *KeyWaiterExtension) WaitKeysWithContext(ctx context.Context, keys []string, options ...any) (<-chan [2]any, error) {
	config := parseKeyWaiterOptions(options...)
	if err := e.checkKeys(keys, config.MustInPrompt); err != nil {
		return nil, err
	}
	stream, err := e.getConsumerWithContext(ctx)
	if err != nil {
		return nil, err
	}
	wanted := map[string]struct{}{}
	for _, key := range keys {
		wanted[key] = struct{}{}
	}
	out := make(chan [2]any)
	go func() {
		defer close(out)
		for item := range stream {
			data, ok := item.(types.StreamingData)
			if !ok {
				continue
			}
			if _, exist := wanted[data.Path]; exist && data.IsComplete {
				select {
				case out <- [2]any{data.Path, data.Value}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func (e *KeyWaiterExtension) WaitKeys(args ...any) (<-chan [2]any, error) {
	invokeRaw, keys, keyWaiterRaw := parseWaitKeysCallArgs(args...)
	ctx, cancel := core.BuildInvokeContext(e.agent.Settings(), invokeRaw...)
	ch, err := e.WaitKeysWithContext(ctx, keys, keyWaiterRaw...)
	if err != nil {
		cancel()
		return nil, err
	}
	out := make(chan [2]any)
	go func() {
		defer close(out)
		defer cancel()
		for item := range ch {
			out <- item
		}
	}()
	return out, nil
}

func (e *KeyWaiterExtension) OnKey(key string, handler func(any) any) *KeyWaiterExtension {
	if _, ok := e.keyHandler[key]; !ok {
		e.keyHandler[key] = []func(any) any{}
	}
	e.keyHandler[key] = append(e.keyHandler[key], handler)
	return e
}

func (e *KeyWaiterExtension) StartWaiterWithContext(ctx context.Context, options ...any) ([][3]any, error) {
	config := parseKeyWaiterOptions(options...)
	if len(e.keyHandler) == 0 {
		return nil, fmt.Errorf("call OnKey before StartWaiter")
	}
	keys := make([]string, 0, len(e.keyHandler))
	for key := range e.keyHandler {
		keys = append(keys, key)
	}
	stream, err := e.WaitKeysWithContext(ctx, keys, config.MustInPrompt)
	if err != nil {
		return nil, err
	}
	results := make([][3]any, 0)
	for item := range stream {
		path := fmt.Sprint(item[0])
		value := item[1]
		for _, handler := range e.keyHandler[path] {
			results = append(results, [3]any{path, value, handler(value)})
		}
	}
	e.agent.Prompt().Clear()
	return results, nil
}

func (e *KeyWaiterExtension) StartWaiter(options ...any) ([][3]any, error) {
	keyWaiterRaw, invokeRaw := splitKeyWaiterInvokeOptions(options...)
	ctx, cancel := core.BuildInvokeContext(e.agent.Settings(), invokeRaw...)
	defer cancel()
	return e.StartWaiterWithContext(ctx, keyWaiterRaw...)
}

func parseGetKeyResultCallArgs(args ...any) (invokeRaw []any, key string, keyWaiterRaw []any) {
	if len(args) == 0 {
		panic("GetKeyResult expects at least key")
	}
	index := 0
	if _, ok := args[0].(context.Context); ok {
		invokeRaw = append(invokeRaw, args[0])
		index = 1
	}
	if index >= len(args) {
		panic("GetKeyResult expects key after context")
	}
	key = fmt.Sprint(args[index])
	index++
	keyWaiterRaw, invokeMore := splitKeyWaiterInvokeOptions(args[index:]...)
	invokeRaw = append(invokeRaw, invokeMore...)
	return
}

func parseWaitKeysCallArgs(args ...any) (invokeRaw []any, keys []string, keyWaiterRaw []any) {
	if len(args) == 0 {
		panic("WaitKeys expects keys")
	}
	index := 0
	if _, ok := args[0].(context.Context); ok {
		invokeRaw = append(invokeRaw, args[0])
		index = 1
	}
	if index >= len(args) {
		panic("WaitKeys expects keys after context")
	}
	typed, ok := args[index].([]string)
	if !ok {
		panic(fmt.Sprintf("WaitKeys expects []string keys, got %T", args[index]))
	}
	keys = typed
	index++
	keyWaiterRaw, invokeMore := splitKeyWaiterInvokeOptions(args[index:]...)
	invokeRaw = append(invokeRaw, invokeMore...)
	return
}
