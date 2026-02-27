package agentextensions

import (
	"context"

	"github.com/AgentEra/Agently-Go/agently/core"
)

type AutoFuncExtension struct {
	agent *core.BaseAgent
}

func NewAutoFuncExtension(agent *core.BaseAgent) *AutoFuncExtension {
	return &AutoFuncExtension{agent: agent}
}

func (e *AutoFuncExtension) AutoFunc(instruction string, outputSchema any) func(context.Context, map[string]any) (any, error) {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := e.agent.CreateTempRequest()
		req.Input(input).Instruct(instruction).Output(outputSchema)
		return req.GetData(ctx, core.GetDataOptions{Type: "parsed"})
	}
}
