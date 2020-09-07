package broker

import (
	"context"
)

const (
	// Errors
	errBrokerClient = "NATS client error"
	errBrokerServer = "NATS server error"
)

// MessageBroker is a pub/sub NATS Client
type MessageBroker interface {
	// Publish to a certain topic
	Pub(topic string, message interface{}) error

	// Subscribe to a topic, context for cancellation
	// provide a channel to receive messages and errorCh to receive errors
	Sub(ctx context.Context, topic string, receive chan []byte, errCh chan error)
}
