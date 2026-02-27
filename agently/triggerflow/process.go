package triggerflow

import (
	"fmt"
	"sync"
)

var emptyMarker = &struct{}{}

type Process struct {
	flowChunk    func(any) *Chunk
	triggerEvent string
	triggerType  TriggerType
	bluePrint    *BluePrint
	blockData    *BlockData
	options      map[string]any
}

func NewProcess(flowChunk func(any) *Chunk, triggerEvent string, triggerType TriggerType, bluePrint *BluePrint, blockData *BlockData, options map[string]any) *Process {
	if options == nil {
		options = map[string]any{}
	}
	if blockData == nil {
		blockData = NewBlockData(nil, map[string]any{})
	}
	return &Process{flowChunk: flowChunk, triggerEvent: triggerEvent, triggerType: triggerType, bluePrint: bluePrint, blockData: blockData, options: options}
}

func (p *Process) clone(triggerEvent string, triggerType TriggerType, blockData *BlockData) *Process {
	if blockData == nil {
		blockData = p.blockData
	}
	return NewProcess(p.flowChunk, triggerEvent, triggerType, p.bluePrint, blockData, p.options)
}

// When registers signal conditions. Trigger can be string/*Chunk or map[TriggerType][]string.
func (p *Process) When(trigger any, mode string) *Process {
	if mode == "" {
		mode = "and"
	}
	switch typed := trigger.(type) {
	case *Chunk:
		return p.clone(typed.Trigger, TriggerTypeEvent, NewBlockData(nil, map[string]any{}))
	case string:
		if chunk, ok := p.bluePrint.Chunks[typed]; ok {
			return p.clone(chunk.Trigger, TriggerTypeEvent, NewBlockData(nil, map[string]any{}))
		}
		return p.clone(typed, TriggerTypeEvent, NewBlockData(nil, map[string]any{}))
	case map[TriggerType][]string:
		values := map[TriggerType]map[string]any{}
		triggerCount := 0
		currentType := TriggerTypeEvent
		currentEvent := ""
		for t, events := range typed {
			if _, ok := values[t]; !ok {
				values[t] = map[string]any{}
			}
			for _, event := range events {
				if t == TriggerTypeEvent {
					if chunk, ok := p.bluePrint.Chunks[event]; ok {
						event = chunk.Trigger
					}
				}
				values[t][event] = emptyMarker
				currentType = t
				currentEvent = event
				triggerCount++
			}
		}

		if triggerCount == 1 {
			return p.clone(currentEvent, currentType, NewBlockData(nil, map[string]any{}))
		}

		whenTrigger := nextID("when")
		waitTrigger := func(data *EventData) (any, error) {
			switch mode {
			case "or", "simple_or":
				value := data.Value
				if mode == "or" {
					value = map[string]any{"trigger_type": data.TriggerType, "trigger_event": data.TriggerEvent, "value": data.Value}
				}
				return nil, data.EmitWithMarks(whenTrigger, value, data.layerMarksCopy(), TriggerTypeEvent)
			default:
				if _, ok := values[data.TriggerType][data.TriggerEvent]; ok {
					values[data.TriggerType][data.TriggerEvent] = data.Value
				}
				for _, triggerEventMap := range values {
					for _, v := range triggerEventMap {
						if v == emptyMarker {
							return nil, nil
						}
					}
				}
				return nil, data.EmitWithMarks(whenTrigger, values, data.layerMarksCopy(), TriggerTypeEvent)
			}
		}

		for t, eventMap := range values {
			for event := range eventMap {
				p.bluePrint.AddHandler(t, event, waitTrigger, "")
			}
		}

		return p.clone(whenTrigger, TriggerTypeEvent, NewBlockData(nil, map[string]any{}))
	default:
		return p
	}
}

func (p *Process) To(target any, options ...any) *Process {
	config := parseToOptions(options...)
	var chunk *Chunk
	switch typed := target.(type) {
	case string:
		if found, ok := p.bluePrint.Chunks[typed]; ok {
			chunk = found
		}
	case *Chunk:
		chunk = typed
	case Handler:
		chunk = p.flowChunk(NamedChunk{Name: config.Name, Handler: typed})
	case NamedChunk:
		chunk = p.flowChunk(typed)
	}
	if chunk == nil {
		return p
	}
	p.bluePrint.AddHandler(p.triggerType, p.triggerEvent, chunk.Call, "")
	if config.SideBranch {
		return p.clone(p.triggerEvent, p.triggerType, p.blockData)
	}
	return p.clone(chunk.Trigger, p.triggerType, p.blockData)
}

func (p *Process) SideBranch(target any, name string) *Process {
	return p.To(target, ToSideBranch(), WithToName(name))
}

