package triggerflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type TriggerFlow struct {
	Name string

	settings *utils.Settings
	flowData *utils.RuntimeData

	bluePrint      *BluePrint
	skipExceptions bool
	executions     map[string]*Execution
	mu             sync.RWMutex

	startProcess *Process

	Chunks map[string]*Chunk
}

func New(bluePrint *BluePrint, name string, options ...any) *TriggerFlow {
	config := parseFlowCreateOptions(options...)
	if name == "" {
		name = nextID("triggerflow")
	}
	if bluePrint == nil {
		bluePrint = NewBluePrint("")
	}
	flow := &TriggerFlow{
		Name:           name,
		settings:       core.NewDefaultSettings(nil),
		flowData:       utils.NewRuntimeData("TriggerFlow-FlowData", map[string]any{}, nil),
		bluePrint:      bluePrint,
		skipExceptions: config.SkipExceptions,
		executions:     map[string]*Execution{},
	}
	flow.startProcess = NewProcess(flow.Chunk, "START", TriggerTypeEvent, flow.bluePrint, NewBlockData(nil, map[string]any{}), nil)
	flow.Chunks = flow.bluePrint.Chunks
	return flow
}

func (f *TriggerFlow) Settings() *utils.Settings { return f.settings }

func (f *TriggerFlow) SetSettings(key string, value any) *TriggerFlow {
	f.settings.SetSettings(key, value, false)
	return f
}

func (f *TriggerFlow) GetFlowData(path string, defaultValue any) any {
	return f.flowData.Get(path, defaultValue, true)
}

func (f *TriggerFlow) SetFlowData(path string, value any, options ...any) error {
	config := parseDataChangeOptions(options...)
	return f.changeFlowData(nil, "set", path, value, config.Emit)
}

func (f *TriggerFlow) AppendFlowData(path string, value any, options ...any) error {
	config := parseDataChangeOptions(options...)
	return f.changeFlowData(nil, "append", path, value, config.Emit)
}

func (f *TriggerFlow) DelFlowData(path string, options ...any) error {
	config := parseDataChangeOptions(options...)
	return f.changeFlowData(nil, "del", path, nil, config.Emit)
}

