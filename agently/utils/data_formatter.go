package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var defaultPlaceholderPattern = regexp.MustCompile(`\$\{\s*([^}]+?)\s*\}`)

type Formatter struct{}

var DataFormatter = Formatter{}

func (Formatter) Sanitize(value any, remainType bool) any {
	return DataFormatterSanitize(value, remainType)
}

func (Formatter) ToStrKeyDict(value any, valueFormat string, defaultKey string, defaultValue any) map[string]any {
	return DataFormatterToStrKeyDict(value, valueFormat, defaultKey, defaultValue)
}

func (Formatter) FromSchemaToKwargsFormat(schema map[string]any) map[string]any {
	return DataFormatterFromSchemaToKwargsFormat(schema)
}

func (Formatter) SubstitutePlaceholder(obj any, mappings map[string]any, pattern *regexp.Regexp) any {
	return DataFormatterSubstitutePlaceholder(obj, mappings, pattern)
}

func DataFormatterSanitize(value any, remainType bool) any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return typed
	case time.Time:
		return typed.Format(time.RFC3339)
	case map[string]any:
		out := map[string]any{}
		for k, v := range typed {
			out[k] = DataFormatterSanitize(v, remainType)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, v := range typed {
			out = append(out, DataFormatterSanitize(v, remainType))
		}
		return out
	}

	rv := reflect.ValueOf(value)
	rt := rv.Type()

	if rv.Kind() == reflect.Pointer && !rv.IsNil() {
		return DataFormatterSanitize(rv.Elem().Interface(), remainType)
	}

	if rv.Kind() == reflect.Map {
		out := map[string]any{}
		iter := rv.MapRange()
		for iter.Next() {
			out[fmt.Sprint(iter.Key().Interface())] = DataFormatterSanitize(iter.Value().Interface(), remainType)
		}
		return out
	}

	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out = append(out, DataFormatterSanitize(rv.Index(i).Interface(), remainType))
		}
		return out
	}

	if rv.Kind() == reflect.Struct {
		out := map[string]any{}
		for i := 0; i < rv.NumField(); i++ {
			field := rt.Field(i)
			if field.PkgPath != "" {
				continue
			}
			out[field.Name] = DataFormatterSanitize(rv.Field(i).Interface(), remainType)
		}
		if len(out) > 0 {
			return out
		}
	}

	if rv.Kind() == reflect.Func {
		if remainType {
			return value
		}
		return rt.String()
	}

	if rv.Kind() == reflect.Interface {
		return DataFormatterSanitize(rv.Elem().Interface(), remainType)
	}

	if remainType {
		return value
	}
	return fmt.Sprint(value)
}

func DataFormatterToStrKeyDict(value any, valueFormat string, defaultKey string, defaultValue any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		out := map[string]any{}
		for k, v := range m {
			out[k] = formatValue(v, valueFormat)
		}
		return out
	}

	rv := reflect.ValueOf(value)
	if rv.IsValid() && rv.Kind() == reflect.Map {
		out := map[string]any{}
		iter := rv.MapRange()
		for iter.Next() {
			out[fmt.Sprint(iter.Key().Interface())] = formatValue(iter.Value().Interface(), valueFormat)
		}
		return out
	}

	if defaultKey != "" {
		return map[string]any{defaultKey: formatValue(value, valueFormat)}
	}
	if defaultMap, ok := defaultValue.(map[string]any); ok {
		return defaultMap
	}
	return map[string]any{}
}

func formatValue(value any, valueFormat string) any {
	switch strings.ToLower(strings.TrimSpace(valueFormat)) {
	case "serializable":
		return DataFormatterSanitize(value, false)
	case "str":
		return fmt.Sprint(DataFormatterSanitize(value, false))
	default:
		return value
	}
}

func DataFormatterFromSchemaToKwargsFormat(inputSchema map[string]any) map[string]any {
	if inputSchema == nil {
		return nil
	}
	t, _ := inputSchema["type"].(string)
	if t != "object" {
		return nil
	}
	result := map[string]any{}
	if props, ok := inputSchema["properties"].(map[string]any); ok {
		for name, propRaw := range props {
			prop, _ := propRaw.(map[string]any)
			typeText := "any"
			if pt, ok := prop["type"].(string); ok && pt != "" {
				typeText = pt
			}
			descParts := []string{}
			for k, v := range prop {
				if k == "type" || k == "title" {
					continue
				}
				descParts = append(descParts, fmt.Sprintf("%s: %v", k, v))
			}
			result[name] = []any{typeText, strings.Join(descParts, ";")}
		}
	}
	if ap, ok := inputSchema["additionalProperties"]; ok {
		switch typed := ap.(type) {
		case bool:
			if typed {
				result["<*>"] = []any{"any", ""}
			}
		case map[string]any:
			typeText := "any"
			if pt, ok := typed["type"].(string); ok && pt != "" {
				typeText = pt
			}
			descParts := []string{}
			for k, v := range typed {
				if k == "type" || k == "title" {
					continue
				}
				descParts = append(descParts, fmt.Sprintf("%s: %v", k, v))
			}
			result["<*>"] = []any{typeText, strings.Join(descParts, ";")}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func DataFormatterSubstitutePlaceholder(obj any, variableMappings map[string]any, placeholderPattern *regexp.Regexp) any {
	if placeholderPattern == nil {
		placeholderPattern = defaultPlaceholderPattern
	}
	if variableMappings == nil {
		return obj
	}

	switch typed := obj.(type) {
	case string:
		full := placeholderPattern.FindStringSubmatch(typed)
		if len(full) == 2 && strings.TrimSpace(full[0]) == typed {
			if v, ok := variableMappings[strings.TrimSpace(full[1])]; ok {
				return v
			}
			return typed
		}
		return placeholderPattern.ReplaceAllStringFunc(typed, func(m string) string {
			sub := placeholderPattern.FindStringSubmatch(m)
			if len(sub) != 2 {
				return m
			}
			if v, ok := variableMappings[strings.TrimSpace(sub[1])]; ok {
				return fmt.Sprint(v)
			}
			return m
		})
	case map[string]any:
		out := map[string]any{}
		for k, v := range typed {
			key := fmt.Sprint(DataFormatterSubstitutePlaceholder(k, variableMappings, placeholderPattern))
			out[key] = DataFormatterSubstitutePlaceholder(v, variableMappings, placeholderPattern)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, v := range typed {
			out = append(out, DataFormatterSubstitutePlaceholder(v, variableMappings, placeholderPattern))
		}
		return out
	default:
		rv := reflect.ValueOf(obj)
		if !rv.IsValid() {
			return obj
		}
		if rv.Kind() == reflect.Map {
			out := map[string]any{}
			iter := rv.MapRange()
			for iter.Next() {
				k := DataFormatterSubstitutePlaceholder(iter.Key().Interface(), variableMappings, placeholderPattern)
				v := DataFormatterSubstitutePlaceholder(iter.Value().Interface(), variableMappings, placeholderPattern)
				out[fmt.Sprint(k)] = v
			}
			return out
		}
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			out := make([]any, 0, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out = append(out, DataFormatterSubstitutePlaceholder(rv.Index(i).Interface(), variableMappings, placeholderPattern))
			}
			return out
		}
		return obj
	}
}
