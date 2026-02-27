package agentextensions

import (
	"context"
	"fmt"

	"github.com/AgentEra/Agently-Go/agently/core"
)

type StreamingPrintExtension struct {
	agent *core.BaseAgent
}

func NewStreamingPrintExtension(agent *core.BaseAgent) *StreamingPrintExtension {
	return &StreamingPrintExtension{agent: agent}
}

func (e *StreamingPrintExtension) StreamingPrintWithContext(ctx context.Context) error {
	stream, err := e.agent.GetGeneratorWithContext(ctx, "delta")
	if err != nil {
		return err
	}
	fmt.Println()
	for delta := range stream {
		fmt.Print(delta)
	}
	fmt.Println()
	return nil
}

func (e *StreamingPrintExtension) StreamingPrint(options ...any) error {
	stream, err := e.agent.GetGenerator(append([]any{"delta"}, options...)...)
	if err != nil {
		return err
	}
	fmt.Println()
	for delta := range stream {
		fmt.Print(delta)
	}
	fmt.Println()
	return nil
}
