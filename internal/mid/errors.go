package mid

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/pkg/logger"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// ErrorHandler for catching and responding to errors.
func ErrorHandler(next web.Handler) web.Handler {

	// Create the handler that will be attached in the middleware chain.
	h := func(ctx context.Context, w http.ResponseWriter, r *http.Request,
		params map[string]string) error {
		ctx, span := trace.StartSpan(ctx, "internal.mid.ErrorHandler")
		defer span.End()

		v := ctx.Value(web.KeyValues).(*web.Values)

		// In the event of a panic, we want to capture it here so we can send an
		// error down the stack.
		defer func() {
			if r := recover(); r != nil {

				// Indicate this request had an error.
				v.Error = true

				// Log the panic.
				logger.Error(ctx, "Panic Caught : %s", r)

				// Respond with the error.
				web.RespondError(ctx, w, errors.New("unhandled"), http.StatusInternalServerError)

				// Print out the stack.
				logger.Error(ctx, "Stacktrace\n%s", debug.Stack())
			}
		}()

		return next(ctx, w, r, params)
	}

	return h
}
