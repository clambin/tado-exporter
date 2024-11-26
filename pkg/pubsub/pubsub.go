// Package pubsub provides a basic Publish/Subscribe implementation.
package pubsub

import (
	"log/slog"
	"sync"
)

// Publisher allows clients to subscribe and sends them the information provided by Publish.
type Publisher[T any] struct {
	clients map[chan T]struct{}
	logger  *slog.Logger
	lock    sync.RWMutex
}

// New returns a new Publisher
func New[T any](logger *slog.Logger) *Publisher[T] {
	return &Publisher[T]{
		clients: make(map[chan T]struct{}),
		logger:  logger,
	}
}

// Subscribe registers the caller and returns a new channel on which it will publish updates.
func (p *Publisher[T]) Subscribe() chan T {
	p.lock.Lock()
	defer p.lock.Unlock()
	ch := make(chan T)
	p.clients[ch] = struct{}{}
	p.logger.Debug("subscriber added", slog.Int("subscribers", len(p.clients)))
	return ch
}

// Unsubscribe removes the registered client/channel.
func (p *Publisher[T]) Unsubscribe(ch chan T) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.clients, ch)
	p.logger.Debug("subscriber removed", slog.Int("subscribers", len(p.clients)))
}

// Publish sends info to all registered clients.
func (p *Publisher[T]) Publish(info T) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for ch := range p.clients {
		ch <- info
	}
}

// Subscribers returns the current number of subscribers
func (p *Publisher[T]) Subscribers() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.clients)
}
