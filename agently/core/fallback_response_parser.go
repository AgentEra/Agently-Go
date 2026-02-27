package core

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/AgentEra/Agently-Go/agently/types"
)

type fallbackResponseParser struct {
	stream  <-chan types.ResponseMessage
	once    sync.Once
	done    chan struct{}
	meta    map[string]any
	dataAll []types.ResponseMessage
	text    strings.Builder
	parsed  any
	err     error
}

func NewFallbackResponseParser(stream <-chan types.ResponseMessage) ResponseParser {
	return &fallbackResponseParser{
		stream: stream,
		done:   make(chan struct{}),
		meta:   map[string]any{},
	}
}

func (f *fallbackResponseParser) ensureConsumed() {
	f.once.Do(func() {
		defer close(f.done)
		for msg := range f.stream {
			f.dataAll = append(f.dataAll, msg)
			switch msg.Event {
			case types.ResponseEventDelta:
				f.text.WriteString(fmt.Sprint(msg.Data))
			case types.ResponseEventDone:
				f.parsed = msg.Data
			case types.ResponseEventMeta:
				if m, ok := msg.Data.(map[string]any); ok {
					for k, v := range m {
						f.meta[k] = v
					}
				}
			case types.ResponseEventError:
				if err, ok := msg.Data.(error); ok {
					f.err = err
				}
			}
		}
	})
}

func (f *fallbackResponseParser) wait(ctx context.Context) error {
	f.ensureConsumed()
	select {
	case <-f.done:
		return f.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *fallbackResponseParser) GetMeta(ctx context.Context) (map[string]any, error) {
	if err := f.wait(ctx); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k, v := range f.meta {
		out[k] = v
	}
	return out, nil
}

func (f *fallbackResponseParser) GetData(ctx context.Context, dataType string) (any, error) {
	if err := f.wait(ctx); err != nil {
		return nil, err
	}
	switch dataType {
	case "original":
		if len(f.dataAll) == 0 {
			return nil, nil
		}
		return f.dataAll[len(f.dataAll)-1].Data, nil
	case "all":
		out := make([]types.ResponseMessage, len(f.dataAll))
		copy(out, f.dataAll)
		return out, nil
	default:
		if f.parsed != nil {
			return f.parsed, nil
		}
		return f.text.String(), nil
	}
}

func (f *fallbackResponseParser) GetDataObject(ctx context.Context) (any, error) {
	return f.GetData(ctx, "parsed")
}

func (f *fallbackResponseParser) GetText(ctx context.Context) (string, error) {
	if err := f.wait(ctx); err != nil {
		return "", err
	}
	return f.text.String(), nil
}

func (f *fallbackResponseParser) GetStream(ctx context.Context, streamType string, options ...any) (<-chan any, error) {
	config := ParseStreamOptions(options...)
	specific := config.Specific
	out := make(chan any, 64)
	go func() {
		defer close(out)
		for {
			select {
			case item, ok := <-f.stream:
				if !ok {
					return
				}
				switch streamType {
				case "all":
					out <- item
				case "specific":
					for _, evt := range specific {
						if evt == string(item.Event) {
							out <- item
							break
						}
					}
				case "original":
					if strings.HasPrefix(string(item.Event), "original") {
						out <- item.Data
					}
				default:
					if item.Event == types.ResponseEventDelta {
						out <- item.Data
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
