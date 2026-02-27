package triggerflow

import (
	"fmt"
	"time"
)

// FlowCreateOptions configures TriggerFlow creation.
type FlowCreateOptions struct {
	SkipExceptions bool
}

// FlowCreateOption is a functional option for flow creation.
type FlowCreateOption func(*FlowCreateOptions)

func WithFlowSkipExceptions(skip bool) FlowCreateOption {
	return func(options *FlowCreateOptions) {
		options.SkipExceptions = skip
	}
}

func parseFlowCreateOptions(raw ...any) FlowCreateOptions {
	options := FlowCreateOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case FlowCreateOption:
			typed(&options)
		case bool:
			options.SkipExceptions = typed
		case FlowCreateOptions:
			options = typed
		case *FlowCreateOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported flow-create option type: %T", item))
		}
	}
	return options
}

// ExecutionCreateOptions configures execution creation.
type ExecutionCreateOptions struct {
	SkipExceptions *bool
	Concurrency    *int
}

// ExecutionCreateOption is a functional option for execution creation.
type ExecutionCreateOption func(*ExecutionCreateOptions)

func WithExecutionSkipExceptions(skip bool) ExecutionCreateOption {
	return func(options *ExecutionCreateOptions) {
		options.SkipExceptions = &skip
	}
}

func WithExecutionConcurrency(concurrency int) ExecutionCreateOption {
	return func(options *ExecutionCreateOptions) {
		options.Concurrency = &concurrency
	}
}

func parseExecutionCreateOptions(raw ...any) ExecutionCreateOptions {
	options := ExecutionCreateOptions{}
	legacyIntIndex := 0
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case ExecutionCreateOption:
			typed(&options)
		case *bool:
			options.SkipExceptions = typed
		case *int:
			options.Concurrency = typed
		case bool:
			v := typed
			options.SkipExceptions = &v
		case int:
			v := typed
			options.Concurrency = &v
		case ExecutionCreateOptions:
			options = typed
		case *ExecutionCreateOptions:
			if typed != nil {
				options = *typed
			}
		default:
			// legacy positional fallback: first int means concurrency
			if legacyIntIndex == 0 {
				if v, ok := item.(int); ok {
					vv := v
					options.Concurrency = &vv
					legacyIntIndex++
					continue
				}
			}
			panic(fmt.Sprintf("unsupported execution-create option type: %T", item))
		}
	}
	return options
}

// RunOptions configures flow/execution run behavior.
type RunOptions struct {
	WaitForResult *bool
	Timeout       *time.Duration
	Concurrency   *int
}

// RunOption is a functional option for run behavior.
type RunOption func(*RunOptions)

func WithWaitForResult(wait bool) RunOption {
	return func(options *RunOptions) {
		options.WaitForResult = &wait
	}
}

func WaitForResult() RunOption {
	return WithWaitForResult(true)
}

func NoWaitForResult() RunOption {
	return WithWaitForResult(false)
}

func WithRunTimeout(timeout time.Duration) RunOption {
	return func(options *RunOptions) {
		options.Timeout = &timeout
	}
}

func WithRunConcurrency(concurrency int) RunOption {
	return func(options *RunOptions) {
		options.Concurrency = &concurrency
	}
}

func parseRunOptions(raw ...any) RunOptions {
	options := RunOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case RunOption:
			typed(&options)
		case bool:
			v := typed
			options.WaitForResult = &v
		case time.Duration:
			v := typed
			options.Timeout = &v
		case *int:
			options.Concurrency = typed
		case int:
			v := typed
			options.Concurrency = &v
		case RunOptions:
			options = typed
		case *RunOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported run option type: %T", item))
		}
	}
	return options
}

func splitRunInvokeOptions(raw ...any) (runRaw []any, invokeRaw []any) {
	for _, item := range raw {
		switch item.(type) {
		case nil, RunOption, bool, time.Duration, *int, int, RunOptions, *RunOptions:
			runRaw = append(runRaw, item)
		default:
			invokeRaw = append(invokeRaw, item)
		}
	}
	return
}

// DataChangeOptions configures flow/runtime data mutation behavior.
type DataChangeOptions struct {
	Emit bool
}

// DataChangeOption is a functional option for data mutation behavior.
type DataChangeOption func(*DataChangeOptions)

func WithEmitSignal(emit bool) DataChangeOption {
	return func(options *DataChangeOptions) {
		options.Emit = emit
	}
}

func EmitSignal() DataChangeOption {
	return WithEmitSignal(true)
}

func parseDataChangeOptions(raw ...any) DataChangeOptions {
	options := DataChangeOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case DataChangeOption:
			typed(&options)
		case bool:
			options.Emit = typed
		case DataChangeOptions:
			options = typed
		case *DataChangeOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported data-change option type: %T", item))
		}
	}
	return options
}

// ToOptions configures To/SideBranch behavior.
type ToOptions struct {
	SideBranch bool
	Name       string
}

// ToOption is a functional option for To behavior.
type ToOption func(*ToOptions)

func WithToSideBranch(side bool) ToOption {
	return func(options *ToOptions) {
		options.SideBranch = side
	}
}

func ToSideBranch() ToOption {
	return WithToSideBranch(true)
}

func WithToName(name string) ToOption {
	return func(options *ToOptions) {
		options.Name = name
	}
}

func parseToOptions(raw ...any) ToOptions {
	options := ToOptions{}
	stringSeen := false
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case ToOption:
			typed(&options)
		case bool:
			options.SideBranch = typed
		case string:
			if !stringSeen {
				options.Name = typed
				stringSeen = true
			}
		case ToOptions:
			options = typed
		case *ToOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported to option type: %T", item))
		}
	}
	return options
}

// BatchOptions configures batch behavior.
type BatchOptions struct {
	SideBranch  bool
	Concurrency int
}

// BatchOption is a functional option for batch behavior.
type BatchOption func(*BatchOptions)

func WithBatchSideBranch(side bool) BatchOption {
	return func(options *BatchOptions) {
		options.SideBranch = side
	}
}

func WithBatchConcurrency(concurrency int) BatchOption {
	return func(options *BatchOptions) {
		options.Concurrency = concurrency
	}
}

func parseBatchOptions(raw ...any) BatchOptions {
	options := BatchOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case BatchOption:
			typed(&options)
		case bool:
			options.SideBranch = typed
		case int:
			options.Concurrency = typed
		case BatchOptions:
			options = typed
		case *BatchOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported batch option type: %T", item))
		}
	}
	return options
}
