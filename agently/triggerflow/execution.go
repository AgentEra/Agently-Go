package triggerflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type EventData struct {
	TriggerEvent string
	TriggerType  TriggerType
	Event        string
	Type         TriggerType
	Value        any
	ExecutionID  string
	Settings     *utils.Settings

	execution  *Execution
	layerMarks []string
}

func (d *EventData) GetFlowData(path string, defaultValue any) any {
	return d.execution.GetFlowData(path, defaultValue)
}
func (d *EventData) SetFlowData(path string, value any, options ...any) error {
	return d.execution.SetFlowData(path, value, options...)
}
func (d *EventData) AppendFlowData(path string, value any, options ...any) error {
	return d.execution.AppendFlowData(path, value, options...)
}
func (d *EventData) DelFlowData(path string, options ...any) error {
	return d.execution.DelFlowData(path, options...)
}

func (d *EventData) GetRuntimeData(path string, defaultValue any) any {
	return d.execution.runtimeData.Get(path, defaultValue, true)
}
func (d *EventData) SetRuntimeData(path string, value any, options ...any) error {
	return d.execution.SetRuntimeData(path, value, options...)
}
func (d *EventData) AppendRuntimeData(path string, value any, options ...any) error {
	return d.execution.AppendRuntimeData(path, value, options...)
}
func (d *EventData) DelRuntimeData(path string, options ...any) error {
	return d.execution.DelRuntimeData(path, options...)
}

func (d *EventData) Emit(triggerEvent string, value any, triggerType TriggerType) error {
	return d.execution.EmitWithMarks(triggerEvent, value, d.layerMarksCopy(), triggerType)
}

func (d *EventData) EmitWithMarks(triggerEvent string, value any, marks []string, triggerType TriggerType) error {
	return d.execution.EmitWithMarks(triggerEvent, value, marks, triggerType)
}

func (d *EventData) SetResult(result any) {
	d.execution.SetResult(result)
}

func (d *EventData) PutIntoStream(item any) error { return d.execution.PutIntoStream(item) }
func (d *EventData) StopStream() error            { return d.execution.StopStream() }

func (d *EventData) UpperLayerMark() string {
	if len(d.layerMarks) > 1 {
		return d.layerMarks[len(d.layerMarks)-2]
	}
	return ""
}

func (d *EventData) LayerMark() string {
	if len(d.layerMarks) > 0 {
		return d.layerMarks[len(d.layerMarks)-1]
	}
	return ""
}

func (d *EventData) LayerIn() {
	d.layerMarks = append(d.layerMarks, nextID("layer"))
}

func (d *EventData) LayerOut() {
	if len(d.layerMarks) == 0 {
		return
	}
	d.layerMarks = d.layerMarks[:len(d.layerMarks)-1]
}

func (d *EventData) layerMarksCopy() []string {
	out := make([]string, len(d.layerMarks))
	copy(out, d.layerMarks)
	return out
}

// Execution is a signal-driven runtime that dispatches handlers by signal type and key.
type Execution struct {
	ID string

	handlers    AllHandlers
	triggerFlow *TriggerFlow

	runtimeData       *utils.RuntimeData
	systemRuntimeData *utils.RuntimeData
	skipExceptions    bool
	semaphore         chan struct{}
	settings          *utils.Settings

	started bool

	runtimeQueue    chan any
	runtimeConsumer *utils.GeneratorConsumer

	result      any
	resultSet   bool
	resultReady chan struct{}
	mu          sync.RWMutex
}

func NewExecution(handlers AllHandlers, flow *TriggerFlow, id string, skipExceptions bool, concurrency int) *Execution {
	if id == "" {
		id = nextID("execution")
	}
	exec := &Execution{
		ID:                id,
		handlers:          handlers,
		triggerFlow:       flow,
		runtimeData:       utils.NewRuntimeData("execution-runtime", map[string]any{}, nil),
		systemRuntimeData: utils.NewRuntimeData("execution-system-runtime", map[string]any{}, nil),
		skipExceptions:    skipExceptions,
		settings:          utils.NewSettings("TriggerFlowExecution-Settings", map[string]any{}, flow.settings),
		runtimeQueue:      make(chan any, 256),
		resultReady:       make(chan struct{}),
	}
	if concurrency > 0 {
		exec.semaphore = make(chan struct{}, concurrency)
	}
	exec.systemRuntimeData.Set("result", RuntimeStreamStop)
	return exec
}

