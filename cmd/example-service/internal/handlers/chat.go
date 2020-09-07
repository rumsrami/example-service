package handlers

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/rumsrami/example-service/internal/platform/broker"
)

const (
	errInvalidCredentials = "invalid username or role"
)

// a new streamer is created for each ServerSentEvents /stream request
// it connects the embedded nats server with the http handler through
// the nats client (broker) interface
// each request to /stream endpoint spawns a streamer go routine
type streamer struct {
	broker broker.MessageBroker
	logger zerolog.Logger
}

func newStreamer(broker broker.MessageBroker, logger zerolog.Logger) streamer {
	return streamer{
		broker: broker,
		logger: logger,
	}
}

// this could be improved by passing request context to run()
// Then selecting on context.Done then returning
func (s streamer) start(ctx context.Context, topic string, brokerMsgCh chan []byte, brokerErrCh chan error) {
	defer func() {
		s.logger.Info().Msg("Streamer closed")
	}()
	fmt.Println("TOPIC", topic)
	go s.broker.Sub(ctx, topic, brokerMsgCh, brokerErrCh)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msgf("client closed connection on topic: %s", topic)
			<-brokerMsgCh
			return
		}
	}
}
