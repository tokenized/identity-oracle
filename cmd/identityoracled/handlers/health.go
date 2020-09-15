package handlers

import (
	"context"
	"net/http"

	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/pkg/logger"
)

// Health provides health checks.
type Health struct {
	MasterDB *db.DB
}

// Health just returns a 200 okay status.
func (h *Health) Health(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {
	var status struct {
		Status string `json:"status"`
	}

	if err := checkDB(ctx, h.MasterDB); err != nil {
		status.Status = err.Error()
		web.Respond(ctx, log, w, status, http.StatusInternalServerError)
	}

	web.Respond(ctx, log, w, nil, http.StatusOK)
	return nil
}

// checkDB performs a status check on a DB.
func checkDB(ctx context.Context, db *db.DB) error {
	dbConn := db.Copy()
	defer dbConn.Close()

	return dbConn.StatusCheck(ctx)
}
