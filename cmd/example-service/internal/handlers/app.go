package handlers

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/rumsrami/example-service/internal/db"
	"github.com/rumsrami/example-service/internal/platform/broker"
	"github.com/rumsrami/example-service/internal/platform/web"
	"github.com/rumsrami/example-service/internal/proto"
	"github.com/rumsrami/example-service/internal/rpc"
)

// Mount connects the dots :)
func Mount(build string, db *db.Database, authority, audience string, mb broker.MessageBroker, app *web.App, stOutLogger zerolog.Logger) {
	// Create struct validator
	validate := validator.New()

	// Create new RPC Handler
	chat := rpc.NewChat(app, build, db, stOutLogger, validate, mb)

	app.Mux.Use(middleware.RequestID)
	app.Mux.Use(middleware.RealIP)
	app.Mux.Use(middleware.Recoverer)
	app.Mux.Mount("/_ah/debug", middleware.Profiler())
	app.Mux.Get("/_ah/warmup", warmup)
	app.Mux.Get("/_ah/health", getHealth(build))

	// Handle Websockets
	// Authenticates using a secure cookie
	app.Mux.Group(func(r chi.Router) {
		cors := cors.New(cors.Options{
			AllowOriginFunc:  allowOriginFunc,
			AllowedMethods:   []string{"GET", "OPTIONS", "POST"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "User-Agent"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           600,
		})
		r.Use(cors.Handler)
		r.Handle("/stream", Stream(mb, stOutLogger))
	})

	// Handle RPC calls
	// Authenticates using JWT token and access control
	app.Mux.Group(func(r chi.Router) {
		cors := cors.New(cors.Options{
			AllowOriginFunc:  allowOriginFunc,
			AllowedMethods:   []string{"OPTIONS", "POST"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "User-Agent"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           600,
		})
		r.Use(cors.Handler)
		//Handle rpc calls
		webrpcHandler := proto.NewChatServer(chat)
		r.Handle("/rpc/*", webrpcHandler)
	})
}
