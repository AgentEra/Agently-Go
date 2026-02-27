package agentextensions

import "fmt"

// ConfigurePromptLoadOptions configures JSON/YAML prompt loading behavior.
type ConfigurePromptLoadOptions struct {
	Mappings      map[string]any
	PromptKeyPath string
}

// ConfigurePromptLoadOption is a functional option for ConfigurePromptLoadOptions.
type ConfigurePromptLoadOption func(*ConfigurePromptLoadOptions)

func WithConfigureMappings(mappings map[string]any) ConfigurePromptLoadOption {
	return func(options *ConfigurePromptLoadOptions) {
		options.Mappings = mappings
	}
}

func WithPromptKeyPath(path string) ConfigurePromptLoadOption {
	return func(options *ConfigurePromptLoadOptions) {
		options.PromptKeyPath = path
	}
}

func parseConfigurePromptLoadOptions(raw ...any) ConfigurePromptLoadOptions {
	options := ConfigurePromptLoadOptions{}
	legacyIndex := 0
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case ConfigurePromptLoadOption:
			typed(&options)
		case map[string]any:
			options.Mappings = typed
		case string:
			if legacyIndex == 0 {
				options.PromptKeyPath = typed
			}
			legacyIndex++
		case ConfigurePromptLoadOptions:
			options = typed
		case *ConfigurePromptLoadOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported configure prompt option type: %T", item))
		}
	}
	return options
}

// KeyWaiterOptions configures key waiter behavior.
type KeyWaiterOptions struct {
	MustInPrompt bool
}

// KeyWaiterOption is a functional option for KeyWaiterOptions.
type KeyWaiterOption func(*KeyWaiterOptions)

func WithMustInPrompt(must bool) KeyWaiterOption {
	return func(options *KeyWaiterOptions) {
		options.MustInPrompt = must
	}
}

func MustInPrompt() KeyWaiterOption {
	return WithMustInPrompt(true)
}

func parseKeyWaiterOptions(raw ...any) KeyWaiterOptions {
	options := KeyWaiterOptions{}
	for _, item := range raw {
		switch typed := item.(type) {
		case nil:
			continue
		case KeyWaiterOption:
			typed(&options)
		case bool:
			options.MustInPrompt = typed
		case KeyWaiterOptions:
			options = typed
		case *KeyWaiterOptions:
			if typed != nil {
				options = *typed
			}
		default:
			panic(fmt.Sprintf("unsupported key waiter option type: %T", item))
		}
	}
	return options
}

func splitKeyWaiterInvokeOptions(raw ...any) (keyWaiterRaw []any, invokeRaw []any) {
	for _, item := range raw {
		switch item.(type) {
		case nil, KeyWaiterOption, bool, KeyWaiterOptions, *KeyWaiterOptions:
			keyWaiterRaw = append(keyWaiterRaw, item)
		default:
			invokeRaw = append(invokeRaw, item)
		}
	}
	return
}
