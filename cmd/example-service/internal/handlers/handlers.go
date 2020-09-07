package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-contrib/sse"
	"github.com/rs/zerolog"
	"gopkg.in/matryer/respond.v1"

	"github.com/rumsrami/example-service/internal/platform/broker"
)

const (
	chatTopicPrefix = "users.chat."

	// sse authentication error
	sseAuthErr = "websocket error"

	// authentication error
	authErr = "authentication error"

	// sse upgrader error
	sseUpgraderErr = " websocket upgrader error"
)

// warmup broker
func warmup(w http.ResponseWriter, r *http.Request) {
	respond.With(w, r, http.StatusOK, ".")
}

// getHealth validates the service is healthy and ready to accept requests.
func getHealth(build string) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {

		health := struct {
			Version string `json:"version"`
			Status  string `json:"status"`
		}{
			Version: build,
		}

		health.Status = "ok"
		respond.With(w, r, http.StatusOK, health)
	}
	return fn
}

// allowOriginFunc allows origin
func allowOriginFunc(r *http.Request, origin string) bool {
	if origin == "http://localhost:3000" {
		return true
	}
	return false
}

// Stream handles server streams
func Stream(broker broker.MessageBroker, logger zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// extract username and role from url
		email := r.URL.Query().Get("email")
		role := r.URL.Query().Get("role")

		// get request context and wait for it to be cancelled
		ctx := r.Context()

		logger.Info().Msgf("stream handler called by: %s, with the role of: %s\n", email, role)

		// upgrade connection
		f, ok := w.(http.Flusher)
		if !ok {
			logger.Err(errors.New("browser not supported")).Msgf("%v : stream handler cannot upgrade: %s, with the role of: %s", sseUpgraderErr, email, role)
		}

		// add response headers
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// create a new channel to receive messages from the broker
		// this channel will be passed on to the streamer which in turn
		// will pass it to the broker
		brokerMessageChan := make(chan []byte, 512)
		// create an error channel to receive errors from broker
		// this channel will also be passed along from streamer to broker
		brokerErrCh := make(chan error, 100)

		// create a new streamer and bind the message channel
		// message channel will get messages from the broker
		// then messages with be received here in the handler
		// messages are then sent to the browser
		newStreamer := newStreamer(broker, logger)

		topic := fmt.Sprintf("%s%s", chatTopicPrefix, email)

		// run the streamer
		go newStreamer.start(ctx, topic, brokerMessageChan, brokerErrCh)

		defer func() {
			// Done.
			logger.Info().Msgf("stream handler ended by: %s, with the role of: %s", email, role)
			return
		}()

		// wait for messages to come from the broker
		for {
			select {
			// error connecting to the broker
			case err := <-brokerErrCh:
				// streamer or broker should close this channel
				// depending on which one of them had the error
				<-brokerMessageChan
				logger.Info().Msgf("cannot connect to broker: ", err)
				return
			// this listens for messages from broker
			// if messageChan closes unexpectedly for any reason
			// return from the handler and close connection
			case brokerMessage, open := <-brokerMessageChan:
				if !open {
					// If our messageChan was closed, this means that the client has
					// disconnected.
					logger.Info().Msgf("streamer broker channel closed")
					return
				}
				// send the messages to client
				logger.Info().Msgf("SSE: %s", string(brokerMessage))

				_ = sse.Encode(w, sse.Event{
					Event: "message",
					Data:  string(brokerMessage),
				})

				f.Flush()
			}
		}

	}
}