package mid

import (
	"context"
	"net/http"
	"time"

	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/smart-contract/pkg/logger"

	"go.opencensus.io/trace"
)

// RequestLogger writes some information about the request to the logs in
// the format: TraceID : (200) GET /foo -> IP ADDR (latency)
func RequestLogger(next web.Handler) web.Handler {

	// Wrap this handler around the next one provided.
	h := func(ctx context.Context, log logger.Logger, w http.ResponseWriter, r *http.Request, params map[string]string) error {
		ctx, span := trace.StartSpan(ctx, "internal.mid.RequestLogger")
		defer span.End()

		err := next(ctx, log, w, r, params)

		v := ctx.Value(web.KeyValues).(*web.Values)

		log.Printf("%s : (%d) : %s %s -> %s (%s)",
			v.TraceID,
			v.StatusCode,
			r.Method, r.URL.Path,
			r.RemoteAddr, time.Since(v.Now),
		)

		// For consistency return the error we received.
		return err
	}

	return h
}
