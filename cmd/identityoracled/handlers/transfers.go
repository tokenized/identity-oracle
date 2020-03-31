package handlers

import (
	"context"
	"encoding/hex"
	"net/http"

	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"

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
func (t *Transfers) TransferSignature(ctx context.Context, log logger.Logger, w http.ResponseWriter,
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

	// Check that xpub is in DB. Check that entity associated xpub meets criteria for asset.
	sigHash, height, approved, err := oracle.ApproveTransfer(ctx, dbConn, t.BlockHandler,
		requestData.Contract, requestData.AssetID, xpub, requestData.Index, requestData.Quantity)
	if err != nil {
		return translate(errors.Wrap(err, "approve transfer"))
	}

	sig, err := t.Key.Sign(sigHash[:])
	if err != nil {
		return translate(errors.Wrap(err, "sign"))
	}

	response := struct {
		Approved     bool   `json:"approved"`
		SigAlgorithm uint32 `json:"algorithm"`
		Sig          string `json:"signature"`
		BlockHeight  uint32 `json:"block_height"`
	}{
		Approved:     approved,
		SigAlgorithm: 1,
		Sig:          hex.EncodeToString(sig.Bytes()),
		BlockHeight:  height,
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}
