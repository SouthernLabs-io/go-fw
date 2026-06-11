package pubsub

import (
	"context"

	"github.com/southernlabs-io/go-fw/errors"
)

// ErrBrokerClosed is returned when an operation is attempted on a closed broker.
const ErrCodeBrokerClosed = "BROKER_CLOSED"

var ErrBrokerClosed = errors.Newf(ErrCodeBrokerClosed, "broker was already closed, no new operations can be performed")

// Message represents a data payload received from a specific channel.
type Message struct {
	Channel string
	Payload []byte // Kept low-level; serialization is handled above this layer.
}

// Broker defines the main pubsub operations.
type Broker interface {
	// Subscribe creates a new stream subscription.
	// It can be initialized with zero or more channels.
	Subscribe(channels ...string) Subscription

	// Publish sends a payload to a specific channel. It returns once the message is accepted for delivery.
	// An error is returned if the broker is closed or in a failed state, or any other reason that prevents the message from being accepted.
	Publish(ctx context.Context, channel string, payload []byte) error

	// PublishAsync sends a payload to a specific channel asynchronously, it returns an immediate error if the broker is closed or in a failed state.
	// This is a fire and forget version of Publish, if confirmation of delivery is needed, the caller should use Publish and handle the error accordingly.
	PublishAsync(ctx context.Context, channel string, payload []byte) error

	// Close shuts down the pubsub broker and terminates all active subscriptions
	// It is idempotent and safe to call multiple times. If an error was returned on a previous call to Close, subsequent calls will return the same error.
	// It will block until all active subscriptions are closed or the context is cancelled.
	Close(ctx context.Context) error
}

// Subscription represents an active stream of messages from one or more channels.
type Subscription interface {

	// AddChannel dynamically adds a new channel to this subscription.
	AddChannel(channel string) error

	// RemoveChannel dynamically stops receiving messages for a channel.
	RemoveChannel(channel string) error

	// Recv blocks until a new message is available, the context is cancelled, or the subscription is closed.
	// It returns error io.EOF when the subscription is closed or any other error that prevents receiving messages.
	// It can be retried if the error is due to a transient issue, but if the subscription is closed, it will always return io.EOF.
	Recv(ctx context.Context) (Message, error)

	// Close stops the subscription, unsubscribes from all channels, and cleans up resources.
	// It is idempotent and safe to call multiple times. If an error was returned on a previous call to Close, subsequent calls will return the same error.
	// It will block until the subscription is fully closed or the context is cancelled.
	Close(ctx context.Context) error
}
