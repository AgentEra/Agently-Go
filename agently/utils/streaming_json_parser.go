package utils

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/AgentEra/Agently-Go/agently/types"
)

// StreamingJSONParser parses streamed JSON chunks and emits delta/done events.
type StreamingJSONParser struct {
	schema               map[string]any
	completer            *StreamingJSONCompleter
	previousData         any
	currentData          any
	fieldCompletion      map[string]struct{}
	expectedFieldOrder   []string
	allPossibleFieldPath map[string]struct{}
}

func NewStreamingJSONParser(schema map[string]any) *StreamingJSONParser {
	orders, _ := ExtractParsingKeyOrders(schema, "dot")
	paths, _ := ExtractPossiblePaths(schema, "dot")
	all := map[string]struct{}{}
	for _, p := range paths {
		all[p] = struct{}{}
	}
	return &StreamingJSONParser{
		schema:               schema,
		completer:            NewStreamingJSONCompleter(),
		previousData:         map[string]any{},
		currentData:          map[string]any{},
		fieldCompletion:      map[string]struct{}{},
		expectedFieldOrder:   orders,
		allPossibleFieldPath: all,
	}
}

func (s *StreamingJSONParser) ParseChunk(chunk string) ([]types.StreamingData, error) {
	s.completer.Append(chunk)
	completedJSON := s.completer.Complete()
	located := LocateOutputJSON(completedJSON, s.schema)
	if located == "" {
		return nil, nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(located), &parsed); err != nil {
		return nil, nil
	}

	s.previousData = deepCopyAny(s.currentData)
	s.currentData = parsed
	out := make([]types.StreamingData, 0)
	s.compareAndGenerate(parsed, s.previousData, []any{}, &out)
	return out, nil
}

func (s *StreamingJSONParser) ParseStream(ctx context.Context, chunkStream <-chan string) <-chan types.StreamingData {
	out := make(chan types.StreamingData)
	go func() {
		defer close(out)
		for {
			select {
			case chunk, ok := <-chunkStream:
				if !ok {
					final := s.Finalize()
					for _, evt := range final {
						select {
						case out <- evt:
						case <-ctx.Done():
							return
						}
					}
					return
				}
				events, _ := s.ParseChunk(chunk)
				for _, evt := range events {
					select {
					case out <- evt:
					case <-ctx.Done():
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (s *StreamingJSONParser) Finalize() []types.StreamingData {
	out := make([]types.StreamingData, 0)
	var walk func(value any, path []any)
	walk = func(value any, path []any) {
		currentPath := BuildDotPath(path)
		if currentPath != "" {
			if _, ok := s.fieldCompletion[currentPath]; !ok {
				evt := types.StreamingData{
					Path:       currentPath,
					Value:      deepCopyAny(value),
					IsComplete: true,
					EventType:  types.StreamEventDone,
					FullData:   deepCopyAny(s.currentData),
				}
				evt.WildcardPath, evt.Indexes = toWildcard(evt.Path)
				s.fieldCompletion[currentPath] = struct{}{}
				out = append(out, evt)
			}
		}
		switch typed := value.(type) {
		case map[string]any:
			for k, v := range typed {
				walk(v, append(path, k))
			}
		case []any:
			for i, v := range typed {
				walk(v, append(path, i))
			}
		}
	}
	walk(s.currentData, []any{})
	return out
}

func (s *StreamingJSONParser) compareAndGenerate(current, previous any, path []any, out *[]types.StreamingData) {
	currentPath := BuildDotPath(path)
	switch curr := current.(type) {
	case string:
		prev, _ := previous.(string)
		if curr != prev {
			delta := curr
			if prev != "" && len(curr) >= len(prev) && curr[:len(prev)] == prev {
				delta = curr[len(prev):]
			}
			if delta != "" {
				evt := types.StreamingData{
					Path:       currentPath,
					Value:      curr,
					Delta:      delta,
					IsComplete: false,
					EventType:  types.StreamEventDelta,
					FullData:   deepCopyAny(s.currentData),
				}
				evt.WildcardPath, evt.Indexes = toWildcard(evt.Path)
				*out = append(*out, evt)
			}
		}
	case map[string]any:
		prevMap, _ := previous.(map[string]any)
		for k, v := range curr {
			s.compareAndGenerate(v, prevMap[k], append(path, k), out)
		}
	case []any:
		prevList, _ := previous.([]any)
		for i, v := range curr {
			var prevVal any
			if i < len(prevList) {
				prevVal = prevList[i]
			}
			s.compareAndGenerate(v, prevVal, append(path, i), out)
		}
	default:
		if !reflect.DeepEqual(current, previous) && currentPath != "" {
			evt := types.StreamingData{
				Path:       currentPath,
				Value:      deepCopyAny(current),
				Delta:      anyToString(current),
				IsComplete: false,
				EventType:  types.StreamEventDelta,
				FullData:   deepCopyAny(s.currentData),
			}
			evt.WildcardPath, evt.Indexes = toWildcard(evt.Path)
			*out = append(*out, evt)
		}
	}
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 32)
	case int:
		return strconv.Itoa(t)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func toWildcard(path string) (string, []int) {
	wildcard, indexes := ComputeWildcardPath(path)
	return wildcard, indexes
}
