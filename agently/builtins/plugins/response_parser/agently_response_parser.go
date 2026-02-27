package responseparser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type AgentlyResponseParser struct {
	agentName  string
	responseID string
	prompt     *core.Prompt
	settings   *utils.Settings

	responseStream <-chan types.ResponseMessage
	consumer       *utils.GeneratorConsumer
	initOnce       sync.Once
	initErr        error

	promptObject      types.PromptObject
	streamingParser   *utils.StreamingJSONParser
	fullResultData    *types.ModelResult
	streamingCanceled bool
}

const PluginName = "AgentlyResponseParser"

var DefaultSettings = map[string]any{
	"$global": map[string]any{
		"response": map[string]any{
			"streaming_parse":            false,
			"streaming_parse_path_style": "dot",
		},
	},
}

func New(agentName, responseID string, prompt *core.Prompt, response <-chan types.ResponseMessage, settings *utils.Settings) core.ResponseParser {
	obj, _ := prompt.ToPromptObject()
	parser := &AgentlyResponseParser{
		agentName:      agentName,
		responseID:     responseID,
		prompt:         prompt,
		settings:       settings,
		responseStream: response,
		promptObject:   obj,
		fullResultData: &types.ModelResult{
			Meta:         map[string]any{},
			OriginalData: []any{},
			OriginalDone: map[string]any{},
			TextResult:   "",
			Cleaned:      "",
			Parsed:       nil,
			ResultObject: nil,
			Errors:       []error{},
			Extra:        map[string]any{},
		},
	}
	if obj.OutputFormat == types.OutputJSON {
		if schema, ok := obj.Output.(map[string]any); ok {
			parser.streamingParser = utils.NewStreamingJSONParser(schema)
		}
	}
	return parser
}

func (p *AgentlyResponseParser) ensureConsumer() error {
	p.initOnce.Do(func() {
		stream := make(chan any, 256)
		go func() {
			defer close(stream)
			for msg := range p.responseStream {
				p.processMessage(msg)
				stream <- msg
			}
		}()
		p.consumer = utils.NewGeneratorConsumer(stream)
	})
	return p.initErr
}

func (p *AgentlyResponseParser) processMessage(msg types.ResponseMessage) {
	switch msg.Event {
	case types.ResponseEventOriginalDelta:
		p.fullResultData.OriginalData = append(p.fullResultData.OriginalData, msg.Data)
	case types.ResponseEventDelta:
		p.fullResultData.TextResult += fmt.Sprint(msg.Data)
		if core.IsModelLogsEnabled(p.settings) {
			if p.settings.Get("$log.cancel_logs", false, true) != true {
				p.emitModelSystemMessage("Streaming", fmt.Sprint(msg.Data), true)
			} else if !p.streamingCanceled {
				p.emitModelSystemMessage("Streaming", fmt.Sprintf("(logging canceled for Agent-%s / Response-%s)\n", p.agentName, p.responseID), true)
				p.streamingCanceled = true
			}
		}
	case types.ResponseEventOriginalDone:
		p.fullResultData.OriginalDone = msg.Data
	case types.ResponseEventDone:
		p.fullResultData.TextResult = fmt.Sprint(msg.Data)
		if p.promptObject.OutputFormat == types.OutputJSON {
			parsedDone := false
			if schema, ok := p.promptObject.Output.(map[string]any); ok {
				candidates := []string{fmt.Sprint(msg.Data)}
				if text := strings.TrimSpace(p.fullResultData.TextResult); text != "" && text != candidates[0] {
					candidates = append(candidates, text)
				}
				for _, candidate := range candidates {
					cleaned := utils.LocateOutputJSON(candidate, schema)
					if cleaned == "" {
						continue
					}
					completer := utils.NewStreamingJSONCompleter()
					completer.Reset(cleaned)
					completed := completer.Complete()
					parsed := map[string]any{}
					if err := json.Unmarshal([]byte(completed), &parsed); err != nil {
						continue
					}
					p.fullResultData.Cleaned = completed
					p.fullResultData.Parsed = parsed
					p.fullResultData.ResultObject = parsed
					parsedDone = true
					break
				}
			}
			if core.IsModelLogsEnabled(p.settings) {
				if parsedDone {
					p.emitModelSystemMessage("Done", fmt.Sprint(msg.Data), false)
				} else {
					p.emitModelSystemMessage("Done", "Can not parse this result as requested output schema.", false)
				}
			}
		} else {
			p.fullResultData.Parsed = msg.Data
			p.fullResultData.ResultObject = msg.Data
			if core.IsModelLogsEnabled(p.settings) && p.settings.Get("$log.cancel_logs", false, true) != true {
				p.emitModelSystemMessage("Done", fmt.Sprint(msg.Data), false)
			}
		}
	case types.ResponseEventMeta:
		if m, ok := msg.Data.(map[string]any); ok {
			for k, v := range m {
				p.fullResultData.Meta[k] = v
			}
		}
	case types.ResponseEventExtra:
		if m, ok := msg.Data.(map[string]any); ok {
			for k, v := range m {
				p.fullResultData.Extra[k] = v
			}
		}
	case types.ResponseEventError:
		if err, ok := msg.Data.(error); ok {
			p.fullResultData.Errors = append(p.fullResultData.Errors, err)
		}
	}
}