func (p *Process) Batch(chunks []any, options ...any) *Process {
	config := parseBatchOptions(options...)
	batchTrigger := nextID("batch")
	results := map[string]any{}
	triggersToWait := map[string]bool{}
	triggerToChunkName := map[string]string{}
	stateMu := sync.Mutex{}
	var semaphore chan struct{}
	if config.Concurrency > 0 {
		semaphore = make(chan struct{}, config.Concurrency)
	}

	waitAllChunks := func(data *EventData) (any, error) {
		stateMu.Lock()
		if _, ok := triggersToWait[data.Event]; ok {
			results[triggerToChunkName[data.Event]] = data.Value
			triggersToWait[data.Event] = true
		}
		allDone := true
		for _, done := range triggersToWait {
			if !done {
				allDone = false
				break
			}
		}
		snapshot := map[string]any{}
		if allDone {
			for k, v := range results {
				snapshot[k] = v
			}
		}
		stateMu.Unlock()
		if !allDone {
			return nil, nil
		}
		return nil, data.EmitWithMarks(batchTrigger, snapshot, data.layerMarksCopy(), TriggerTypeEvent)
	}

	for _, target := range chunks {
		var chunk *Chunk
		switch typed := target.(type) {
		case *Chunk:
			chunk = typed
		case Handler:
			chunk = p.flowChunk(typed)
		case NamedChunk:
			chunk = p.flowChunk(typed)
		}
		if chunk == nil {
			continue
		}
		triggersToWait[chunk.Trigger] = false
		triggerToChunkName[chunk.Trigger] = chunk.Name
		results[chunk.Name] = nil

		handler := chunk.Call
		if semaphore != nil {
			h := handler
			handler = func(data *EventData) (any, error) {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				return h(data)
			}
		}
		p.bluePrint.AddHandler(p.triggerType, p.triggerEvent, handler, "")
		p.bluePrint.AddEventHandler(chunk.Trigger, waitAllChunks, "")
	}

	if config.SideBranch {
		return p.clone(p.triggerEvent, p.triggerType, p.blockData)
	}
	return p.clone(batchTrigger, TriggerTypeEvent, p.blockData)
}

func (p *Process) Collect(collectionName string, branchID string, mode string) *Process {
	if branchID == "" {
		branchID = nextID("branch")
	}
	if mode == "" {
		mode = "filled_and_update"
	}
	collectTrigger := "Collect-" + collectionName
	mu := sync.Mutex{}
	mu.Lock()
	GlobalBlockData.Set(fmt.Sprintf("collections.%s.%s", collectionName, branchID), emptyMarker)
	mu.Unlock()

	collectBranches := func(data *EventData) (any, error) {
		GlobalBlockData.Set(fmt.Sprintf("collections.%s.%s", collectionName, branchID), data.Value)
		items, _ := GlobalBlockData.Get(fmt.Sprintf("collections.%s", collectionName), map[string]any{}, true).(map[string]any)
		for _, value := range items {
			if value == emptyMarker {
				return nil, nil
			}
		}
		if err := data.EmitWithMarks(collectTrigger, items, data.layerMarksCopy(), TriggerTypeEvent); err != nil {
			return nil, err
		}
		if mode == "filled_then_empty" {
			GlobalBlockData.Delete(fmt.Sprintf("collections.%s", collectionName))
		}
		return nil, nil
	}

	p.To(Handler(collectBranches), WithToName("collect"))
	return p.clone(collectTrigger, TriggerTypeEvent, p.blockData)
}

func (p *Process) End() *Process {
	setDefaultResult := func(data *EventData) (any, error) {
		data.execution.mu.RLock()
		resultSet := data.execution.resultSet
		data.execution.mu.RUnlock()
		if !resultSet {
			data.execution.SetResult(data.Value)
		}
		return nil, nil
	}
	return p.To(Handler(setDefaultResult), WithToName("end"))
}