func (f *TriggerFlow) changeFlowData(caller *Execution, op string, key string, value any, emit bool) error {
	switch op {
	case "set":
		f.flowData.Set(key, value)
		value = f.flowData.Get(key, nil, true)
	case "append":
		f.flowData.Append(key, value)
		value = f.flowData.Get(key, nil, true)
	case "del":
		f.flowData.Delete(key)
		value = nil
	}
	if !emit {
		return nil
	}

	f.mu.RLock()
	executions := make([]*Execution, 0, len(f.executions))
	for _, execution := range f.executions {
		executions = append(executions, execution)
	}
	f.mu.RUnlock()

	for _, execution := range executions {
		if caller != nil && caller.ID == execution.ID {
			continue
		}
		if handlers := execution.handlers[TriggerTypeFlowData]; handlers != nil {
			if _, ok := handlers[key]; ok {
				if err := execution.EmitWithMarks(key, value, nil, TriggerTypeFlowData); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (f *TriggerFlow) Chunk(handlerOrName any) *Chunk {
	switch typed := handlerOrName.(type) {
	case string:
		name := typed
		chunk := NewChunk(nil, name)
		f.bluePrint.Chunks[name] = chunk
		return chunk
	case NamedChunk:
		chunk := NewChunk(typed.Handler, typed.Name)
		f.bluePrint.Chunks[typed.Name] = chunk
		return chunk
	case Handler:
		name := nextID("chunk_handler")
		chunk := NewChunk(typed, name)
		f.bluePrint.Chunks[name] = chunk
		return chunk
	default:
		chunk := NewChunk(nil, nextID("chunk"))
		f.bluePrint.Chunks[chunk.Name] = chunk
		return chunk
	}
}

func (f *TriggerFlow) CreateExecution(options ...any) *Execution {
	config := parseExecutionCreateOptions(options...)
	executionID := nextID("execution")
	skip := f.skipExceptions
	if config.SkipExceptions != nil {
		skip = *config.SkipExceptions
	}
	cc := 0
	if config.Concurrency != nil {
		cc = *config.Concurrency
	}
	handlers := f.bluePrint.SnapshotHandlers()
	execution := NewExecution(handlers, f, executionID, skip, cc)
	f.mu.Lock()
	f.executions[executionID] = execution
	f.mu.Unlock()
	return execution
}

func (f *TriggerFlow) RemoveExecution(execution any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch typed := execution.(type) {
	case string:
		delete(f.executions, typed)
	case *Execution:
		delete(f.executions, typed.ID)
	}
}

func (f *TriggerFlow) StartExecution(initialValue any, options ...any) (*Execution, any, error) {
	config := parseRunOptions(options...)
	waitForResult := true
	if config.WaitForResult != nil {
		waitForResult = *config.WaitForResult
	}
	timeout := 10 * time.Second
	if config.Timeout != nil {
		timeout = *config.Timeout
	}
	executionCreateOptions := []any{}
	if config.Concurrency != nil {
		executionCreateOptions = append(executionCreateOptions, WithExecutionConcurrency(*config.Concurrency))
	}
	execution := f.CreateExecution(executionCreateOptions...)
	result, err := execution.Start(initialValue, WithWaitForResult(waitForResult), WithRunTimeout(timeout))
	return execution, result, err
}

func (f *TriggerFlow) Start(initialValue any, options ...any) (any, error) {
	config := parseRunOptions(options...)
	waitForResult := true
	if config.WaitForResult != nil {
		waitForResult = *config.WaitForResult
	}
	timeout := 10 * time.Second
	if config.Timeout != nil {
		timeout = *config.Timeout
	}
	executionCreateOptions := []any{}
	if config.Concurrency != nil {
		executionCreateOptions = append(executionCreateOptions, WithExecutionConcurrency(*config.Concurrency))
	}
	execution := f.CreateExecution(executionCreateOptions...)
	result, err := execution.Start(initialValue, NoWaitForResult(), WithRunTimeout(timeout))
	if err != nil {
		return nil, err
	}
	if waitForResult {
		return execution.GetResult(core.WithTimeout(timeout))
	}
	return result, nil
}

func (f *TriggerFlow) GetAsyncRuntimeStreamWithContext(ctx context.Context, initialValue any, options ...any) (<-chan any, error) {
	config := parseRunOptions(options...)
	timeout := 10 * time.Second
	if config.Timeout != nil {
		timeout = *config.Timeout
	}
	executionCreateOptions := []any{}
	if config.Concurrency != nil {
		executionCreateOptions = append(executionCreateOptions, WithExecutionConcurrency(*config.Concurrency))
	}
	execution := f.CreateExecution(executionCreateOptions...)
	return execution.GetAsyncRuntimeStreamWithContext(ctx, initialValue, timeout)
}

func (f *TriggerFlow) GetAsyncRuntimeStream(args ...any) (<-chan any, error) {
	ctx, cancel, initialValue, runRaw := f.parseRuntimeStreamArgs("GetAsyncRuntimeStream", args...)
	ch, err := f.GetAsyncRuntimeStreamWithContext(ctx, initialValue, runRaw...)
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

func (f *TriggerFlow) GetRuntimeStreamWithContext(ctx context.Context, initialValue any, options ...any) (<-chan any, error) {
	return f.GetAsyncRuntimeStreamWithContext(ctx, initialValue, options...)
}

func (f *TriggerFlow) GetRuntimeStream(args ...any) (<-chan any, error) {
	return f.GetAsyncRuntimeStream(args...)
}

func (f *TriggerFlow) SaveBluePrint() *BluePrint {
	return f.bluePrint.Copy("")
}

func (f *TriggerFlow) When(trigger any, mode string) *Process {
	return f.startProcess.When(trigger, mode)
}
func (f *TriggerFlow) To(target any, options ...any) *Process {
	return f.startProcess.To(target, options...)
}
func (f *TriggerFlow) SideBranch(target any, name string) *Process {
	return f.startProcess.SideBranch(target, name)
}
func (f *TriggerFlow) Batch(chunks []any, options ...any) *Process {
	return f.startProcess.Batch(chunks, options...)
}
func (f *TriggerFlow) ForEach(options ...any) *Process {
	concurrency := 0
	for _, item := range options {
		switch typed := item.(type) {
		case nil:
			continue
		case int:
			concurrency = typed
		default:
			panic(fmt.Sprintf("unsupported for-each option type: %T", item))
		}
	}
	return f.startProcess.ForEach(concurrency)
}

func (f *TriggerFlow) String() string {
	return fmt.Sprintf("TriggerFlow<%s>", f.Name)
}

func (f *TriggerFlow) parseRuntimeStreamArgs(method string, args ...any) (context.Context, context.CancelFunc, any, []any) {
	invokeRaw := make([]any, 0)
	index := 0
	if len(args) > 0 {
		if _, ok := args[0].(context.Context); ok {
			invokeRaw = append(invokeRaw, args[0])
			index = 1
		}
	}
	var initialValue any
	if index < len(args) {
		initialValue = args[index]
		index++
	}
	runRaw, invokeMore := splitRunInvokeOptions(args[index:]...)
	invokeRaw = append(invokeRaw, invokeMore...)
	ctx, cancel := core.BuildInvokeContext(f.settings, invokeRaw...)
	if ctx == nil {
		panic(fmt.Sprintf("%s failed to build context", method))
	}
	return ctx, cancel, initialValue, runRaw
}