func (p *AgentlyResponseParser) emitModelSystemMessage(stage string, detail any, delta bool) {
	_ = core.EmitSystemMessage(p.settings, types.SystemEventModelRequest, map[string]any{
		"agent_name":  p.agentName,
		"response_id": p.responseID,
		"content": map[string]any{
			"stage":  stage,
			"detail": detail,
			"delta":  delta,
		},
	})
}

func (p *AgentlyResponseParser) waitResult(ctx context.Context) error {
	if err := p.ensureConsumer(); err != nil {
		return err
	}
	_, err := p.consumer.Result(ctx)
	return err
}

func (p *AgentlyResponseParser) GetMeta(ctx context.Context) (map[string]any, error) {
	if err := p.waitResult(ctx); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k, v := range p.fullResultData.Meta {
		out[k] = v
	}
	return out, nil
}

func (p *AgentlyResponseParser) GetData(ctx context.Context, dataType string) (any, error) {
	if err := p.waitResult(ctx); err != nil {
		return nil, err
	}
	switch dataType {
	case "original":
		return p.fullResultData.OriginalDone, nil
	case "all":
		copyData := *p.fullResultData
		return copyData, nil
	default:
		return p.fullResultData.Parsed, nil
	}
}

func (p *AgentlyResponseParser) GetDataObject(ctx context.Context) (any, error) {
	if p.promptObject.OutputFormat != types.OutputJSON {
		return nil, fmt.Errorf("cannot create data object for non-json output")
	}
	if err := p.waitResult(ctx); err != nil {
		return nil, err
	}
	return p.fullResultData.ResultObject, nil
}

func (p *AgentlyResponseParser) GetText(ctx context.Context) (string, error) {
	if err := p.waitResult(ctx); err != nil {
		return "", err
	}
	return p.fullResultData.TextResult, nil
}

func (p *AgentlyResponseParser) GetStream(ctx context.Context, streamType string, options ...any) (<-chan any, error) {
	if err := p.ensureConsumer(); err != nil {
		return nil, err
	}
	if streamType == "" {
		streamType = "delta"
	}
	config := core.ParseStreamOptions(options...)
	specific := config.Specific
	src, err := p.consumer.Subscribe(ctx)
	if err != nil {
		return nil, err
	}
	out := make(chan any)

	go func() {
		defer close(out)
		pathStyle := fmt.Sprint(p.settings.Get("response.streaming_parse_path_style", "dot", true))
		if pathStyle != "dot" && pathStyle != "slash" {
			pathStyle = "dot"
		}
		for item := range src {
			msg, ok := item.(types.ResponseMessage)
			if !ok {
				continue
			}
			switch streamType {
			case "all":
				select {
				case out <- msg:
				case <-ctx.Done():
					return
				}
			case "delta":
				if msg.Event == types.ResponseEventDelta {
					select {
					case out <- msg.Data:
					case <-ctx.Done():
						return
					}
				}
			case "specific":
				events := specific
				if len(events) == 0 {
					events = []string{"delta"}
				}
				for _, evt := range events {
					if evt == string(msg.Event) {
						select {
						case out <- msg:
						case <-ctx.Done():
							return
						}
						break
					}
				}
			case "instant", "streaming_parse":
				if p.streamingParser == nil {
					continue
				}
				if msg.Event == types.ResponseEventDelta {
					events, _ := p.streamingParser.ParseChunk(fmt.Sprint(msg.Data))
					for _, evt := range events {
						if pathStyle == "slash" {
							evt.Path = utils.ConvertDotToSlash(evt.Path)
						}
						select {
						case out <- evt:
						case <-ctx.Done():
							return
						}
					}
				} else if msg.Event == types.ResponseEventToolCalls {
					evt := types.StreamingData{Path: "$tool_calls", Value: msg.Data}
					select {
					case out <- evt:
					case <-ctx.Done():
						return
					}
				} else if msg.Event == types.ResponseEventDone {
					for _, evt := range p.streamingParser.Finalize() {
						if pathStyle == "slash" {
							evt.Path = utils.ConvertDotToSlash(evt.Path)
						}
						select {
						case out <- evt:
						case <-ctx.Done():
							return
						}
					}
				}
			case "original":
				if strings.HasPrefix(string(msg.Event), "original") {
					select {
					case out <- msg.Data:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return out, nil
}
