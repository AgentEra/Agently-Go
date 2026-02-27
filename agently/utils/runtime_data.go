package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// RuntimeData provides dot-path aware hierarchical mutable data with inheritance.
type RuntimeData struct {
	mu     sync.RWMutex
	data   map[string]any
	parent *RuntimeData
	name   string
}

func NewRuntimeData(name string, data map[string]any, parent *RuntimeData) *RuntimeData {
	if data == nil {
		data = map[string]any{}
	}
	return &RuntimeData{name: name, data: deepCopyMap(data), parent: parent}
}

func (r *RuntimeData) Name() string { return r.name }

func (r *RuntimeData) Parent() *RuntimeData { return r.parent }

func (r *RuntimeData) SetParent(parent *RuntimeData) { r.parent = parent }

func (r *RuntimeData) Data(inherit bool) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !inherit || r.parent == nil {
		return deepCopyMap(r.data)
	}
	parent := r.parent.Data(true)
	return mergeMaps(parent, r.data)
}

func (r *RuntimeData) Get(path string, defaultValue any, inherit bool) any {
	if strings.TrimSpace(path) == "" {
		if m, ok := defaultValue.(map[string]any); ok && m == nil {
			return r.Data(inherit)
		}
		return r.Data(inherit)
	}
	base := r.Data(inherit)
	value, ok := getByDotPath(base, path)
	if !ok {
		return defaultValue
	}
	return deepCopyAny(value)
}

func (r *RuntimeData) Has(path string, inherit bool) bool {
	base := r.Data(inherit)
	_, ok := getByDotPath(base, path)
	return ok
}

func (r *RuntimeData) Set(path string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	setByDotPath(&r.data, path, value, false)
}

func (r *RuntimeData) SetCover(path string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	setByDotPath(&r.data, path, value, true)
}

func (r *RuntimeData) SetDefault(path string, value any, inherit bool) any {
	if r.Get(path, nil, inherit) == nil {
		r.Set(path, value)
	}
	return r.Get(path, nil, inherit)
}

func (r *RuntimeData) Update(data map[string]any) {
	if data == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range data {
		setByDotPath(&r.data, k, v, false)
	}
}

func (r *RuntimeData) Delete(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deleteByDotPath(&r.data, path)
}

func (r *RuntimeData) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = map[string]any{}
}

func (r *RuntimeData) Append(path string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := getByDotPath(r.data, path)
	if !ok || cur == nil {
		setByDotPath(&r.data, path, []any{deepCopyAny(value)}, true)
		return
	}
	switch typed := cur.(type) {
	case []any:
		typed = append(typed, deepCopyAny(value))
		setByDotPath(&r.data, path, typed, true)
	case []string:
		setByDotPath(&r.data, path, appendStringSlice(typed, fmt.Sprint(value)), true)
	default:
		setByDotPath(&r.data, path, []any{deepCopyAny(cur), deepCopyAny(value)}, true)
	}
}

func (r *RuntimeData) Extend(path string, values []any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := getByDotPath(r.data, path)
	if !ok || cur == nil {
		setByDotPath(&r.data, path, deepCopySlice(values), true)
		return
	}
	switch typed := cur.(type) {
	case []any:
		typed = append(typed, deepCopySlice(values)...)
		setByDotPath(&r.data, path, typed, true)
	default:
		merged := []any{deepCopyAny(cur)}
		merged = append(merged, deepCopySlice(values)...)
		setByDotPath(&r.data, path, merged, true)
	}
}

