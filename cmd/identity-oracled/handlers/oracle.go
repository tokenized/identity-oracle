package handlers

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/tokenized/identity-oracle/internal/oracle"
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
		XPub   string `json:"xpub" validate:"required"`
		UserID string `json:"user_id" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	// TODO Authenticate access to add xpub to UserID

	// Insert xpub
	xpub := &oracle.XPub{
		ID:          uuid.New().String(),
		UserID:      requestData.UserID,
		DateCreated: time.Now(),
	}

	var err error
	xpub.XPub, err = bitcoin.ExtendedKeyFromStr(requestData.XPub)
	if err != nil {
		return translate(errors.Wrap(err, "decode xpub"))
	}

	if err := oracle.CreateXPub(ctx, dbConn, xpub); err != nil {
		return translate(errors.Wrap(err, "create xpub"))
	}

	response := struct {
		Status string `json:"status"`
	}{
		Status: "Extended Public Key Added",
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}
