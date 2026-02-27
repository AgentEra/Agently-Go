package core

import (
	"context"

	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type PluginType string

const (
	PluginTypePromptGenerator PluginType = "PromptGenerator"
	PluginTypeModelRequester  PluginType = "ModelRequester"
	PluginTypeResponseParser  PluginType = "ResponseParser"
	PluginTypeToolManager     PluginType = "ToolManager"
)

type PromptMessageOptions struct {
	RoleMapping      map[string]string
	RichContent      bool
	StrictRoleOrders bool
}

type PromptGenerator interface {
	ToPromptObject() types.PromptObject
	ToText(roleMapping map[string]string) (string, error)
	ToMessages(options PromptMessageOptions) ([]map[string]any, error)
	ToOutputModelSchema() (any, error)
	ToSerializablePromptData(inherit bool) (map[string]any, error)
	ToJSONPrompt(inherit bool) (string, error)
	ToYAMLPrompt(inherit bool) (string, error)
}

type PromptGeneratorCreator func(prompt *Prompt, settings *utils.Settings) PromptGenerator

type ModelRequester interface {
	GenerateRequestData() (types.RequestData, error)
	RequestModel(ctx context.Context, requestData types.RequestData) (<-chan types.ResponseMessage, error)
	BroadcastResponse(ctx context.Context, source <-chan types.ResponseMessage) (<-chan types.ResponseMessage, error)
}

type ModelRequesterCreator func(prompt *Prompt, settings *utils.Settings) ModelRequester

type ResponseParser interface {
	GetMeta(ctx context.Context) (map[string]any, error)
	GetData(ctx context.Context, dataType string) (any, error)
	GetDataObject(ctx context.Context) (any, error)
	GetText(ctx context.Context) (string, error)
	GetStream(ctx context.Context, streamType string, options ...any) (<-chan any, error)
}

type ResponseParserCreator func(agentName, responseID string, prompt *Prompt, response <-chan types.ResponseMessage, settings *utils.Settings) ResponseParser

type ToolManager interface {
	Register(info types.ToolInfo, fn any) error
	Tag(toolNames []string, tags []string) error
	GetToolInfo(tags []string) map[string]types.ToolInfo
	GetToolList(tags []string) []types.ToolInfo
	GetToolFunc(name string) (any, bool)
	CallTool(ctx context.Context, name string, kwargs map[string]any) (any, error)
}

type ToolManagerCreator func(settings *utils.Settings) ToolManager
