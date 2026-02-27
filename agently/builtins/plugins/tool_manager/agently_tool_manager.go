package toolmanager

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type AgentlyToolManager struct {
	settings *utils.Settings

	toolFuncs   map[string]any
	toolInfo    map[string]types.ToolInfo
	tagMappings map[string]map[string]struct{}
}

const PluginName = "AgentlyToolManager"

var DefaultSettings = map[string]any{}

func New(settings *utils.Settings) core.ToolManager {
	return &AgentlyToolManager{
		settings:    settings,
		toolFuncs:   map[string]any{},
		toolInfo:    map[string]types.ToolInfo{},
		tagMappings: map[string]map[string]struct{}{},
	}
}

func (m *AgentlyToolManager) Register(info types.ToolInfo, fn any) error {
	if stringsTrim(info.Name) == "" {
		return errors.New("tool name is required")
	}
	if fn == nil {
		return errors.New("tool function is required")
	}
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return fmt.Errorf("tool function for %s must be function, got %T", info.Name, fn)
	}
	if info.Kwargs == nil {
		info.Kwargs = map[string]any{}
	}
	m.toolFuncs[info.Name] = fn
	m.toolInfo[info.Name] = info
	if len(info.Tags) > 0 {
		_ = m.Tag([]string{info.Name}, info.Tags)
	}
	return nil
}

func (m *AgentlyToolManager) Tag(toolNames []string, tags []string) error {
	for _, toolName := range toolNames {
		if _, ok := m.toolInfo[toolName]; !ok {
			return fmt.Errorf("tool %s not found", toolName)
		}
		for _, tag := range tags {
			if _, ok := m.tagMappings[tag]; !ok {
				m.tagMappings[tag] = map[string]struct{}{}
			}
			m.tagMappings[tag][toolName] = struct{}{}
		}
	}
	return nil
}

func (m *AgentlyToolManager) GetToolInfo(tags []string) map[string]types.ToolInfo {
	if len(tags) == 0 {
		out := map[string]types.ToolInfo{}
		for k, v := range m.toolInfo {
			out[k] = v
		}
		return out
	}
	result := map[string]types.ToolInfo{}
	for _, tag := range tags {
		for toolName := range m.tagMappings[tag] {
			result[toolName] = m.toolInfo[toolName]
		}
	}
	return result
}

func (m *AgentlyToolManager) GetToolList(tags []string) []types.ToolInfo {
	info := m.GetToolInfo(tags)
	names := make([]string, 0, len(info))
	for name := range info {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]types.ToolInfo, 0, len(names))
	for _, name := range names {
		out = append(out, info[name])
	}
	return out
}

func (m *AgentlyToolManager) GetToolFunc(name string) (any, bool) {
	fn, ok := m.toolFuncs[name]
	return fn, ok
}

func (m *AgentlyToolManager) CallTool(ctx context.Context, name string, kwargs map[string]any) (any, error) {
	fn, ok := m.toolFuncs[name]
	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	return invokeTool(ctx, fn, kwargs)
}

func invokeTool(ctx context.Context, fn any, kwargs map[string]any) (any, error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	t := reflect.TypeOf(fn)
	v := reflect.ValueOf(fn)
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("fn must be function, got %T", fn)
	}

	args := make([]reflect.Value, 0, t.NumIn())
	switch t.NumIn() {
	case 0:
	case 1:
		in0 := t.In(0)
		if in0 == reflect.TypeOf((*context.Context)(nil)).Elem() {
			args = append(args, reflect.ValueOf(ctx))
		} else if in0.Kind() == reflect.Map {
			args = append(args, reflect.ValueOf(kwargs))
		} else {
			return nil, fmt.Errorf("unsupported tool signature: %s", t.String())
		}
	case 2:
		if t.In(0) == reflect.TypeOf((*context.Context)(nil)).Elem() && t.In(1).Kind() == reflect.Map {
			args = append(args, reflect.ValueOf(ctx), reflect.ValueOf(kwargs))
		} else {
			return nil, fmt.Errorf("unsupported tool signature: %s", t.String())
		}
	default:
		return nil, fmt.Errorf("unsupported tool signature: %s", t.String())
	}

	results := v.Call(args)
	switch len(results) {
	case 0:
		return nil, nil
	case 1:
		if err, ok := results[0].Interface().(error); ok {
			return nil, err
		}
		return results[0].Interface(), nil
	default:
		var err error
		if e, ok := results[len(results)-1].Interface().(error); ok {
			err = e
		}
		return results[0].Interface(), err
	}
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}
