package core

import (
	"context"

	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type RequestPrefixHandler func(context.Context, *Prompt, *utils.Settings) error
type BroadcastPrefixHandler func(context.Context, *types.ModelResult, *utils.Settings) ([]types.ResponseMessage, error)
type BroadcastSuffixHandler func(context.Context, types.ResponseEvent, any, *types.ModelResult, *utils.Settings) ([]types.ResponseMessage, error)
type FinallyHandler func(context.Context, *ModelResponseResult, *utils.Settings) error

type ExtensionHandlers struct {
	RequestPrefixes   []RequestPrefixHandler
	BroadcastPrefixes []BroadcastPrefixHandler
	BroadcastSuffixes map[types.ResponseEvent][]BroadcastSuffixHandler
	FinallyHandlers   []FinallyHandler
}

func NewExtensionHandlers(parent *ExtensionHandlers) *ExtensionHandlers {
	h := &ExtensionHandlers{
		RequestPrefixes:   []RequestPrefixHandler{},
		BroadcastPrefixes: []BroadcastPrefixHandler{},
		BroadcastSuffixes: map[types.ResponseEvent][]BroadcastSuffixHandler{},
		FinallyHandlers:   []FinallyHandler{},
	}
	if parent != nil {
		h.RequestPrefixes = append(h.RequestPrefixes, parent.RequestPrefixes...)
		h.BroadcastPrefixes = append(h.BroadcastPrefixes, parent.BroadcastPrefixes...)
		for event, handlers := range parent.BroadcastSuffixes {
			h.BroadcastSuffixes[event] = append([]BroadcastSuffixHandler{}, handlers...)
		}
		h.FinallyHandlers = append(h.FinallyHandlers, parent.FinallyHandlers...)
	}
	return h
}

func (h *ExtensionHandlers) AppendRequestPrefix(handler RequestPrefixHandler) {
	h.RequestPrefixes = append(h.RequestPrefixes, handler)
}

func (h *ExtensionHandlers) AppendBroadcastPrefix(handler BroadcastPrefixHandler) {
	h.BroadcastPrefixes = append(h.BroadcastPrefixes, handler)
}

func (h *ExtensionHandlers) AppendBroadcastSuffix(event types.ResponseEvent, handler BroadcastSuffixHandler) {
	if _, ok := h.BroadcastSuffixes[event]; !ok {
		h.BroadcastSuffixes[event] = []BroadcastSuffixHandler{}
	}
	h.BroadcastSuffixes[event] = append(h.BroadcastSuffixes[event], handler)
}

func (h *ExtensionHandlers) AppendFinally(handler FinallyHandler) {
	h.FinallyHandlers = append(h.FinallyHandlers, handler)
}
