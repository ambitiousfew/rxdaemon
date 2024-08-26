package intracom

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type Topic[T any] interface {
	Name() string                                                              // Name returns the unique name of the topic.
	PublishChannel() chan<- T                                                  // PublishChannel returns the channel publishers use to send messages to the topic.
	Subscribe(ctx context.Context, conf SubscriberConfig[T]) (<-chan T, error) // Subscribe will attemp to add a consumer group to the topic.
	Unsubscribe(consumer string, ch <-chan T) error                            // Unsubscribe will remove the consumer group from the topic and close the subscriber channel.
	Close() error                                                              // Close will remove all consumer groups from the topic and close all channels.
}

type TopicOption[T any] func(*topic[T])

func WithBroadcaster[T any](b Broadcaster[T]) TopicOption[T] {
	return func(t *topic[T]) {
		t.bc = b
	}
}

type Subscription struct {
	Topic         string
	ConsumerGroup string
	// MaxWaitTimeout is the max time to wait before error due to a topic not existing.
	MaxWaitTimeout time.Duration
}

type TopicConfig struct {
	Name            string // unique name for the topic
	ErrIfExists     bool   // return error if topic already exists
	SubscriberAware bool   // if true, topic broadcaster wont broadcast if there are no subscribers.
}

type topic[T any] struct {
	name     string
	publishC chan T
	requestC chan any
	bc       Broadcaster[T]
	closed   atomic.Bool
	mu       sync.RWMutex
}

func NewTopic[T any](conf TopicConfig, opts ...TopicOption[T]) Topic[T] {
	publishC := make(chan T)
	requestC := make(chan any, 1)

	t := &topic[T]{
		name:     conf.Name,
		publishC: publishC,
		requestC: requestC,
		closed:   atomic.Bool{},
		bc: SyncBroadcaster[T]{
			SubscriberAware: conf.SubscriberAware,
		},
		mu: sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(t)
	}

	// start a broadcaster for this topic
	go t.bc.Broadcast(requestC, publishC)

	return t
}

func (t *topic[T]) Name() string {
	return t.name
}

func (t *topic[T]) PublishChannel() chan<- T {
	return t.publishC
}

func (t *topic[T]) Subscribe(ctx context.Context, conf SubscriberConfig[T]) (<-chan T, error) {
	if t.closed.Load() {
		return nil, errors.New("cannot subscribe, topic already closed")
	}

	responseC := make(chan subscribeResponse[T], 1)
	select {
	case <-ctx.Done():
		return nil, errors.New("subscribe request timed out 1")
	case t.requestC <- subscribeRequest[T]{conf: conf, responseC: responseC}:
	}

	select {
	case <-ctx.Done():
		return nil, errors.New("subscribe response timed out 2")
	case res := <-responseC:
		return res.ch, res.err
	}
}

// Unsubscribe will remove the consumer group from the topic.
func (t *topic[T]) Unsubscribe(consumer string, ch <-chan T) error {
	if t.closed.Load() {
		return errors.New("cannot unsubscribe, topic already closed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	responseC := make(chan unsubscribeResponse, 1)
	select {
	case <-ctx.Done():
		return errors.New("unsubscribe request timed out")
	case t.requestC <- unsubscribeRequest[T]{consumer: consumer, ch: ch, responseC: responseC}:
	}

	select {
	case <-ctx.Done():
		return errors.New("unsubscribe response timed out")
	case resp := <-responseC:
		return resp.err
	}

}

func (t *topic[T]) Close() error {
	if t.closed.Swap(true) {
		return errors.New("topic already closed")
	}

	responseC := make(chan closeResponse, 1)
	t.requestC <- closeRequest{responseC: responseC}
	<-responseC

	// now we can close the request channel
	close(t.requestC)
	close(t.publishC)
	return nil
}
