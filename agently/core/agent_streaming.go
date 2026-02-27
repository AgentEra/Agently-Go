package core

import (
	"context"
	"fmt"
)

func (a *BaseAgent) StreamingPrintWithContext(ctx context.Context) error {
	stream, err := a.GetGeneratorWithContext(ctx, "delta")
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

func (a *BaseAgent) StreamingPrint(options ...any) error {
	stream, err := a.GetGenerator(append([]any{"delta"}, options...)...)
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
