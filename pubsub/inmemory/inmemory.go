package inmemory

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/pubsub"
)

// broker implements pubsub.Broker using in-memory channels.
type broker struct {
	subsByChannel map[string]map[*subscription]struct{}
	allSubs       map[*subscription]struct{}
	mu            sync.RWMutex
	closed        bool
}

// NewBroker creates a new in-memory pubsub broker.
func NewBroker() pubsub.Broker {
	return &broker{
		subsByChannel: make(map[string]map[*subscription]struct{}),
		allSubs:       make(map[*subscription]struct{}),
	}
}

func (b *broker) Subscribe(channels ...string) pubsub.Subscription {
	sub := &subscription{
		broker: b,
		ch:     make(chan pubsub.Message, 100), // Buffered to handle minor backpressure
		done:   make(chan struct{}),
	}

	b.mu.Lock()
	if !b.closed {
		b.allSubs[sub] = struct{}{}
	} else {
		// If the broker is already closed, close the subscription's done channel immediately.
		close(sub.done)
		sub.closed = true
	}
	b.mu.Unlock()

	for _, ch := range channels {
		_ = sub.AddChannel(ch)
	}

	return sub
}

func (b *broker) Publish(ctx context.Context, channel string, payload []byte) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return pubsub.ErrBrokerClosed
	}

	// Snapshot targets to avoid holding lock during send
	var targets []*subscription
	if subs, ok := b.subsByChannel[channel]; ok {
		targets = make([]*subscription, 0, len(subs))
		for sub := range subs {
			targets = append(targets, sub)
		}
	}
	b.mu.RUnlock()

	msg := pubsub.Message{
		Channel: channel,
		Payload: payload,
	}

	for _, sub := range targets {
		if err := sub.send(ctx, msg); err != nil {
			// If context is canceled, return immediately.
			if errors.Is(err, context.Canceled) {
				return errors.NewCanceledf("publish canceled: %w", err)
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return errors.Newf(errors.ErrCodeUnknown, "publish deadline exceeded: %w", err)
			}
			// If subscription is closed (io.EOF), just ignore and continue
		}
	}

	return nil
}

func (b *broker) PublishAsync(ctx context.Context, channel string, payload []byte) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return pubsub.ErrBrokerClosed
	}
	b.mu.RUnlock()

	// Clone payload to prevent data races if caller modifies it
	payloadCopy := make([]byte, len(payload))
	copy(payloadCopy, payload)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = b.Publish(ctx, channel, payloadCopy)
	}()

	return nil
}

func (b *broker) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true

	var all []*subscription
	for sub := range b.allSubs {
		all = append(all, sub)
	}

	// Help GC
	b.subsByChannel = nil
	b.allSubs = nil
	b.mu.Unlock()

	for _, sub := range all {
		_ = sub.Close(ctx)
	}

	return nil
}

// INTERNAL METHODS for subscription to call

func (b *broker) addChannelToSub(sub *subscription, channel string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return pubsub.ErrBrokerClosed
	}

	if b.subsByChannel[channel] == nil {
		b.subsByChannel[channel] = make(map[*subscription]struct{})
	}
	b.subsByChannel[channel][sub] = struct{}{}
	return nil
}

func (b *broker) removeChannelFromSub(sub *subscription, channel string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return pubsub.ErrBrokerClosed
	}

	if subs, ok := b.subsByChannel[channel]; ok {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(b.subsByChannel, channel)
		}
	}
	return nil
}

func (b *broker) removeSub(sub *subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	delete(b.allSubs, sub)
	for channel, subs := range b.subsByChannel {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(b.subsByChannel, channel)
		}
	}
}

// subscription implements pubsub.Subscription.
type subscription struct {
	broker *broker
	ch     chan pubsub.Message
	done   chan struct{}

	mu     sync.Mutex
	closed bool
}

func (s *subscription) AddChannel(channel string) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errors.Newf(errors.ErrCodeBadState, "subscription closed: %w", io.EOF)
	}
	s.mu.Unlock()

	return s.broker.addChannelToSub(s, channel)
}

func (s *subscription) RemoveChannel(channel string) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errors.Newf(errors.ErrCodeBadState, "subscription closed: %w", io.EOF)
	}
	s.mu.Unlock()

	return s.broker.removeChannelFromSub(s, channel)
}

func (s *subscription) Recv(ctx context.Context) (pubsub.Message, error) {
	select {
	case <-ctx.Done():
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			return pubsub.Message{}, errors.NewCanceledf("recv canceled: %w", err)
		}
		return pubsub.Message{}, errors.Newf(errors.ErrCodeUnknown, "recv deadline exceeded: %w", err)
	case msg := <-s.ch:
		return msg, nil
	case <-s.done:
		// Drain buffer if any messages are left
		select {
		case msg := <-s.ch:
			return msg, nil
		default:
			return pubsub.Message{}, errors.Newf(errors.ErrCodeBadState, "subscription closed: %w", io.EOF)
		}
	}
}

func (s *subscription) send(ctx context.Context, msg pubsub.Message) error {
	select {
	case <-s.done:
		return io.EOF
	case <-ctx.Done():
		return ctx.Err()
	case s.ch <- msg:
		return nil
	}
}

func (s *subscription) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.done) // Broadcast closure to all receivers and senders safely
	s.mu.Unlock()

	s.broker.removeSub(s)
	return nil
}
