package triggerflow

import (
	"fmt"
	"sync/atomic"

	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

type TriggerType string

const (
	TriggerTypeEvent       TriggerType = "event"
	TriggerTypeFlowData    TriggerType = "flow_data"
	TriggerTypeRuntimeData TriggerType = "runtime_data"
)

type Handler func(*EventData) (any, error)

type Handlers map[string]map[string]Handler

type AllHandlers map[TriggerType]Handlers

var RuntimeStreamStop = types.NewAvoidCopy()

var uidCounter uint64

func nextID(prefix string) string {
	id := atomic.AddUint64(&uidCounter, 1)
	if prefix == "" {
		prefix = "id"
	}
	return fmt.Sprintf("%s-%d", prefix, id)
}

type BlockData struct {
	Outer *BlockData
	Data  *utils.RuntimeData
}

var GlobalBlockData = utils.NewRuntimeData("TriggerFlow-Global-BlockData", map[string]any{}, nil)

func NewBlockData(outer *BlockData, data map[string]any) *BlockData {
	return &BlockData{Outer: outer, Data: utils.NewRuntimeData("TriggerFlow-BlockData", data, nil)}
}

type Condition func(*EventData) bool

type NamedChunk struct {
	Name    string
	Handler Handler
}
