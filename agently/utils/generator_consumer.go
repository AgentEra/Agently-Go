package utils

import (
	"context"
	"errors"
	"sync"
)

// GeneratorConsumer replays source stream history to multiple subscribers.
type GeneratorConsumer struct {
	mu          sync.Mutex
	history     []any
	listeners   map[uint64]chan any
	nextID      uint64
	source      <-chan any
	done        chan struct{}
	closed      bool
	started     bool
	startOnce   sync.Once
	sourceErr   error
	closeSignal chan struct{}
}

func NewGeneratorConsumer(source <-chan any) *GeneratorConsumer {
	return &GeneratorConsumer{
		history:     make([]any, 0),
		listeners:   map[uint64]chan any{},
		source:      source,
		done:        make(chan struct{}),
		closeSignal: make(chan struct{}),
	}
}

func (g *GeneratorConsumer) ensureStarted() {
	g.startOnce.Do(func() {
		g.started = true
		go g.consume()
	})
}

func (g *GeneratorConsumer) consume() {
	defer close(g.done)
	for {
		select {
		case <-g.closeSignal:
			g.mu.Lock()
			g.closed = true
			for id, listener := range g.listeners {
				close(listener)
				delete(g.listeners, id)
			}
			g.mu.Unlock()
			return
		case item, ok := <-g.source:
			if !ok {
				g.mu.Lock()
				for id, listener := range g.listeners {
					close(listener)
					delete(g.listeners, id)
				}
				g.mu.Unlock()
				return
			}
			g.mu.Lock()
			g.history = append(g.history, item)
			listeners := make([]chan any, 0, len(g.listeners))
			for _, ch := range g.listeners {
				listeners = append(listeners, ch)
			}
			g.mu.Unlock()
			for _, ch := range listeners {
				select {
				case ch <- item:
				default:
				}
			}
		}
	}
}

func (g *GeneratorConsumer) Subscribe(ctx context.Context) (<-chan any, error) {
	g.ensureStarted()

	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return nil, errors.New("generator consumer closed")
	}
	ch := make(chan any, 64)
	id := g.nextID
	g.nextID++
	history := append([]any(nil), g.history...)
	g.listeners[id] = ch
	g.mu.Unlock()

	go func() {
		for _, item := range history {
			select {
			case ch <- item:
			case <-ctx.Done():
				g.unsubscribe(id)
				return
			}
		}

		select {
		case <-ctx.Done():
			g.unsubscribe(id)
		case <-g.done:
			g.unsubscribe(id)
		}
	}()

	return ch, nil
}

func (g *GeneratorConsumer) unsubscribe(id uint64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if ch, ok := g.listeners[id]; ok {
		close(ch)
		delete(g.listeners, id)
	}
}

func (g *GeneratorConsumer) Result(ctx context.Context) ([]any, error) {
	g.ensureStarted()
	select {
	case <-g.done:
		g.mu.Lock()
		defer g.mu.Unlock()
		out := make([]any, len(g.history))
		copy(out, g.history)
		return out, g.sourceErr
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (g *GeneratorConsumer) Close() {
	select {
	case <-g.closeSignal:
	default:
		close(g.closeSignal)
	}
}
