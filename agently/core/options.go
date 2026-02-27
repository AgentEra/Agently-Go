package core

import (
	"context"
	"fmt"
	"time"

	"github.com/AgentEra/Agently-Go/agently/utils"
)

// PromptSetOptions configures prompt write operations.
// It supports both request-level and agent-level prompt setters.
type PromptSetOptions struct {
	Mappings map[string]any
	Always   bool
}

// PromptSetOption is a functional option for PromptSetOptions.
type PromptSetOption func(*PromptSetOptions)

// WithMappings sets placeholder mappings for prompt value substitution.
func WithMappings(mappings map[string]any) PromptSetOption {
	return func(options *PromptSetOptions) {
		options.Mappings = mappings
	}
}

// WithAlways controls whether agent prompt setters write to the always-on agent prompt.
func WithAlways(always bool) PromptSetOption {
	return func(options *PromptSetOptions) {
		options.Always = always
	}
}

// Always writes to the always-on agent prompt.
func Always() PromptSetOption {
	return WithAlways(true)
}

// RequestOnly writes to the request-local prompt.
func RequestOnly() PromptSetOption {
	return WithAlways(false)
}

func resolvePromptSetOptions(raw ...any) PromptSetOptions {
	options := PromptSetOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case PromptSetOption:
			typed(&options)
		case map[string]any:
			options.Mappings = typed
		case bool:
			options.Always = typed
		case PromptSetOptions:
			options = typed
		case *PromptSetOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported prompt option type: %T", item))
		}
	}
	return options
}

// PromptTextOptions configures prompt text rendering behavior.
type PromptTextOptions struct {
	RoleMapping map[string]string
}

// PromptTextOption is a functional option for PromptTextOptions.
type PromptTextOption func(*PromptTextOptions)

// WithRoleMapping sets role mapping for prompt text rendering.
func WithRoleMapping(mapping map[string]string) PromptTextOption {
	return func(options *PromptTextOptions) {
		options.RoleMapping = mapping
	}
}

// ParsePromptTextOptions parses legacy and functional options for prompt text rendering.
func ParsePromptTextOptions(raw ...any) PromptTextOptions {
	options := PromptTextOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case PromptTextOption:
			typed(&options)
		case map[string]string:
			options.RoleMapping = typed
		case PromptTextOptions:
			options = typed
		case *PromptTextOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported prompt text option type: %T", item))
		}
	}
	return options
}

// PromptMessageOption is a functional option for PromptMessageOptions.
type PromptMessageOption func(*PromptMessageOptions)

// WithPromptRoleMapping sets role mapping for prompt message rendering.
func WithPromptRoleMapping(mapping map[string]string) PromptMessageOption {
	return func(options *PromptMessageOptions) {
		options.RoleMapping = mapping
	}
}

// WithRichContent sets rich-content mode for prompt message rendering.
func WithRichContent(rich bool) PromptMessageOption {
	return func(options *PromptMessageOptions) {
		options.RichContent = rich
	}
}

// RichContent enables rich-content mode for prompt message rendering.
func RichContent() PromptMessageOption {
	return WithRichContent(true)
}

// WithStrictRoleOrders sets strict-role-orders mode for prompt message rendering.
func WithStrictRoleOrders(strict bool) PromptMessageOption {
	return func(options *PromptMessageOptions) {
		options.StrictRoleOrders = strict
	}
}

// StrictRoleOrders enables strict role order merge behavior.
func StrictRoleOrders() PromptMessageOption {
	return WithStrictRoleOrders(true)
}

// ParsePromptMessageOptions parses legacy and functional options for prompt message rendering.
func ParsePromptMessageOptions(raw ...any) PromptMessageOptions {
	options := PromptMessageOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case PromptMessageOption:
			typed(&options)
		case PromptMessageOptions:
			options = typed
		case *PromptMessageOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported prompt message option type: %T", item))
		}
	}
	return options
}

// InheritOptions configures operations with optional inheritance behavior.
type InheritOptions struct {
	Inherit bool
}

// InheritOption is a functional option for InheritOptions.
type InheritOption func(*InheritOptions)

// WithInherit explicitly sets inherit behavior.
func WithInherit(inherit bool) InheritOption {
	return func(options *InheritOptions) {
		options.Inherit = inherit
	}
}

// Inherit enables inherit behavior.
func Inherit() InheritOption {
	return WithInherit(true)
}

// ParseInheritOptions parses legacy and functional inheritance options.
func ParseInheritOptions(raw ...any) InheritOptions {
	options := InheritOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case InheritOption:
			typed(&options)
		case bool:
			options.Inherit = typed
		case InheritOptions:
			options = typed
		case *InheritOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported inherit option type: %T", item))
		}
	}
	return options
}

// StreamOptions configures stream retrieval behavior.
type StreamOptions struct {
	Specific []string
}

// StreamOption is a functional option for StreamOptions.
type StreamOption func(*StreamOptions)

// WithSpecific filters stream events by specific event names.
func WithSpecific(events ...string) StreamOption {
	return func(options *StreamOptions) {
		if len(events) == 0 {
			options.Specific = nil
			return
		}
		options.Specific = append([]string(nil), events...)
	}
}

// ParseStreamOptions parses legacy and functional stream options.
func ParseStreamOptions(raw ...any) StreamOptions {
	options := StreamOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case StreamOption:
			typed(&options)
		case []string:
			options.Specific = append([]string(nil), typed...)
		case StreamOptions:
			options = typed
		case *StreamOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported stream option type: %T", item))
		}
	}
	return options
}

// SplitStreamAndInvokeOptions separates stream-specific options from invoke options.
func SplitStreamAndInvokeOptions(raw ...any) (streamRaw []any, invokeRaw []any) {
	for _, item := range raw {
		switch item.(type) {
		case nil, StreamOption, []string, StreamOptions, *StreamOptions:
			streamRaw = append(streamRaw, item)
		default:
			invokeRaw = append(invokeRaw, item)
		}
	}
	return
}

