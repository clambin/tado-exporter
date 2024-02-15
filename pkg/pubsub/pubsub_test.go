package pubsub_test

import (
	"github.com/clambin/tado-exporter/pkg/pubsub"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"sync"
	"testing"
)

func TestPublisher(t *testing.T) {
	p := pubsub.New[int](slog.Default())

	const clients = 10
	var chs []chan int
	for range clients {
		chs = append(chs, p.Subscribe())
	}

	go p.Publish(123)

	var wg sync.WaitGroup
	wg.Add(len(chs))

	for _, ch := range chs {
		go func(ch chan int) {
			defer wg.Done()
			assert.Equal(t, 123, <-ch)

			p.Unsubscribe(ch)
		}(ch)
	}

	wg.Wait()
}
