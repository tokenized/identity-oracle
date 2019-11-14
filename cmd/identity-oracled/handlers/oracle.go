package handlers

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/pkg/errors"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"

	"go.opencensus.io/trace"
)

// Oracle provides support for orchestration health checks.
type Oracle struct {
	Config   *web.Config
	MasterDB *db.DB
	Key      bitcoin.Key
	Entity   actions.EntityField
}

// Identity returns identity information about the oracle.
func (o *Oracle) Identity(ctx context.Context, log *log.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.Identity")
	defer span.End()

	response := struct {
		Entity    actions.EntityField `json:"entity"`
		URL       string              `json:"url"`
		PublicKey string              `json:"public_key"`
	}{
		Entity:    o.Entity,
		URL:       o.Config.RootURL,
		PublicKey: hex.EncodeToString(o.Key.PublicKey().Bytes()),
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}

// Register adds a new xpub to the system
func (o *Oracle) Register(ctx context.Context, log *log.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.Register")
	defer span.End()

	var requestData struct {
		XPub   string              `json:"xpub" validate:"required"`
		Entity actions.EntityField `json:"entity" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return errors.Wrap(err, "unmarshal request")
	}

	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	// TODO Insert xpub

	response := struct {
		Status string
	}{
		Status: "Extended Public Key Added",
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}