// InvokeOptions configures context/timeout behavior for high-level API calls.
type InvokeOptions struct {
	Context        context.Context
	Timeout        time.Duration
	DisableTimeout bool
}

// InvokeOption is a functional option for InvokeOptions.
type InvokeOption func(*InvokeOptions)

// WithContext overrides the base context for one invocation.
func WithContext(ctx context.Context) InvokeOption {
	return func(options *InvokeOptions) {
		options.Context = ctx
	}
}

// WithTimeout overrides timeout for one invocation.
func WithTimeout(timeout time.Duration) InvokeOption {
	return func(options *InvokeOptions) {
		options.Timeout = timeout
		options.DisableTimeout = false
	}
}

// NoTimeout disables timeout for one invocation.
func NoTimeout() InvokeOption {
	return func(options *InvokeOptions) {
		options.DisableTimeout = true
		options.Timeout = 0
	}
}

// ParseInvokeOptions parses legacy and functional invoke options.
func ParseInvokeOptions(raw ...any) InvokeOptions {
	options := InvokeOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case InvokeOption:
			typed(&options)
		case context.Context:
			options.Context = typed
		case time.Duration:
			options.Timeout = typed
			options.DisableTimeout = false
		case int:
			options.Timeout = time.Duration(typed) * time.Second
			options.DisableTimeout = false
		case int64:
			options.Timeout = time.Duration(typed) * time.Second
			options.DisableTimeout = false
		case float64:
			options.Timeout = time.Duration(typed * float64(time.Second))
			options.DisableTimeout = false
		case InvokeOptions:
			options = typed
		case *InvokeOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported invoke option type: %T", item))
		}
	}
	return options
}

// BuildInvokeContext creates invocation context with default timeout from settings when needed.
func BuildInvokeContext(settings *utils.Settings, raw ...any) (context.Context, context.CancelFunc) {
	invoke := ParseInvokeOptions(raw...)
	base := invoke.Context
	if base == nil {
		base = context.Background()
	}
	if invoke.DisableTimeout {
		return base, func() {}
	}

	timeout := invoke.Timeout
	if timeout <= 0 {
		timeout = defaultInvokeTimeout(settings)
	}
	if timeout <= 0 {
		return base, func() {}
	}
	ctx, cancel := context.WithTimeout(base, timeout)
	return ctx, cancel
}

func defaultInvokeTimeout(settings *utils.Settings) time.Duration {
	if settings == nil {
		return 0
	}
	raw := settings.Get("runtime.default_timeout_seconds", 0, true)
	switch typed := raw.(type) {
	case int:
		if typed > 0 {
			return time.Duration(typed) * time.Second
		}
	case int64:
		if typed > 0 {
			return time.Duration(typed) * time.Second
		}
	case float64:
		if typed > 0 {
			return time.Duration(typed * float64(time.Second))
		}
	case string:
		// Allow values like "120s" for explicit settings.
		if d, err := time.ParseDuration(typed); err == nil && d > 0 {
			return d
		}
	}
	return 0
}

// SettingsSetOptions configures settings write operations.
type SettingsSetOptions struct {
	AutoLoadEnv bool
}

// SettingsSetOption is a functional option for SettingsSetOptions.
type SettingsSetOption func(*SettingsSetOptions)

// WithAutoLoadEnv toggles environment placeholder substitution for settings value.
func WithAutoLoadEnv(auto bool) SettingsSetOption {
	return func(options *SettingsSetOptions) {
		options.AutoLoadEnv = auto
	}
}

// AutoLoadEnv enables environment placeholder substitution.
func AutoLoadEnv() SettingsSetOption {
	return WithAutoLoadEnv(true)
}

// ParseSettingsSetOptions parses both legacy and functional option forms.
func ParseSettingsSetOptions(raw ...any) SettingsSetOptions {
	options := SettingsSetOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case SettingsSetOption:
			typed(&options)
		case bool:
			options.AutoLoadEnv = typed
		case SettingsSetOptions:
			options = typed
		case *SettingsSetOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported settings option type: %T", item))
		}
	}
	return options
}

// RequestCreateOptions configures BaseAgent.CreateRequest behavior.
type RequestCreateOptions struct {
	InheritAgentPrompt       bool
	InheritExtensionHandlers bool
}

// RequestCreateOption is a functional option for RequestCreateOptions.
type RequestCreateOption func(*RequestCreateOptions)

func WithInheritAgentPrompt(inherit bool) RequestCreateOption {
	return func(options *RequestCreateOptions) {
		options.InheritAgentPrompt = inherit
	}
}

func InheritAgentPrompt() RequestCreateOption {
	return WithInheritAgentPrompt(true)
}

func WithInheritExtensionHandlers(inherit bool) RequestCreateOption {
	return func(options *RequestCreateOptions) {
		options.InheritExtensionHandlers = inherit
	}
}

func InheritExtensionHandlers() RequestCreateOption {
	return WithInheritExtensionHandlers(true)
}

// ParseRequestCreateOptions parses both legacy bool args and functional options.
func ParseRequestCreateOptions(raw ...any) RequestCreateOptions {
	options := RequestCreateOptions{}
	legacyBoolIndex := 0
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case RequestCreateOption:
			typed(&options)
		case bool:
			if legacyBoolIndex == 0 {
				options.InheritAgentPrompt = typed
			} else {
				options.InheritExtensionHandlers = typed
			}
			legacyBoolIndex++
		case RequestCreateOptions:
			options = typed
		case *RequestCreateOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported request-create option type: %T", item))
		}
	}
	return options
}
