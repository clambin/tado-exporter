package pubsub

import (
	"log/slog"
	"sync"
)

type Publisher[T any] struct {
	clients map[chan T]struct{}
	logger  *slog.Logger
	lock    sync.Mutex
}

func New[T any](logger *slog.Logger) *Publisher[T] {
	return &Publisher[T]{
		clients: make(map[chan T]struct{}),
		logger:  logger,
	}
}

func (p *Publisher[T]) Subscribe() chan T {
	p.lock.Lock()
	defer p.lock.Unlock()
	ch := make(chan T)
	p.clients[ch] = struct{}{}
	p.logger.Debug("subscriber added", slog.Int("subscribers", len(p.clients)))
	return ch
}

func (p *Publisher[T]) Unsubscribe(ch chan T) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.clients, ch)
	p.logger.Debug("subscriber removed", slog.Int("subscribers", len(p.clients)))
}

func (p *Publisher[T]) Publish(info T) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for ch := range p.clients {
		ch <- info
	}
}
