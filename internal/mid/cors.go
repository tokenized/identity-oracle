package mid

import (
	"context"
	"log"
	"net/http"

	"github.com/tokenized/identity-oracle/internal/platform/web"

	"go.opencensus.io/trace"
)

// CORS middleware
func CORS(next web.Handler) web.Handler {

	// Wrap this handler around the next one provided.
	h := func(ctx context.Context, log *log.Logger, w http.ResponseWriter, r *http.Request, params map[string]string) error {
		ctx, span := trace.StartSpan(ctx, "internal.mid.CORS")
		defer span.End()

		CORSHandler(w, r, params)

		err := next(ctx, log, w, r, params)

		// For consistency return the error we received.
		return err
	}

	return h
}

// Adds CORS headers
func CORSHandler(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, PUT, GET, HEAD, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Request-ID")
		// w.WriteHeader(http.StatusNoContent)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
}