func (e *Execution) SetSettings(key string, value any) *Execution {
	e.settings.SetSettings(key, value, false)
	return e
}

func (e *Execution) SetConcurrency(concurrency int) *Execution {
	if concurrency > 0 {
		e.semaphore = make(chan struct{}, concurrency)
	} else {
		e.semaphore = nil
	}
	return e
}

func (e *Execution) Emit(triggerEvent string, value any) error {
	return e.EmitWithMarks(triggerEvent, value, nil, TriggerTypeEvent)
}

func (e *Execution) EmitWithMarks(triggerEvent string, value any, marks []string, triggerType TriggerType) error {
	if marks == nil {
		marks = []string{}
	}
	showTriggerLogs := core.IsTriggerFlowLogsEnabled(e.settings)
	if showTriggerLogs {
		_ = core.EmitSystemMessage(e.settings, types.SystemEventTriggerFlow, map[string]any{
			"TYPE":  triggerType,
			"EVENT": triggerEvent,
			"VALUE": value,
		})
	}
	e.mu.Lock()
	e.started = true
	e.mu.Unlock()

	handlers := e.handlers[triggerType]
	if handlers == nil {
		return nil
	}
	targetHandlers := handlers[triggerEvent]
	if len(targetHandlers) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(targetHandlers))

	for handlerID, handler := range targetHandlers {
		h := handler
		wg.Add(1)
		go func() {
			defer wg.Done()
			if showTriggerLogs {
				_ = core.EmitSystemMessage(e.settings, types.SystemEventTriggerFlow, map[string]any{
					"EVENT":   triggerEvent,
					"TYPE":    triggerType,
					"HANDLER": handlerID,
				})
			}
			if e.semaphore != nil {
				e.semaphore <- struct{}{}
				defer func() { <-e.semaphore }()
			}
			data := &EventData{
				TriggerEvent: triggerEvent,
				TriggerType:  triggerType,
				Event:        triggerEvent,
				Type:         triggerType,
				Value:        value,
				ExecutionID:  e.ID,
				Settings:     e.settings,
				execution:    e,
				layerMarks:   append([]string{}, marks...),
			}
			if _, err := h(data); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)
	if e.skipExceptions {
		return nil
	}
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Execution) changeRuntimeData(op string, key string, value any, emit bool) error {
	switch op {
	case "set":
		e.runtimeData.Set(key, value)
	case "append":
		e.runtimeData.Append(key, value)
	case "del":
		e.runtimeData.Delete(key)
	}
	if emit {
		return e.EmitWithMarks(key, e.runtimeData.Get(key, nil, true), nil, TriggerTypeRuntimeData)
	}
	return nil
}

func (e *Execution) SetRuntimeData(key string, value any, options ...any) error {
	config := parseDataChangeOptions(options...)
	return e.changeRuntimeData("set", key, value, config.Emit)
}

func (e *Execution) AppendRuntimeData(key string, value any, options ...any) error {
	config := parseDataChangeOptions(options...)
	return e.changeRuntimeData("append", key, value, config.Emit)
}

func (e *Execution) DelRuntimeData(key string, options ...any) error {
	config := parseDataChangeOptions(options...)
	return e.changeRuntimeData("del", key, nil, config.Emit)
}

func (e *Execution) Start(initialValue any, options ...any) (any, error) {
	config := parseRunOptions(options...)
	waitForResult := true
	if config.WaitForResult != nil {
		waitForResult = *config.WaitForResult
	}
	timeout := 10 * time.Second
	if config.Timeout != nil {
		timeout = *config.Timeout
	}
	if err := e.EmitWithMarks("START", initialValue, nil, TriggerTypeEvent); err != nil {
		return nil, err
	}
	if waitForResult {
		ctx := context.Background()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		return e.GetResult(ctx)
	}
	return nil, nil
}

func (e *Execution) PutIntoStream(item any) error {
	e.runtimeQueue <- item
	return nil
}

func (e *Execution) StopStream() error {
	e.runtimeQueue <- RuntimeStreamStop
	return nil
}

