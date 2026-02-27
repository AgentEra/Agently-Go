package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func BuildDotPath(keys []any) string {
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys))
	for i, key := range keys {
		switch v := key.(type) {
		case int:
			parts = append(parts, fmt.Sprintf("[%d]", v))
		case int64:
			parts = append(parts, fmt.Sprintf("[%d]", v))
		default:
			s := fmt.Sprint(v)
			if s == "*" || s == "[]" || s == "[*]" {
				parts = append(parts, "[*]")
				continue
			}
			if i == 0 {
				parts = append(parts, s)
			} else {
				parts = append(parts, "."+s)
			}
		}
	}
	return strings.Join(parts, "")
}

func BuildSlashPath(keys []any) string {
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys)+1)
	parts = append(parts, "")
	for _, key := range keys {
		parts = append(parts, fmt.Sprint(key))
	}
	return strings.Join(parts, "/")
}

func ConvertDotToSlash(dotPath string) string {
	if strings.TrimSpace(dotPath) == "" {
		return "/"
	}
	parts := make([]string, 0)
	buffer := strings.Builder{}
	for i := 0; i < len(dotPath); i++ {
		ch := dotPath[i]
		switch ch {
		case '[':
			if buffer.Len() > 0 {
				parts = append(parts, buffer.String())
				buffer.Reset()
			}
			end := strings.IndexByte(dotPath[i:], ']')
			if end < 0 {
				parts = append(parts, dotPath[i:])
				continue
			}
			actualEnd := i + end
			parts = append(parts, dotPath[i:actualEnd+1])
			i = actualEnd
		case '.':
			if buffer.Len() > 0 {
				parts = append(parts, buffer.String())
				buffer.Reset()
			}
		default:
			buffer.WriteByte(ch)
		}
	}
	if buffer.Len() > 0 {
		parts = append(parts, buffer.String())
	}
	return "/" + strings.Join(parts, "/")
}

func ConvertSlashToDot(slashPath string) string {
	trimmed := strings.TrimSpace(slashPath)
	if trimmed == "" || trimmed == "/" {
		return ""
	}
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	built := make([]string, 0, len(parts))
	for i, part := range parts {
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			built = append(built, part)
			continue
		}
		if i == 0 {
			built = append(built, part)
		} else {
			built = append(built, "."+part)
		}
	}
	return strings.Join(built, "")
}

func ExtractPossiblePaths(schema map[string]any, style string) ([]string, error) {
	if schema == nil {
		return []string{""}, nil
	}
	paths := map[string]struct{}{}
	var walk func(value any, pathKeys []any)
	walk = func(value any, pathKeys []any) {
		current := BuildDotPath(pathKeys)
		if strings.EqualFold(style, "slash") {
			current = BuildSlashPath(pathKeys)
		}
		paths[current] = struct{}{}

		switch typed := value.(type) {
		case map[string]any:
			for k, v := range typed {
				walk(v, append(pathKeys, k))
			}
		case []any:
			for _, item := range typed {
				walk(item, append(pathKeys, "[*]"))
			}
		}
	}
	walk(schema, []any{})
	out := make([]string, 0, len(paths))
	for p := range paths {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

func ExtractParsingKeyOrders(schema map[string]any, style string) ([]string, error) {
	if schema == nil {
		return []string{""}, nil
	}
	ordered := make([]string, 0)
	seen := map[string]struct{}{}
	add := func(path string) {
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		ordered = append(ordered, path)
	}

	var walk func(value any, pathKeys []any)
	walk = func(value any, pathKeys []any) {
		current := BuildDotPath(pathKeys)
		if strings.EqualFold(style, "slash") {
			current = BuildSlashPath(pathKeys)
		}
		switch typed := value.(type) {
		case map[string]any:
			for k, v := range typed {
				walk(v, append(pathKeys, k))
			}
			add(current)
		case []any:
			for _, item := range typed {
				walk(item, append(pathKeys, "[*]"))
			}
			add(current)
		default:
			add(current)
		}
	}
	walk(schema, []any{})
	return ordered, nil
}

var reArrayIndex = regexp.MustCompile(`\[(\d+)\]`)

func GetValueByPath(data any, path string, style string) any {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	keys := make([]any, 0)
	if strings.EqualFold(style, "slash") {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		for _, part := range parts {
			if part == "" {
				continue
			}
			if i, err := strconv.Atoi(part); err == nil {
				keys = append(keys, i)
			} else {
				keys = append(keys, part)
			}
		}
	} else {
		parts := strings.Split(path, ".")
		for _, part := range parts {
			if part == "" {
				continue
			}
			left := part
			for {
				idx := strings.IndexByte(left, '[')
				if idx < 0 {
					if left != "" {
						keys = append(keys, left)
					}
					break
				}
				if idx > 0 {
					keys = append(keys, left[:idx])
				}
				end := strings.IndexByte(left[idx:], ']')
				if end < 0 {
					keys = append(keys, left[idx:])
					break
				}
				inside := left[idx+1 : idx+end]
				if inside == "*" {
					keys = append(keys, "[*]")
				} else if i, err := strconv.Atoi(inside); err == nil {
					keys = append(keys, i)
				}
				left = left[idx+end+1:]
			}
		}
	}
	return resolveByKeys(data, keys)
}

func resolveByKeys(current any, keys []any) any {
	if len(keys) == 0 {
		return current
	}
	key := keys[0]
	remaining := keys[1:]

	switch typed := key.(type) {
	case string:
		if typed == "[*]" {
			s, ok := current.([]any)
			if !ok {
				return nil
			}
			results := make([]any, 0)
			for _, item := range s {
				v := resolveByKeys(item, remaining)
				if arr, ok := v.([]any); ok {
					results = append(results, arr...)
				} else if v != nil {
					results = append(results, v)
				}
			}
			return results
		}
		if m, ok := current.(map[string]any); ok {
			v, exists := m[typed]
			if !exists {
				return nil
			}
			return resolveByKeys(v, remaining)
		}
		return nil
	case int:
		s, ok := current.([]any)
		if !ok || typed < 0 || typed >= len(s) {
			return nil
		}
		return resolveByKeys(s[typed], remaining)
	default:
		return nil
	}
}

func ComputeWildcardPath(path string) (string, []int) {
	if !strings.Contains(path, "[") {
		return path, nil
	}
	indexes := make([]int, 0)
	wildcard := reArrayIndex.ReplaceAllStringFunc(path, func(raw string) string {
		match := reArrayIndex.FindStringSubmatch(raw)
		if len(match) == 2 {
			if idx, err := strconv.Atoi(match[1]); err == nil {
				indexes = append(indexes, idx)
			}
		}
		return "[*]"
	})
	return wildcard, indexes
}
