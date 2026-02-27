package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type GetDataOptions struct {
	Type               string
	EnsureKeys         []string
	KeyStyle           string
	MaxRetries         int
	RaiseEnsureFailure bool
	RetryCount         int
}

type ModelResponseResult struct {
	agentName         string
	responseID        string
	prompt            *Prompt
	pluginManager     *PluginManager
	settings          *utils.Settings
	extensionHandlers *ExtensionHandlers
	parser            ResponseParser

	runFinallyOnce sync.Once
}

func (r *ModelResponseResult) runFinally(ctx context.Context) error {
	var finalErr error
	r.runFinallyOnce.Do(func() {
		for _, handler := range r.extensionHandlers.FinallyHandlers {
			if err := handler(ctx, r, r.settings); err != nil {
				finalErr = err
				return
			}
		}
	})
	return finalErr
}

func (r *ModelResponseResult) GetMetaWithContext(ctx context.Context) (map[string]any, error) {
	meta, err := r.parser.GetMeta(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.runFinally(ctx); err != nil {
		return nil, err
	}
	return meta, nil
}

func (r *ModelResponseResult) GetMeta(options ...any) (map[string]any, error) {
	ctx, cancel := BuildInvokeContext(r.settings, options...)
	defer cancel()
	return r.GetMetaWithContext(ctx)
}

func (r *ModelResponseResult) PeekMetaWithContext(ctx context.Context) (map[string]any, error) {
	return r.parser.GetMeta(ctx)
}

func (r *ModelResponseResult) PeekMeta(options ...any) (map[string]any, error) {
	ctx, cancel := BuildInvokeContext(r.settings, options...)
	defer cancel()
	return r.PeekMetaWithContext(ctx)
}

func (r *ModelResponseResult) GetTextWithContext(ctx context.Context) (string, error) {
	text, err := r.parser.GetText(ctx)
	if err != nil {
		return "", err
	}
	if err := r.runFinally(ctx); err != nil {
		return "", err
	}
	return text, nil
}

func (r *ModelResponseResult) GetText(options ...any) (string, error) {
	ctx, cancel := BuildInvokeContext(r.settings, options...)
	defer cancel()
	return r.GetTextWithContext(ctx)
}

func (r *ModelResponseResult) PeekTextWithContext(ctx context.Context) (string, error) {
	return r.parser.GetText(ctx)
}

func (r *ModelResponseResult) PeekText(options ...any) (string, error) {
	ctx, cancel := BuildInvokeContext(r.settings, options...)
	defer cancel()
	return r.PeekTextWithContext(ctx)
}

func (r *ModelResponseResult) GetDataWithContext(ctx context.Context, opts GetDataOptions) (any, error) {
	if opts.Type == "" {
		opts.Type = "parsed"
	}
	if opts.KeyStyle == "" {
		opts.KeyStyle = "dot"
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 3
	}
	if !opts.RaiseEnsureFailure {
		// default true
		opts.RaiseEnsureFailure = true
	}

	data, err := r.parser.GetData(ctx, opts.Type)
	if err != nil {
		return nil, err
	}

	if opts.Type == "parsed" && len(opts.EnsureKeys) > 0 {
		missing := make([]string, 0)
		for _, key := range opts.EnsureKeys {
			marker := &struct{}{}
			v := utils.LocatePathInData(data, key, opts.KeyStyle, marker)
			if v == marker {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			if IsModelLogsEnabled(r.settings) {
				retryText, _ := r.parser.GetText(ctx)
				_ = EmitSystemMessage(r.settings, types.SystemEventModelRequest, map[string]any{
					"agent_name":  r.agentName,
					"response_id": r.responseID,
					"content": map[string]any{
						"stage": "No Target Data in Response, Preparing Retry",
						"detail": fmt.Sprintf(
							"\n[Response]: %s\n[Retried Times]: %d",
							retryText,
							opts.RetryCount,
						),
					},
				})
			}
			if opts.RetryCount < opts.MaxRetries {
				response := NewModelResponse(r.agentName, r.pluginManager, r.settings, r.prompt, r.extensionHandlers)
				return response.Result.GetDataWithContext(ctx, GetDataOptions{
					Type:               opts.Type,
					EnsureKeys:         opts.EnsureKeys,
					KeyStyle:           opts.KeyStyle,
					MaxRetries:         opts.MaxRetries,
					RaiseEnsureFailure: opts.RaiseEnsureFailure,
					RetryCount:         opts.RetryCount + 1,
				})
			}
			if opts.RaiseEnsureFailure {
				if err := r.runFinally(ctx); err != nil {
					return nil, err
				}
				return nil, fmt.Errorf("ensure_keys %v missing after %d retries", missing, opts.MaxRetries)
			}
		}
	}

	if err := r.runFinally(ctx); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *ModelResponseResult) GetData(args ...any) (any, error) {
	opts, invokeRaw := parseGetDataCallArgs("GetData", args...)
	ctx, cancel := BuildInvokeContext(r.settings, invokeRaw...)
	defer cancel()
	return r.GetDataWithContext(ctx, opts)
}

func (r *ModelResponseResult) PeekDataWithContext(ctx context.Context, dataType string) (any, error) {
	if dataType == "" {
		dataType = "parsed"
	}
	return r.parser.GetData(ctx, dataType)
}

func (r *ModelResponseResult) PeekData(dataType string, options ...any) (any, error) {
	ctx, cancel := BuildInvokeContext(r.settings, options...)
	defer cancel()
	return r.PeekDataWithContext(ctx, dataType)
}

func (r *ModelResponseResult) GetDataObjectWithContext(ctx context.Context, opts GetDataOptions) (any, error) {
	if len(opts.EnsureKeys) > 0 {
		if _, err := r.GetDataWithContext(ctx, opts); err != nil {
			return nil, err
		}
	}
	obj, err := r.parser.GetDataObject(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.runFinally(ctx); err != nil {
		return nil, err
	}
	return obj, nil
}

func (r *ModelResponseResult) GetDataObject(args ...any) (any, error) {
	opts, invokeRaw := parseGetDataCallArgs("GetDataObject", args...)
	ctx, cancel := BuildInvokeContext(r.settings, invokeRaw...)
	defer cancel()
	return r.GetDataObjectWithContext(ctx, opts)
}

func (r *ModelResponseResult) Prompt() *Prompt {
	return r.prompt
}

func (r *ModelResponseResult) GetGeneratorWithContext(ctx context.Context, streamType string, options ...any) (<-chan any, error) {
	if streamType == "" {
		streamType = "delta"
	}
	config := ParseStreamOptions(options...)
	ch, err := r.parser.GetStream(ctx, streamType, config.Specific)
	if err != nil {
		return nil, err
	}
	out := make(chan any)
	go func() {
		defer close(out)
		for {
			select {
			case item, ok := <-ch:
				if !ok {
					_ = r.runFinally(ctx)
					return
				}
				select {
				case out <- item:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func (r *ModelResponseResult) GetGenerator(args ...any) (<-chan any, error) {
	streamType, streamRaw, invokeRaw := parseGeneratorCallArgs("GetGenerator", args...)
	ctx, cancel := BuildInvokeContext(r.settings, invokeRaw...)
	ch, err := r.GetGeneratorWithContext(ctx, streamType, streamRaw...)
	if err != nil {
		cancel()
		return nil, err
	}
	out := make(chan any)
	go func() {
		defer close(out)
		defer cancel()
		for item := range ch {
			out <- item
		}
	}()
	return out, nil
}

type ModelResponse struct {
	AgentName         string
	ID                string
	pluginManager     *PluginManager
	settings          *utils.Settings
	prompt            *Prompt
	extensionHandlers *ExtensionHandlers
	Result            *ModelResponseResult
}

func NewModelResponse(agentName string, pluginManager *PluginManager, settings *utils.Settings, prompt *Prompt, extensionHandlers *ExtensionHandlers) *ModelResponse {
	if agentName == "" {
		agentName = "Directly Request"
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	settingsSnapshot, _ := settings.Get("", map[string]any{}, true).(map[string]any)
	settingsCopy := utils.NewSettings("Response-Settings", settingsSnapshot, nil)
	settingsCopy.Set("$log.cancel_logs", false)

	promptSnapshot, _ := prompt.Get("", map[string]any{}, true).(map[string]any)
	promptCopy := NewPrompt(pluginManager, settingsCopy, promptSnapshot, nil, "Response-Prompt")

	handlersCopy := NewExtensionHandlers(extensionHandlers)

	response := &ModelResponse{
		AgentName:         agentName,
		ID:                id,
		pluginManager:     pluginManager,
		settings:          settingsCopy,
		prompt:            promptCopy,
		extensionHandlers: handlersCopy,
	}

	responseStream := response.getResponseGenerator()
	spec, err := pluginManager.GetActivatedPlugin(PluginTypeResponseParser)
	if err != nil {
		// fallback parser that only forwards text
		response.Result = &ModelResponseResult{
			agentName:         agentName,
			responseID:        id,
			prompt:            promptCopy,
			pluginManager:     pluginManager,
			settings:          settingsCopy,
			extensionHandlers: handlersCopy,
			parser:            NewFallbackResponseParser(responseStream),
		}
		return response
	}
	creator, ok := spec.Creator.(ResponseParserCreator)
	if !ok {
		response.Result = &ModelResponseResult{
			agentName:         agentName,
			responseID:        id,
			prompt:            promptCopy,
			pluginManager:     pluginManager,
			settings:          settingsCopy,
			extensionHandlers: handlersCopy,
			parser:            NewFallbackResponseParser(responseStream),
		}
		return response
	}
	parser := creator(agentName, id, promptCopy, responseStream, settingsCopy)
	response.Result = &ModelResponseResult{
		agentName:         agentName,
		responseID:        id,
		prompt:            promptCopy,
		pluginManager:     pluginManager,
		settings:          settingsCopy,
		extensionHandlers: handlersCopy,
		parser:            parser,
	}
	return response
}

func (r *ModelResponse) CancelLogs() {
	r.settings.Set("$log.cancel_logs", true)
}

func (r *ModelResponse) getResponseGenerator() <-chan types.ResponseMessage {
	out := make(chan types.ResponseMessage, 64)
	go func() {
		defer close(out)
		ctx := context.Background()

		spec, err := r.pluginManager.GetActivatedPlugin(PluginTypeModelRequester)
		if err != nil {
			out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
			return
		}
		creator, ok := spec.Creator.(ModelRequesterCreator)
		if !ok {
			out <- types.ResponseMessage{Event: types.ResponseEventError, Data: fmt.Errorf("model requester creator type invalid")}
			return
		}
		for _, prefix := range r.extensionHandlers.RequestPrefixes {
			if err := prefix(ctx, r.prompt, r.settings); err != nil {
				out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
				return
			}
		}

		requester := creator(r.prompt, r.settings)
		requestData, err := requester.GenerateRequestData()
		if err != nil {
			out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
			return
		}
		raw, err := requester.RequestModel(ctx, requestData)
		if err != nil {
			out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
			return
		}
		broadcast, err := requester.BroadcastResponse(ctx, raw)
		if err != nil {
			out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
			return
		}

		fullResult := &types.ModelResult{Meta: map[string]any{}, Extra: map[string]any{}}
		for _, prefix := range r.extensionHandlers.BroadcastPrefixes {
			messages, err := prefix(ctx, fullResult, r.settings)
			if err != nil {
				out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
				continue
			}
			for _, m := range messages {
				out <- m
			}
		}

		for msg := range broadcast {
			out <- msg
			suffixes := r.extensionHandlers.BroadcastSuffixes[msg.Event]
			for _, suffix := range suffixes {
				messages, err := suffix(ctx, msg.Event, msg.Data, fullResult, r.settings)
				if err != nil {
					out <- types.ResponseMessage{Event: types.ResponseEventError, Data: err}
					continue
				}
				for _, m := range messages {
					out <- m
				}
			}
		}
	}()
	return out
}

type ModelRequest struct {
	agentName         string
	pluginManager     *PluginManager
	settings          *utils.Settings
	prompt            *Prompt
	extensionHandlers *ExtensionHandlers
}

func NewModelRequest(pluginManager *PluginManager, agentName string, parentSettings *utils.Settings, parentPrompt *Prompt, parentExtensionHandlers *ExtensionHandlers) *ModelRequest {
	if agentName == "" {
		agentName = "Directly Request"
	}
	if parentSettings == nil {
		parentSettings = NewDefaultSettings(nil)
	}
	settings := utils.NewSettings("Request-Settings", map[string]any{}, parentSettings)
	prompt := NewPrompt(pluginManager, settings, map[string]any{}, parentPrompt, "Request-Prompt")
	handlers := parentExtensionHandlers
	if handlers == nil {
		handlers = NewExtensionHandlers(nil)
	}
	return &ModelRequest{
		agentName:         agentName,
		pluginManager:     pluginManager,
		settings:          settings,
		prompt:            prompt,
		extensionHandlers: handlers,
	}
}

func (r *ModelRequest) Settings() *utils.Settings { return r.settings }

func (r *ModelRequest) Prompt() *Prompt { return r.prompt }

func (r *ModelRequest) ExtensionHandlers() *ExtensionHandlers { return r.extensionHandlers }

func (r *ModelRequest) SetPrompt(key string, value any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set(key, value, config.Mappings)
	return r
}

func (r *ModelRequest) System(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("system", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) Rule(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("system", []any{"{system.rule} ARE IMPORTANT RULES YOU SHALL FOLLOW!"})
	r.prompt.Set("system.rule", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) Role(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("system", []any{"YOU MUST REACT AND RESPOND AS {system.your_role}!"})
	r.prompt.Set("system.your_role", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) UserInfo(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("system", []any{"{system.user_info} IS IMPORTANT INFORMATION ABOUT USER!"})
	r.prompt.Set("system.user_info", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) Input(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("input", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) Info(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("info", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) Instruct(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("instruct", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) Examples(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("examples", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) Output(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("output", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) Attachment(prompt any, options ...any) *ModelRequest {
	config := resolvePromptSetOptions(options...)
	r.prompt.Set("attachment", prompt, config.Mappings)
	return r
}

func (r *ModelRequest) GetResponse() *ModelResponse {
	response := NewModelResponse(r.agentName, r.pluginManager, r.settings, r.prompt, r.extensionHandlers)
	r.prompt.Clear()
	return response
}

func (r *ModelRequest) GetResult() *ModelResponseResult {
	return r.GetResponse().Result
}

func (r *ModelRequest) GetMetaWithContext(ctx context.Context) (map[string]any, error) {
	return r.GetResponse().Result.GetMetaWithContext(ctx)
}

func (r *ModelRequest) GetMeta(options ...any) (map[string]any, error) {
	return r.GetResponse().Result.GetMeta(options...)
}

func (r *ModelRequest) GetTextWithContext(ctx context.Context) (string, error) {
	return r.GetResponse().Result.GetTextWithContext(ctx)
}

func (r *ModelRequest) GetText(options ...any) (string, error) {
	return r.GetResponse().Result.GetText(options...)
}

func (r *ModelRequest) GetDataWithContext(ctx context.Context, opts GetDataOptions) (any, error) {
	return r.GetResponse().Result.GetDataWithContext(ctx, opts)
}

func (r *ModelRequest) GetData(args ...any) (any, error) {
	return r.GetResponse().Result.GetData(args...)
}

func (r *ModelRequest) GetDataObjectWithContext(ctx context.Context, opts GetDataOptions) (any, error) {
	return r.GetResponse().Result.GetDataObjectWithContext(ctx, opts)
}

func (r *ModelRequest) GetDataObject(args ...any) (any, error) {
	return r.GetResponse().Result.GetDataObject(args...)
}

func (r *ModelRequest) GetGeneratorWithContext(ctx context.Context, streamType string, options ...any) (<-chan any, error) {
	return r.GetResponse().Result.GetGeneratorWithContext(ctx, streamType, options...)
}

func (r *ModelRequest) GetGenerator(args ...any) (<-chan any, error) {
	return r.GetResponse().Result.GetGenerator(args...)
}

func parseGetDataCallArgs(method string, args ...any) (GetDataOptions, []any) {
	opts := GetDataOptions{}
	invokeRaw := make([]any, 0)
	if len(args) == 0 {
		return opts, invokeRaw
	}
	index := 0
	if _, ok := args[0].(context.Context); ok {
		invokeRaw = append(invokeRaw, args[0])
		index = 1
	}
	if index < len(args) {
		if typed, ok := args[index].(GetDataOptions); ok {
			opts = typed
			index++
		} else if _, ok := args[index].(InvokeOption); !ok {
			switch args[index].(type) {
			case InvokeOptions, *InvokeOptions, context.Context, time.Duration, int, int64, float64:
				// invoke args without explicit GetDataOptions
			default:
				panic(fmt.Sprintf("%s expects GetDataOptions, got %T", method, args[index]))
			}
		}
	}
	invokeRaw = append(invokeRaw, args[index:]...)
	return opts, invokeRaw
}

func parseGeneratorCallArgs(method string, args ...any) (string, []any, []any) {
	streamType := "delta"
	streamRaw := make([]any, 0)
	invokeRaw := make([]any, 0)
	if len(args) == 0 {
		return streamType, streamRaw, invokeRaw
	}
	index := 0
	if _, ok := args[0].(context.Context); ok {
		invokeRaw = append(invokeRaw, args[0])
		index = 1
	}
	if index < len(args) {
		if typed, ok := args[index].(string); ok {
			if typed != "" {
				streamType = typed
			}
			index++
		} else {
			switch args[index].(type) {
			case StreamOption, []string, StreamOptions, *StreamOptions, InvokeOption, InvokeOptions, *InvokeOptions, context.Context, time.Duration, int, int64, float64:
				// streamType omitted, use default "delta"
			default:
				if len(args) == 1 || index == 0 {
					panic(fmt.Sprintf("%s expects streamType as string, got %T", method, args[index]))
				}
			}
		}
	}
	rawStream, rawInvoke := SplitStreamAndInvokeOptions(args[index:]...)
	streamRaw = append(streamRaw, rawStream...)
	invokeRaw = append(invokeRaw, rawInvoke...)
	return streamType, streamRaw, invokeRaw
}
