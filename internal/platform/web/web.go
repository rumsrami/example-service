package web

import (
	"os"
	"syscall"

	"github.com/go-chi/chi"
)

// App is the entry point into our application and what configures our context
// object for each of our http handlers. Feel free to add any configuration
type App struct {
	*chi.Mux
	shutdown chan os.Signal
}

// NewApp creates an App value that handle a set of routes for the application.
func NewApp(shutdown chan os.Signal) *App {
	app := App{
		Mux:      chi.NewRouter(),
		shutdown: shutdown,
	}
	return &app
}

// SignalShutdown is used to gracefully shutdown the app when an integrity
// issue is identified.
func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
}
