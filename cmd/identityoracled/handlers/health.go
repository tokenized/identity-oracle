package handlers

import (
	"context"
	"net/http"

	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/tokenized/smart-contract/pkg/logger"
)

// Health provides health checks.
type Health struct{}

// Health just returns a 200 okay status.
func (h *Health) Health(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {
	web.Respond(ctx, log, w, nil, http.StatusOK)
	return nil
}