func (r *RuntimeData) Keys(inherit bool) []string {
	m := r.Data(inherit)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (r *RuntimeData) Namespace(path string) *RuntimeDataNamespace {
	return &RuntimeDataNamespace{root: r, namespace: strings.Trim(path, ".")}
}

func (r *RuntimeData) String() string {
	b, _ := json.Marshal(r.Data(true))
	return string(b)
}

// RuntimeDataNamespace is a view over RuntimeData with an automatic path prefix.
type RuntimeDataNamespace struct {
	root      *RuntimeData
	namespace string
}

func (n *RuntimeDataNamespace) full(path string) string {
	path = strings.Trim(path, ".")
	if path == "" {
		return n.namespace
	}
	if n.namespace == "" {
		return path
	}
	return n.namespace + "." + path
}

func (n *RuntimeDataNamespace) Data(inherit bool) any {
	return n.root.Get(n.namespace, map[string]any{}, inherit)
}

func (n *RuntimeDataNamespace) Get(path string, defaultValue any, inherit bool) any {
	return n.root.Get(n.full(path), defaultValue, inherit)
}

func (n *RuntimeDataNamespace) Set(path string, value any) {
	n.root.Set(n.full(path), value)
}

func (n *RuntimeDataNamespace) SetCover(path string, value any) {
	n.root.SetCover(n.full(path), value)
}

func (n *RuntimeDataNamespace) SetDefault(path string, value any, inherit bool) any {
	return n.root.SetDefault(n.full(path), value, inherit)
}

func (n *RuntimeDataNamespace) Update(data map[string]any) {
	for k, v := range data {
		n.Set(k, v)
	}
}

func (n *RuntimeDataNamespace) Append(path string, value any) {
	n.root.Append(n.full(path), value)
}

func (n *RuntimeDataNamespace) Extend(path string, values []any) {
	n.root.Extend(n.full(path), values)
}

func (n *RuntimeDataNamespace) Delete(path string) {
	n.root.Delete(n.full(path))
}

func (n *RuntimeDataNamespace) Has(path string, inherit bool) bool {
	return n.root.Has(n.full(path), inherit)
}

func mergeMaps(base map[string]any, override map[string]any) map[string]any {
	result := deepCopyMap(base)
	for k, v := range override {
		if existing, ok := result[k]; ok {
			result[k] = mergeValue(existing, v)
		} else {
			result[k] = deepCopyAny(v)
		}
	}
	return result
}

func mergeValue(existing, incoming any) any {
	switch ex := existing.(type) {
	case map[string]any:
		if in, ok := incoming.(map[string]any); ok {
			return mergeMaps(ex, in)
		}
	case []any:
		switch in := incoming.(type) {
		case []any:
			return appendUniqueAny(ex, in)
		default:
			return appendUniqueAny(ex, []any{in})
		}
	}
	return deepCopyAny(incoming)
}

func appendUniqueAny(base []any, incoming []any) []any {
	result := deepCopySlice(base)
	for _, it := range incoming {
		if !containsAny(result, it) {
			result = append(result, deepCopyAny(it))
		}
	}
	return result
}

func appendStringSlice(base []string, v string) []string {
	for _, it := range base {
		if it == v {
			return base
		}
	}
	return append(base, v)
}

func containsAny(slice []any, target any) bool {
	for _, it := range slice {
		if reflect.DeepEqual(it, target) {
			return true
		}
	}
	return false
}

func deepCopyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = deepCopyAny(v)
	}
	return out
}

func deepCopySlice(in []any) []any {
	if in == nil {
		return nil
	}
	out := make([]any, 0, len(in))
	for _, v := range in {
		out = append(out, deepCopyAny(v))
	}
	return out
}

func deepCopyAny(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return deepCopyMap(typed)
	case []any:
		return deepCopySlice(typed)
	case []string:
		out := make([]string, len(typed))
		copy(out, typed)
		return out
	default:
		return typed
	}
}

func getByDotPath(data map[string]any, path string) (any, bool) {
	if strings.TrimSpace(path) == "" {
		return data, true
	}
	parts := strings.Split(path, ".")
	var current any = data
	for _, part := range parts {
		switch typed := current.(type) {
		case map[string]any:
			val, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = val
		default:
			return nil, false
		}
	}
	return current, true
}

func setByDotPath(root *map[string]any, path string, value any, cover bool) {
	if strings.TrimSpace(path) == "" {
		if m, ok := value.(map[string]any); ok {
			*root = deepCopyMap(m)
		}
		return
	}
	parts := strings.Split(path, ".")
	current := *root
	for i, part := range parts {
		if i == len(parts)-1 {
			if !cover {
				if old, exists := current[part]; exists {
					current[part] = mergeValue(old, value)
					return
				}
			}
			current[part] = deepCopyAny(value)
			return
		}
		next, ok := current[part]
		if !ok {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			child = map[string]any{}
			current[part] = child
		}
		current = child
	}
}

func deleteByDotPath(root *map[string]any, path string) {
	if strings.TrimSpace(path) == "" {
		*root = map[string]any{}
		return
	}
	parts := strings.Split(path, ".")
	current := *root
	for i, part := range parts {
		if i == len(parts)-1 {
			delete(current, part)
			return
		}
		next, ok := current[part]
		if !ok {
			return
		}
		child, ok := next.(map[string]any)
		if !ok {
			return
		}
		current = child
	}
}
