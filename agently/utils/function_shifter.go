package utils

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

// FutureResult is a generic result carrier used by Future/Async helpers.
type FutureResult[T any] struct {
	Value T
	Err   error
}

// RunAsyncFuncInThread mirrors Agently's run_async_func_in_thread behavior.
func RunAsyncFuncInThread[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	return Await(ctx, Future(ctx, fn))
}

// Future runs fn in a goroutine and returns a one-shot result channel.
func Future[T any](ctx context.Context, fn func(context.Context) (T, error)) <-chan FutureResult[T] {
	result := make(chan FutureResult[T], 1)
	go func() {
		defer close(result)
		value, err := fn(ctx)
		select {
		case result <- FutureResult[T]{Value: value, Err: err}:
		case <-ctx.Done():
			result <- FutureResult[T]{Err: ctx.Err()}
		}
	}()
	return result
}

// Await waits for a future result.
func Await[T any](ctx context.Context, future <-chan FutureResult[T]) (T, error) {
	var zero T
	select {
	case r, ok := <-future:
		if !ok {
			return zero, errors.New("future closed without result")
		}
		return r.Value, r.Err
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// Syncify executes an asynchronous function in blocking mode.
func Syncify[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	return RunAsyncFuncInThread(ctx, fn)
}

// Asyncify wraps a sync function as a context-aware async function.
func Asyncify[T any](fn func() (T, error)) func(context.Context) (T, error) {
	return func(_ context.Context) (T, error) {
		return fn()
	}
}

// SyncifyAsyncGenerator drains an async channel into a slice.
func SyncifyAsyncGenerator[T any](ctx context.Context, stream <-chan T) ([]T, error) {
	result := make([]T, 0)
	for {
		select {
		case item, ok := <-stream:
			if !ok {
				return result, nil
			}
			result = append(result, item)
		case <-ctx.Done():
			return result, ctx.Err()
		}
	}
}

// AsyncifySyncGenerator returns a channel that emits a pre-built sequence.
func AsyncifySyncGenerator[T any](ctx context.Context, values []T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for _, item := range values {
			select {
			case out <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// AutoOptionsCall trims kwargs that are not accepted by the provided function signature.
// It supports function values called via reflection.
func AutoOptionsCall(fn any, args []reflect.Value, kwargs map[string]reflect.Value) ([]reflect.Value, error) {
	f := reflect.ValueOf(fn)
	if f.Kind() != reflect.Func {
		return nil, errors.New("fn must be function")
	}
	t := f.Type()
	in := make([]reflect.Value, 0, t.NumIn())
	in = append(in, args...)
	if len(kwargs) > 0 {
		for i := len(args); i < t.NumIn(); i++ {
			param := t.In(i)
			name := t.In(i).String()
			if v, ok := kwargs[name]; ok && v.IsValid() {
				if v.Type().AssignableTo(param) {
					in = append(in, v)
				} else if v.Type().ConvertibleTo(param) {
					in = append(in, v.Convert(param))
				} else {
					in = append(in, reflect.Zero(param))
				}
			} else {
				in = append(in, reflect.Zero(param))
			}
		}
	}
	return f.Call(in), nil
}

var futureLoopOnce sync.Once
var _ = futureLoopOnce
