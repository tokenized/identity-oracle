package mid

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tokenized/pkg/logger"

	"github.com/google/uuid"
)

const (
	// HeaderXRequestID is the x-request-id HTTP header
	HeaderXRequestID = "X-Request-Id"

	// HeaderXTrace is the x-trace HTTP header
	HeaderXTrace = "X-Trace"

	// HeaderXForwardedFor is a constant for ht x-forwarded-for header
	HeaderXForwardedFor = "X-Forwarded-For"
)

// RequestLoggingMiddleware is our common HTTP request logging middleware.
type RequestLoggingMiddleware struct {
	LogConfig logger.Config
}

// NewRequestLoggingMiddleware returns a new request logging middleware.
func NewRequestLoggingMiddleware(logConfig logger.Config) RequestLoggingMiddleware {
	return RequestLoggingMiddleware{
		LogConfig: logConfig,
	}
}

// Handler processes a http.Request, via a http.Handler.
//
// This function is used to create a chain of middleware that executes before, or after, the
// http.Request is processed.
func (m *RequestLoggingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// create a logger with a request id field.
		ctx := logger.ContextWithLogConfig(r.Context(), m.LogConfig)

		// add the request ID to the context
		traceID := buildTraceID(r.Header)

		// add the request ID to log entries
		ctx = logger.ContextWithLogTrace(ctx, traceID)

		// put the context in the request
		r = r.WithContext(ctx)

		// use a writer that allows us to access the status code.
		lrw := newLoggingResponseWriter(w)

		// handle the request
		next.ServeHTTP(lrw, r)

		// log the result
		logHTTPRequest(start, r, lrw)
	})
}

// logRequest is used for logging a HTTP request/response
//
// The LoggingMiddlware function has set the status code.
func logHTTPRequest(start time.Time,
	r *http.Request,
	lrw *loggingResponseWriter) {

	ctx := r.Context()

	// get elapsed time
	// nanoseconds(billion) 1e9, milliseconds(thousand) 1e3, so divide nanoseconds by 1e6 for
	// milliseconds.
	elapsed := float64(time.Since(start).Nanoseconds()) / 1e6

	// decorate the logger with http/stats friendly fields
	fields := []logger.Field{
		logger.Formatter("elapsed", "%06f", elapsed), // use %06f so it is fixed width
		logger.Int("status", lrw.statusCode),
		logger.String("method", r.Method),
		logger.String("path", r.URL.Path),
		logger.String("type", "http"),
		logger.String("remote", getRemoteAddress(ctx, r)),
	}

	// params, if any
	if len(r.URL.RawQuery) > 0 {
		fields = append(fields, logger.String("params", r.URL.RawQuery))
	}

	username, _, ok := r.BasicAuth()
	if ok && lrw.statusCode == http.StatusUnauthorized {
		// Log the login attempt.
		//
		// This is used for alerts and auditing.
		logger.WarnWithFields(ctx, fields, `{"reason":"login_failed","username":%s}`,
			strconv.Quote(username))

		return
	}

	logger.InfoWithFields(ctx, fields, "")
}

// loggingResponseWriter captures the HTTP status code so we can access it in middleware.
//
// See https://ndersson.me/post/capturing_status_code_in_net_http/
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	// default to 404 so if not routes are matched, we have a sane status code
	return &loggingResponseWriter{w, http.StatusNotFound}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code

	lrw.ResponseWriter.WriteHeader(code)
}

// getRemoteAddress returns the remote address for the HTTP Request.
func getRemoteAddress(ctx context.Context, r *http.Request) string {
	// Get the remote address.
	//
	// Try the header first, falling back to the address in the request.
	addr := r.Header.Get(HeaderXForwardedFor)

	if len(addr) > 0 {
		// Use the address from the header.
		//
		// The header can take the following formats
		//
		// - "1.2.3.42, 1.2.3.4"
		// - "1.2.3.42"
		//
		// Where the left-most address is the client address, with each hop adding their address to
		// the list.
		return strings.Split(strings.Replace(addr, " ", "", -1), ",")[0]
	}

	return r.RemoteAddr
}

// buildTraceID returns a trace ID from a header if provided, otherwise a new ID is returned.
func buildTraceID(h http.Header) string {
	t := h.Get(HeaderXTrace)
	if len(t) > 0 {
		return t
	}

	v := h.Get(HeaderXRequestID)
	if len(v) > 0 {
		return v
	}

	// create a new header
	return uuid.New().String()
}