func (p *Process) ForEach(concurrency int) *Process {
	forEachID := nextID("foreach")
	forEachBlock := NewBlockData(p.blockData, map[string]any{"for_each_id": forEachID})
	sendItemTrigger := fmt.Sprintf("ForEach-%s-Send", forEachID)
	var semaphore chan struct{}
	if concurrency > 0 {
		semaphore = make(chan struct{}, concurrency)
	}

	sendItems := func(data *EventData) (any, error) {
		data.LayerIn()
		forEachInstanceID := data.LayerMark()
		prepare := func(item any) (string, []string, any) {
			data.LayerIn()
			itemID := data.LayerMark()
			marks := data.layerMarksCopy()
			data.execution.systemRuntimeData.Set(fmt.Sprintf("for_each_results.%s.%s", forEachInstanceID, itemID), emptyMarker)
			data.LayerOut()
			return itemID, marks, item
		}
		emitItem := func(item any, marks []string) error {
			if semaphore != nil {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
			}
			return data.EmitWithMarks(sendItemTrigger, item, marks, TriggerTypeEvent)
		}

		if list, ok := data.Value.([]any); ok {
			for _, item := range list {
				_, marks, itemValue := prepare(item)
				if err := emitItem(itemValue, marks); err != nil {
					return nil, err
				}
			}
		} else {
			_, marks, itemValue := prepare(data.Value)
			if err := emitItem(itemValue, marks); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}

	p.To(Handler(sendItems), WithToName("for_each_send"))
	return p.clone(sendItemTrigger, TriggerTypeEvent, forEachBlock)
}

func (p *Process) EndForEach() *Process {
	forEachID := fmt.Sprint(p.blockData.Data.Get("for_each_id", "", true))
	if forEachID == "" {
		return p
	}
	endTrigger := fmt.Sprintf("ForEach-%s-End", forEachID)

	collectResults := func(data *EventData) (any, error) {
		forEachInstanceID := data.UpperLayerMark()
		itemID := data.LayerMark()
		resultsNS := data.execution.systemRuntimeData.Namespace("for_each_results")
		if !resultsNS.Has(forEachInstanceID, true) {
			return nil, nil
		}
		if !resultsNS.Has(fmt.Sprintf("%s.%s", forEachInstanceID, itemID), true) {
			return nil, nil
		}
		resultsNS.Set(fmt.Sprintf("%s.%s", forEachInstanceID, itemID), data.Value)
		values, _ := resultsNS.Get(forEachInstanceID, map[string]any{}, true).(map[string]any)
		for _, value := range values {
			if value == emptyMarker {
				return nil, nil
			}
		}
		data.LayerOut()
		data.LayerOut()
		ordered := make([]any, 0, len(values))
		for _, value := range values {
			ordered = append(ordered, value)
		}
		if err := data.EmitWithMarks(endTrigger, ordered, data.layerMarksCopy(), TriggerTypeEvent); err != nil {
			return nil, err
		}
		resultsNS.Delete(forEachInstanceID)
		return nil, nil
	}

	p.To(Handler(collectResults), WithToName("for_each_collect"))
	block := p.blockData.Outer
	if block == nil {
		block = NewBlockData(nil, map[string]any{})
	}
	return p.clone(endTrigger, TriggerTypeEvent, block)
}

func (p *Process) Match(mode string) *Process {
	if mode == "" {
		mode = "hit_first"
	}
	matchID := nextID("match")
	matchBlock := NewBlockData(p.blockData, map[string]any{})
	matchBlock.Data.Update(map[string]any{
		"match_id":      matchID,
		"cases":         map[string]any{},
		"branch_ends":   []any{},
		"is_first_case": true,
		"has_else":      false,
	})

	matchCase := func(data *EventData) (any, error) {
		data.LayerIn()
		matchedCount := 0
		cases, _ := matchBlock.Data.Get("cases", map[string]any{}, true).(map[string]any)
		for caseID, rawCondition := range cases {
			judgement := false
			switch cond := rawCondition.(type) {
			case Condition:
				judgement = cond(data)
			default:
				judgement = cond == data.Value
			}
			if judgement {
				if mode == "hit_first" {
					return nil, data.EmitWithMarks(fmt.Sprintf("Match-%s-Case-%s", matchID, caseID), data.Value, data.layerMarksCopy(), TriggerTypeEvent)
				}
				if mode == "hit_all" {
					data.LayerIn()
					matchedCount++
					data.execution.systemRuntimeData.Set(fmt.Sprintf("match_results.%s.%s", data.UpperLayerMark(), data.LayerMark()), emptyMarker)
					_ = data.EmitWithMarks(fmt.Sprintf("Match-%s-Case-%s", matchID, caseID), data.Value, data.layerMarksCopy(), TriggerTypeEvent)
					data.LayerOut()
				}
			}
		}
		if matchedCount == 0 {
			if matchBlock.Data.Get("has_else", false, true) == true {
				return nil, data.EmitWithMarks(fmt.Sprintf("Match-%s-Else", matchID), data.Value, data.layerMarksCopy(), TriggerTypeEvent)
			}
			return nil, data.EmitWithMarks(fmt.Sprintf("Match-%s-Result", matchID), data.Value, data.layerMarksCopy(), TriggerTypeEvent)
		}
		return nil, nil
	}

	p.To(Handler(matchCase), WithToName("match"))
	return p.clone(p.triggerEvent, p.triggerType, matchBlock)
}

func (p *Process) Case(condition any) *Process {
	matchID := fmt.Sprint(p.blockData.Data.Get("match_id", "", true))
	if matchID == "" {
		return p
	}
	caseID := nextID("case")
	cases, _ := p.blockData.Data.Get("cases", map[string]any{}, true).(map[string]any)
	cases[caseID] = condition
	p.blockData.Data.Set("cases", cases)

	isFirst := p.blockData.Data.Get("is_first_case", true, true) == true
	if isFirst {
		p.blockData.Data.Set("is_first_case", false)
	} else if !stringsHasPrefix(p.triggerEvent, fmt.Sprintf("Match-%s", matchID)) {
		p.blockData.Data.Append("branch_ends", p.triggerEvent)
	}
	return p.clone(fmt.Sprintf("Match-%s-Case-%s", matchID, caseID), TriggerTypeEvent, p.blockData)
}

func (p *Process) CaseElse() *Process {
	matchID := fmt.Sprint(p.blockData.Data.Get("match_id", "", true))
	if matchID == "" {
		return p
	}
	if p.blockData.Data.Get("is_first_case", true, true) == true {
		return p
	}
	p.blockData.Data.Set("has_else", true)
	if !stringsHasPrefix(p.triggerEvent, fmt.Sprintf("Match-%s", matchID)) {
		p.blockData.Data.Append("branch_ends", p.triggerEvent)
	}
	return p.clone(fmt.Sprintf("Match-%s-Else", matchID), TriggerTypeEvent, p.blockData)
}

func (p *Process) EndMatch() *Process {
	matchID := fmt.Sprint(p.blockData.Data.Get("match_id", "", true))
	if matchID == "" {
		return p
	}
	if !stringsHasPrefix(p.triggerEvent, fmt.Sprintf("Match-%s", matchID)) {
		p.blockData.Data.Append("branch_ends", p.triggerEvent)
	}
	branchEndsRaw, _ := p.blockData.Data.Get("branch_ends", []any{}, true).([]any)
	branchEnds := make([]string, 0, len(branchEndsRaw))
	for _, item := range branchEndsRaw {
		branchEnds = append(branchEnds, fmt.Sprint(item))
	}
	collectBranchResult := func(data *EventData) (any, error) {
		matchKey := fmt.Sprintf("match_results.%s", data.UpperLayerMark())
		matchResults, _ := data.execution.systemRuntimeData.Get(matchKey, map[string]any{}, true).(map[string]any)
		if len(matchResults) > 0 {
			if _, ok := matchResults[data.LayerMark()]; ok {
				matchResults[data.LayerMark()] = data.Value
			}
			for _, value := range matchResults {
				if value == emptyMarker {
					data.execution.systemRuntimeData.Set(matchKey, matchResults)
					return nil, nil
				}
			}
			data.LayerOut()
			resultList := make([]any, 0, len(matchResults))
			for _, value := range matchResults {
				resultList = append(resultList, value)
			}
			if err := data.EmitWithMarks(fmt.Sprintf("Match-%s-Result", matchID), resultList, data.layerMarksCopy(), TriggerTypeEvent); err != nil {
				return nil, err
			}
			data.execution.systemRuntimeData.Delete(matchKey)
			return nil, nil
		}
		data.LayerOut()
		return nil, data.EmitWithMarks(fmt.Sprintf("Match-%s-Result", matchID), data.Value, data.layerMarksCopy(), TriggerTypeEvent)
	}

	for _, trigger := range branchEnds {
		p.When(trigger, "").To(Handler(collectBranchResult), WithToName("match_collect"))
	}

	block := p.blockData.Outer
	if block == nil {
		block = NewBlockData(nil, map[string]any{})
	}
	return p.clone(fmt.Sprintf("Match-%s-Result", matchID), TriggerTypeEvent, block)
}

func (p *Process) IfCondition(condition any) *Process   { return p.Match("").Case(condition) }
func (p *Process) ElifCondition(condition any) *Process { return p.Case(condition) }
func (p *Process) ElseCondition() *Process              { return p.CaseElse() }
func (p *Process) EndCondition() *Process               { return p.EndMatch() }

func (p *Process) Separator(logInfo bool, showValue bool, annotations ...string) *Process {
	if !logInfo && !showValue {
		return p
	}
	output := func(data *EventData) (any, error) {
		msg := map[string]any{}
		if len(annotations) > 0 {
			msg["ANNOTATIONS"] = annotations
		}
		if showValue {
			msg["VALUE"] = data.Value
		}
		if logInfo {
			fmt.Println(msg)
		}
		return nil, nil
	}
	return p.SideBranch(Handler(output), "separator")
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
