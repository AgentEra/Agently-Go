package types

import "sync/atomic"

// JSONValue is the canonical serializable data shape used across the framework.
type JSONValue interface{}

type JSONObject map[string]JSONValue

type JSONArray []JSONValue

// AvoidCopy mirrors Agently's AVOID_COPY behavior for sentinel values.
type AvoidCopy struct {
	id uint64
}

var avoidCopyCounter uint64

func NewAvoidCopy() *AvoidCopy {
	return &AvoidCopy{id: atomic.AddUint64(&avoidCopyCounter, 1)}
}

func (a *AvoidCopy) ID() uint64 {
	if a == nil {
		return 0
	}
	return a.id
}

var Empty = NewAvoidCopy()
