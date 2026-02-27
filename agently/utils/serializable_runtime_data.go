package utils

// SerializableRuntimeData is a typed wrapper over RuntimeData used by Settings-like APIs.
type SerializableRuntimeData struct {
	*RuntimeData
}

func NewSerializableRuntimeData(name string, data map[string]any, parent *SerializableRuntimeData) *SerializableRuntimeData {
	var parentRuntime *RuntimeData
	if parent != nil {
		parentRuntime = parent.RuntimeData
	}
	return &SerializableRuntimeData{RuntimeData: NewRuntimeData(name, data, parentRuntime)}
}

func (s *SerializableRuntimeData) Namespace(path string) *SerializableRuntimeDataNamespace {
	return &SerializableRuntimeDataNamespace{RuntimeDataNamespace: s.RuntimeData.Namespace(path)}
}

type SerializableRuntimeDataNamespace struct {
	*RuntimeDataNamespace
}
