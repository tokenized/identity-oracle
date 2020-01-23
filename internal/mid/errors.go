package mid

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/smart-contract/pkg/logger"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// ErrorHandler for catching and responding to errors.
func ErrorHandler(next web.Handler) web.Handler {

	// Create the handler that will be attached in the middleware chain.
	h := func(ctx context.Context, log logger.Logger, w http.ResponseWriter, r *http.Request, params map[string]string) error {
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
				log.Printf("%s : ERROR : Panic Caught : %s\n", v.TraceID, r)

				// Respond with the error.
				web.RespondError(ctx, log, w, errors.New("unhandled"), http.StatusInternalServerError)

				// Print out the stack.
				log.Printf("%s : ERROR : Stacktrace\n%s\n", v.TraceID, debug.Stack())
			}
		}()

		if err := next(ctx, log, w, r, params); err != nil {

			// Indicate this request had an error.
			v.Error = true

			// What is the root error.
			c := errors.Cause(err)

			if c != web.ErrNotFound {

				// Log the error.
				log.Printf("%s : ERROR : %v\n", v.TraceID, err)
			}

			// Respond with the error.
			web.Error(ctx, log, w, err)

			// The error has been handled so we can stop propagating it.
			return nil
		}

		return nil
	}

	return h
}
