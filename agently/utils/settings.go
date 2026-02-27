package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var envPlaceholderPattern = regexp.MustCompile(`\$\{\s*ENV\.([^}]+?)\s*\}`)

// Settings is RuntimeData with path mapping and key-value mapping capabilities.
type Settings struct {
	*RuntimeData
	pathMappings *RuntimeData
	kvMappings   *RuntimeData
}

func NewSettings(name string, data map[string]any, parent *Settings) *Settings {
	var parentRuntime *RuntimeData
	var parentPaths *RuntimeData
	var parentKV *RuntimeData
	if parent != nil {
		parentRuntime = parent.RuntimeData
		parentPaths = parent.pathMappings
		parentKV = parent.kvMappings
	}
	return &Settings{
		RuntimeData:  NewRuntimeData(name, data, parentRuntime),
		pathMappings: NewRuntimeData(name+"-path-mappings", map[string]any{}, parentPaths),
		kvMappings:   NewRuntimeData(name+"-kv-mappings", map[string]any{}, parentKV),
	}
}

func (s *Settings) RegisterPathMappings(simplifyPath, actualPath string) *Settings {
	if s.kvMappings.Has(simplifyPath, true) {
		panic(fmt.Sprintf("cannot register '%s' in path mappings: key already registered in kv mappings", simplifyPath))
	}
	s.pathMappings.Set(simplifyPath, actualPath)
	return s
}

func (s *Settings) RegisterKVMappings(simplifyPath string, simplifyValue any, actualSettings map[string]any) *Settings {
	if s.pathMappings.Has(simplifyPath, true) {
		panic(fmt.Sprintf("cannot register '%s' in kv mappings: key already registered in path mappings", simplifyPath))
	}
	s.kvMappings.Set(fmt.Sprintf("%s.%v", simplifyPath, simplifyValue), actualSettings)
	return s
}

func (s *Settings) UpdateMappings(mappings map[string]any) {
	if mappings == nil {
		return
	}
	if pathMappings, ok := mappings["path_mappings"].(map[string]any); ok {
		for k, v := range pathMappings {
			s.pathMappings.Set(k, fmt.Sprint(v))
		}
	}
	if kvMappings, ok := mappings["key_value_mappings"].(map[string]any); ok {
		for key, raw := range kvMappings {
			if values, ok := raw.(map[string]any); ok {
				for value, actual := range values {
					if m, ok := actual.(map[string]any); ok {
						s.kvMappings.Set(key+"."+value, m)
					}
				}
			}
		}
	}
	if kvMappings, ok := mappings["kv_mappings"].(map[string]any); ok {
		for key, raw := range kvMappings {
			if values, ok := raw.(map[string]any); ok {
				for value, actual := range values {
					if m, ok := actual.(map[string]any); ok {
						s.kvMappings.Set(key+"."+value, m)
					}
				}
			}
		}
	}
}

func (s *Settings) LoadMappings(dataType, value string) error {
	parsed, err := loadMapByType(dataType, value)
	if err != nil {
		return err
	}
	s.UpdateMappings(parsed)
	return nil
}

func (s *Settings) SetSettings(key string, value any, autoLoadEnv bool) *Settings {
	if autoLoadEnv {
		value = DataFormatterSubstitutePlaceholder(value, mapEnv(), envPlaceholderPattern)
	}
	if mapped := s.pathMappings.Get(key, nil, true); mapped != nil {
		s.Update(map[string]any{fmt.Sprint(mapped): value})
		return s
	}
	if actual := s.kvMappings.Get(fmt.Sprintf("%s.%v", key, value), nil, true); actual != nil {
		if m, ok := actual.(map[string]any); ok {
			s.Update(m)
			return s
		}
	}
	s.Set(key, value)
	return s
}

func (s *Settings) Namespace(path string) *SettingsNamespace {
	return &SettingsNamespace{RuntimeDataNamespace: s.RuntimeData.Namespace(path)}
}

type SettingsNamespace struct {
	*RuntimeDataNamespace
}

func loadMapByType(dataType, value string) (map[string]any, error) {
	var parsed map[string]any
	var raw []byte
	isFile := strings.HasSuffix(dataType, "_file")
	format := strings.TrimSuffix(dataType, "_file")
	if isFile {
		b, err := os.ReadFile(value)
		if err != nil {
			return nil, err
		}
		raw = b
	} else {
		raw = []byte(value)
	}
	switch format {
	case "json":
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
	case "yaml":
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported mappings type: %s", dataType)
	}
	if parsed == nil {
		return nil, errors.New("parsed mappings is nil")
	}
	return parsed, nil
}

func mapEnv() map[string]any {
	out := map[string]any{}
	for _, kv := range os.Environ() {
		splits := strings.SplitN(kv, "=", 2)
		if len(splits) == 2 {
			out[splits[0]] = splits[1]
		}
	}
	return out
}
