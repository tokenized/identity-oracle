package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// Transfer provides support for transferring bitcoin and tokens.
type Transfers struct {
	Config       *web.Config
	MasterDB     *db.DB
	Key          bitcoin.Key
	BlockHandler *oracle.BlockHandler
}

// TransferSignature returns an approve/deny signature for a transfer receiver.
func (t *Transfers) TransferSignature(ctx context.Context, log *log.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Transfers.TransferSignature")
	defer span.End()

	var requestData struct {
		XPub     string `json:"xpub" validate:"required"`
		Index    uint32 `json:"index" validate:"required"`
		Contract string `json:"contract" validate:"required"`
		AssetID  string `json:"asset_id" validate:"required"`
		Quantity uint64 `json:"quantity" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	xpub, err := bitcoin.ExtendedKeysFromStr(requestData.XPub)
	if err != nil {
		return translate(errors.Wrap(err, "decode xpub"))
	}

	dbConn := t.MasterDB.Copy()
	defer dbConn.Close()

	// Fetch user for xpub
	user, err := oracle.FetchUserByXPub(ctx, dbConn, xpub)
	if err != nil {
		return translate(errors.Wrap(err, "fetch user"))
	}

	// Check that xpub is in DB. Check that entity associated xpub meets criteria for asset.
	sig, height, approved, err := oracle.ApproveTransfer(ctx, dbConn, t.BlockHandler, user,
		requestData.Contract, requestData.AssetID, xpub, requestData.Index, requestData.Quantity)
	if err != nil {
		return translate(errors.Wrap(err, "approve transfer"))
	}

	response := struct {
		Approved     bool   `json:"approved"`
		SigAlgorithm uint32 `json:"algorithm"`
		Sig          []byte `json:"signature"`
		BlockHeight  uint32 `json:"block_height"`
	}{
		Approved:     approved,
		SigAlgorithm: 1,
		Sig:          sig,
		BlockHeight:  height,
	}

	web.Respond(ctx, log, w, response, http.StatusOK)
	return nil
}
