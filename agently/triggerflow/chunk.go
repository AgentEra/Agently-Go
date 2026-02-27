package triggerflow

type Chunk struct {
	ID      string
	Name    string
	Trigger string
	handler Handler
}

func NewChunk(handler Handler, name string) *Chunk {
	if name == "" {
		name = nextID("chunk")
	}
	if handler == nil {
		handler = func(data *EventData) (any, error) { return data.Value, nil }
	}
	id := nextID("chunk")
	return &Chunk{ID: id, Name: name, Trigger: "Chunk[" + name + "]-" + id, handler: handler}
}

func (c *Chunk) Call(data *EventData) (any, error) {
	result, err := c.handler(data)
	if err != nil {
		return nil, err
	}
	_ = data.EmitWithMarks(c.Trigger, result, data.layerMarksCopy(), TriggerTypeEvent)
	return result, nil
}
