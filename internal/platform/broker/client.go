package broker

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)


// Client implements the MessageBroker interface
type Client struct {
	Conn *nats.EncodedConn
}

// NewClient creates a new NATS message broker connection
func NewClient(natsServiceName string) (*Client, error) {
	// Create a client with default options
	nc, err := nats.Connect(natsServiceName,
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			// handle disconnect event
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			// handle reconnect event
		}))
	if err != nil {
		return nil, errors.Wrapf(err, "%s: cannot create new message broker", errBrokerServer)
	}

	// Nats takes care of marshalling and unmarshalling
	ec, err := nats.NewEncodedConn(nc, nats.DEFAULT_ENCODER)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: cannot create encoded connection from message broker", errBrokerServer)
	}

	// return a NATS client
	return &Client{
		Conn: ec,
	}, nil
}

// Pub Publishes to a topic
func (c *Client) Pub(topic string, message interface{}) error {
	defer func() {
		log.Printf("Successfully published to topic %s\n", topic)
	}()
	err := c.Conn.Publish(topic, message)
	if err != nil {
		return errors.Wrapf(err, "%s: cannot publish to topic: %s", errBrokerClient, topic)
	}
	return nil
}

// Sub subscribes the caller to the broker through a channel that receives
// the messages and passes them to the caller
func (c *Client) Sub(ctx context.Context, topic string, outputCh chan []byte, errCh chan error) {

	// defer closing connection and output channel
	defer func() {
		//_ = c.Conn.Flush()
		//c.Conn.Close()
		close(outputCh)
		log.Printf("returned from topic: %s\n", topic)
	}()

	// Subscribe to nats by binding the channel from the caller to nats server
	sub, err := c.Conn.Subscribe(topic, func(m *nats.Msg) {
		// Forward the message to the caller
		outputCh <- m.Data
	})

	// if there is error subscribing, return from go routine
	if err != nil {
		log.Printf("cannot subscribe to topic: %s\n", topic)
		// pass the error back to the caller
		errCh <- errors.Wrapf(err, "cannot subscribe to topic: %s", topic)
		return
	}

	// select on the context and return when done
	select {
	case <-ctx.Done():
		_ = sub.Unsubscribe()
		log.Printf("streamer signaled unsubscription from topic: %s\n", topic)
		return
	}
}
