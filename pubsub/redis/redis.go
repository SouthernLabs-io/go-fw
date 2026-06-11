package redis

import (
	"context"
	stderrors "errors"
	"io"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/pubsub"
)

// broker implements pubsub.Broker using Redis Pub/Sub.
type broker struct {
	rdb    redis.UniversalClient
	pubsub *redis.PubSub

	subsByChannel map[string]map[*subscription]struct{}
	allSubs       map[*subscription]struct{}

	mu       sync.RWMutex
	closed   bool
	closeErr error
	netMu    sync.Mutex
}

// NewBroker creates a new Redis pubsub broker.
func NewBroker(rdb redis.UniversalClient) pubsub.Broker {
	b := &broker{
		rdb:           rdb,
		pubsub:        rdb.Subscribe(context.Background()),
		subsByChannel: make(map[string]map[*subscription]struct{}),
		allSubs:       make(map[*subscription]struct{}),
	}

	go b.multiplex()

	return b
}

func (b *broker) multiplex() {
	for {
		msg, err := b.pubsub.ReceiveMessage(context.Background())
		if err != nil {
			b.mu.RLock()
			closed := b.closed
			b.mu.RUnlock()
			if closed {
				return
			}
			// Transient error or timeout, prevent tight looping if Redis is completely down
			time.Sleep(500 * time.Millisecond)
			continue
		}

		b.mu.RLock()
		if b.closed {
			b.mu.RUnlock()
			return
		}

		var targets []*subscription
		if subs, ok := b.subsByChannel[msg.Channel]; ok {
			targets = make([]*subscription, 0, len(subs))
			for sub := range subs {
				targets = append(targets, sub)
			}
		}
		b.mu.RUnlock()

		pm := pubsub.Message{
			Channel: msg.Channel,
			Payload: []byte(msg.Payload),
		}

		for _, sub := range targets {
			_ = sub.sendNonBlocking(pm)
		}
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
	b.mu.RUnlock()

	err := b.rdb.Publish(ctx, channel, payload).Err()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return errors.NewCanceledf("publish canceled: %w", err)
		}
		return errors.Newf(errors.ErrCodeUnknown, "failed to publish message: %w", err)
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
		defer b.mu.Unlock()
		return b.closeErr
	}
	b.closed = true

	var all []*subscription
	for sub := range b.allSubs {
		all = append(all, sub)
	}

	b.subsByChannel = nil
	b.allSubs = nil
	b.mu.Unlock()

	// Close all subscriptions
	var firstErr error
	for _, sub := range all {
		if err := sub.Close(ctx); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	err := b.pubsub.Close()
	b.mu.Lock()
	if err != nil {
		b.closeErr = errors.Newf(errors.ErrCodeUnknown, "failed to close pubsub: %w", err)
	} else if firstErr != nil {
		b.closeErr = firstErr
	}
	defer b.mu.Unlock()
	return b.closeErr
}

func (b *broker) addChannelToSub(sub *subscription, channel string) error {
	b.netMu.Lock()
	defer b.netMu.Unlock()

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return pubsub.ErrBrokerClosed
	}
	needsSubscribe := b.subsByChannel[channel] == nil
	if needsSubscribe {
		b.subsByChannel[channel] = make(map[*subscription]struct{})
	}
	b.mu.Unlock()

	if needsSubscribe {
		if err := b.pubsub.Subscribe(context.Background(), channel); err != nil {
			b.mu.Lock()
			delete(b.subsByChannel, channel)
			b.mu.Unlock()
			return errors.Newf(errors.ErrCodeUnknown, "failed to subscribe to channel: %w", err)
		}
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return pubsub.ErrBrokerClosed
	}
	b.subsByChannel[channel][sub] = struct{}{}
	b.mu.Unlock()
	return nil
}

func (b *broker) removeChannelFromSub(sub *subscription, channel string) error {
	b.netMu.Lock()
	defer b.netMu.Unlock()

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return pubsub.ErrBrokerClosed
	}

	subs, ok := b.subsByChannel[channel]
	if !ok {
		b.mu.Unlock()
		return nil
	}

	delete(subs, sub)
	needsUnsubscribe := len(subs) == 0
	if needsUnsubscribe {
		delete(b.subsByChannel, channel)
	}
	b.mu.Unlock()

	if needsUnsubscribe {
		if err := b.pubsub.Unsubscribe(context.Background(), channel); err != nil {
			return errors.Newf(errors.ErrCodeUnknown, "failed to unsubscribe from channel: %w", err)
		}
	}
	return nil
}

func (b *broker) removeSub(sub *subscription) error {
	b.netMu.Lock()
	defer b.netMu.Unlock()

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	delete(b.allSubs, sub)

	var channelsToUnsubscribe []string
	for channel, subs := range b.subsByChannel {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(b.subsByChannel, channel)
			channelsToUnsubscribe = append(channelsToUnsubscribe, channel)
		}
	}
	b.mu.Unlock()

	var firstErr error
	for _, channel := range channelsToUnsubscribe {
		if err := b.pubsub.Unsubscribe(context.Background(), channel); err != nil {
			if firstErr == nil {
				firstErr = errors.Newf(errors.ErrCodeUnknown, "failed to unsubscribe from channel: %w", err)
			}
		}
	}
	return firstErr
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
		select {
		case msg := <-s.ch:
			return msg, nil
		default:
			return pubsub.Message{}, errors.Newf(errors.ErrCodeBadState, "subscription closed: %w", io.EOF)
		}
	}
}

func (s *subscription) sendNonBlocking(msg pubsub.Message) error {
	select {
	case <-s.done:
		return io.EOF
	case s.ch <- msg:
		return nil
	default:
		return stderrors.New("subscription channel buffer full, message dropped")
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
	close(s.done)
	s.mu.Unlock()

	return s.broker.removeSub(s)
}
