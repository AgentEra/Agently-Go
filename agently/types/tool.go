package types

type ToolInfo struct {
	Name    string         `json:"name"`
	Desc    string         `json:"desc"`
	Kwargs  map[string]any `json:"kwargs"`
	Returns any            `json:"returns,omitempty"`
	Tags    []string       `json:"tags,omitempty"`
}
