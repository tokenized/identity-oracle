package web

import (
	"context"
	"net/http"
	"time"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/dimfeld/httptreemux"
)

// A Handler is a type that handles a HTTP request within our own little mini
// framework.
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request,
	params map[string]string) error

// App is the entrypoint into our application and what configures our context
// object for each of our http handlers. Feel free to add any configuration
// data/logic on this App struct
type App struct {
	*httptreemux.TreeMux
	config *Config
	mw     []Middleware
}

// Web configuration
type Config struct {
	RootURL string
	Net     bitcoin.Network
	IsTest  bool
}

// New creates an App value that handle a set of routes for the application.
func New(config *Config, mw ...Middleware) *App {
	return &App{
		TreeMux: httptreemux.New(),
		config:  config,
		mw:      mw,
	}
}

func (a *App) AddMiddleWare(mw ...Middleware) {
	a.mw = append(a.mw, mw...)
}

// Handle is our mechanism for mounting Handlers for a given HTTP verb and path
// pair, this makes for really easy, convenient routing.
func (a *App) Handle(verb, path string, handler Handler, mw ...Middleware) {

	// Wrap up the application-wide first, this will call the first function
	// of each middleware which will return a function of type Handler.
	handler = wrapMiddleware(wrapMiddleware(handler, mw), a.mw)

	// The function to execute for each request.
	h := func(w http.ResponseWriter, r *http.Request, params map[string]string) {

		// Add network and test values
		ctx := ContextWithValues(r.Context(), a.config.Net, a.config.IsTest)

		// Set the context with the required values to
		// process the request.
		v := Values{
			Now: time.Now(),
		}
		ctx = context.WithValue(ctx, KeyValues, &v)

		// Call the wrapped handler functions.
		if err := handler(ctx, w, r, params); err != nil {
			Error(ctx, w, err)
		}
	}

	// Add this handler for the specified verb and route.
	a.TreeMux.Handle(verb, path, h)
}

// Register a global options handler
func (a *App) HandleOptions(handler httptreemux.HandlerFunc) {
	a.TreeMux.OptionsHandler = handler
}