func (e *Execution) consumeRuntimeStream(ctx context.Context, initialValue any, timeout time.Duration) <-chan any {
	out := make(chan any)
	go func() {
		defer close(out)
		if !e.started {
			_, _ = e.Start(initialValue, false, 0)
		}
		for {
			if timeout > 0 {
				select {
				case item := <-e.runtimeQueue:
					if item == RuntimeStreamStop {
						return
					}
					out <- item
				case <-time.After(timeout):
					return
				case <-ctx.Done():
					return
				}
			} else {
				select {
				case item := <-e.runtimeQueue:
					if item == RuntimeStreamStop {
						return
					}
					out <- item
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

func (e *Execution) GetAsyncRuntimeStreamWithContext(ctx context.Context, initialValue any, timeout time.Duration) (<-chan any, error) {
	if e.runtimeConsumer == nil {
		e.runtimeConsumer = utils.NewGeneratorConsumer(e.consumeRuntimeStream(ctx, initialValue, timeout))
	}
	return e.runtimeConsumer.Subscribe(ctx)
}

func (e *Execution) GetAsyncRuntimeStream(args ...any) (<-chan any, error) {
	ctx, cancel, initialValue, timeout := e.parseRuntimeStreamArgs("GetAsyncRuntimeStream", args...)
	ch, err := e.GetAsyncRuntimeStreamWithContext(ctx, initialValue, timeout)
	if err != nil {
		cancel()
		return nil, err
	}
	out := make(chan any)
	go func() {
		defer close(out)
		defer cancel()
		for item := range ch {
			out <- item
		}
	}()
	return out, nil
}

func (e *Execution) GetRuntimeStreamWithContext(ctx context.Context, initialValue any, timeout time.Duration) (<-chan any, error) {
	return e.GetAsyncRuntimeStreamWithContext(ctx, initialValue, timeout)
}

func (e *Execution) GetRuntimeStream(args ...any) (<-chan any, error) {
	return e.GetAsyncRuntimeStream(args...)
}

func (e *Execution) SetResult(result any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.resultSet {
		e.result = result
		e.resultSet = true
		close(e.resultReady)
	}
}

func (e *Execution) GetResultWithContext(ctx context.Context) (any, error) {
	select {
	case <-e.resultReady:
		e.mu.RLock()
		defer e.mu.RUnlock()
		return e.result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("get result timeout/cancelled: %w", ctx.Err())
	}
}

func (e *Execution) GetResult(options ...any) (any, error) {
	ctx, cancel := core.BuildInvokeContext(e.settings, options...)
	defer cancel()
	return e.GetResultWithContext(ctx)
}

func (e *Execution) GetFlowData(path string, defaultValue any) any {
	return e.triggerFlow.flowData.Get(path, defaultValue, true)
}

func (e *Execution) SetFlowData(path string, value any, options ...any) error {
	config := parseDataChangeOptions(options...)
	return e.triggerFlow.changeFlowData(e, "set", path, value, config.Emit)
}

func (e *Execution) AppendFlowData(path string, value any, options ...any) error {
	config := parseDataChangeOptions(options...)
	return e.triggerFlow.changeFlowData(e, "append", path, value, config.Emit)
}

func (e *Execution) DelFlowData(path string, options ...any) error {
	config := parseDataChangeOptions(options...)
	return e.triggerFlow.changeFlowData(e, "del", path, nil, config.Emit)
}

func (e *Execution) parseRuntimeStreamArgs(method string, args ...any) (context.Context, context.CancelFunc, any, time.Duration) {
	var invokeRaw []any
	var initialValue any
	consumeIndex := 0
	if len(args) > 0 {
		if _, ok := args[0].(context.Context); ok {
			invokeRaw = append(invokeRaw, args[0])
			consumeIndex = 1
		}
	}
	if consumeIndex < len(args) {
		initialValue = args[consumeIndex]
		consumeIndex++
	}
	runRaw, invokeMore := splitRunInvokeOptions(args[consumeIndex:]...)
	invokeRaw = append(invokeRaw, invokeMore...)
	runOptions := parseRunOptions(runRaw...)
	timeout := 10 * time.Second
	if runOptions.Timeout != nil {
		timeout = *runOptions.Timeout
	}
	ctx, cancel := core.BuildInvokeContext(e.settings, invokeRaw...)
	if ctx == nil {
		panic(fmt.Sprintf("%s failed to build context", method))
	}
	return ctx, cancel, initialValue, timeout
}
