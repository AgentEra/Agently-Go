package types

import "net/http"

type RequestData struct {
	ClientOptions map[string]any `json:"client_options"`
	Headers       http.Header    `json:"headers"`
	Data          map[string]any `json:"data"`
	RequestOpts   map[string]any `json:"request_options"`
	RequestURL    string         `json:"request_url"`
	Stream        bool           `json:"stream"`
}

type ResponseEvent string

const (
	ResponseEventError         ResponseEvent = "error"
	ResponseEventOriginalDelta ResponseEvent = "original_delta"
	ResponseEventReasoning     ResponseEvent = "reasoning_delta"
	ResponseEventDelta         ResponseEvent = "delta"
	ResponseEventToolCalls     ResponseEvent = "tool_calls"
	ResponseEventOriginalDone  ResponseEvent = "original_done"
	ResponseEventReasoningDone ResponseEvent = "reasoning_done"
	ResponseEventDone          ResponseEvent = "done"
	ResponseEventMeta          ResponseEvent = "meta"
	ResponseEventExtra         ResponseEvent = "extra"
)

type ResponseMessage struct {
	Event ResponseEvent
	Data  any
}

type StreamEventType string

const (
	StreamEventDelta StreamEventType = "delta"
	StreamEventDone  StreamEventType = "done"
)

type StreamingData struct {
	Path         string          `json:"path"`
	WildcardPath string          `json:"wildcard_path,omitempty"`
	Indexes      []int           `json:"indexes,omitempty"`
	Value        any             `json:"value"`
	Delta        string          `json:"delta,omitempty"`
	IsComplete   bool            `json:"is_complete"`
	EventType    StreamEventType `json:"event_type"`
	FullData     any             `json:"full_data,omitempty"`
}

type ModelResult struct {
	Meta         map[string]any `json:"meta"`
	OriginalData []any          `json:"original_delta"`
	OriginalDone any            `json:"original_done"`
	TextResult   string         `json:"text_result"`
	Cleaned      string         `json:"cleaned_result"`
	Parsed       any            `json:"parsed_result"`
	ResultObject any            `json:"result_object"`
	Errors       []error        `json:"-"`
	Extra        map[string]any `json:"extra"`
}
