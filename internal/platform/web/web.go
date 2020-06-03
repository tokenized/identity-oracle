package web

import (
	"context"
	"net/http"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/dimfeld/httptreemux"
	"go.opencensus.io/plugin/ochttp/propagation/tracecontext"
	"go.opencensus.io/trace"
)

// A Handler is a type that handles a HTTP request within our own little mini
// framework.
type Handler func(ctx context.Context, log logger.Logger, w http.ResponseWriter, r *http.Request,
	params map[string]string) error

// App is the entrypoint into our application and what configures our context
// object for each of our http handlers. Feel free to add any configuration
// data/logic on this App struct
type App struct {
	*httptreemux.TreeMux
	config *Config
	log    logger.Logger
	mw     []Middleware
}

// Web configuration
type Config struct {
	RootURL string
	Net     bitcoin.Network
	IsTest  bool

	// The maximum number of addresses that will be reserved without being touched.
	ReserveMax int

	// Identification of oracle operator
	Entity actions.EntityField
}

// New creates an App value that handle a set of routes for the application.
func New(config *Config, log logger.Logger, mw ...Middleware) *App {
	return &App{
		TreeMux: httptreemux.New(),
		config:  config,
		log:     log,
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

		// This API is using pointer semantic methods on this empty
		// struct type :( This is causing the need to declare this
		// variable here at the top.
		var hf tracecontext.HTTPFormat

		// Check the request for an existing Trace. The WithSpanContext
		// function can unmarshal any existing context or create a new one.
		var ctx context.Context
		var span *trace.Span
		if sc, ok := hf.SpanContextFromRequest(r); ok {
			ctx, span = trace.StartSpanWithRemoteParent(r.Context(), "internal.platform.web", sc)
		} else {
			ctx, span = trace.StartSpan(r.Context(), "internal.platform.web")
		}

		// Add network and test values
		ctx = ContextWithValues(ctx, a.config.Net, a.config.IsTest)

		// Set the context with the required values to
		// process the request.
		v := Values{
			TraceID: span.SpanContext().TraceID.String(),
			Now:     time.Now(),
		}
		ctx = context.WithValue(ctx, KeyValues, &v)

		// Set the parent span on the outgoing requests before any other header to
		// ensure that the trace is ALWAYS added to the request regardless of
		// any error occuring or not.
		hf.SpanContextToRequest(span.SpanContext(), r)

		// Call the wrapped handler functions.
		if err := handler(ctx, a.log, w, r, params); err != nil {
			Error(ctx, a.log, w, err)
		}
	}

	// Add this handler for the specified verb and route.
	a.TreeMux.Handle(verb, path, h)
}

// Register a global options handler
func (a *App) HandleOptions(handler httptreemux.HandlerFunc) {
	a.TreeMux.OptionsHandler = handler
}
