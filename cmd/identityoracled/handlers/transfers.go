package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// Transfer provides support for transferring bitcoin and tokens.
type Transfers struct {
	Config                            *web.Config
	MasterDB                          *db.DB
	Key                               bitcoin.Key
	BlockHandler                      *oracle.BlockHandler
	TransferExpirationDurationSeconds int

	Approver oracle.ApproverInterface
}

// TransferSignature returns an approve/deny signature for a transfer receiver.
func (t *Transfers) TransferSignature(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Transfers.TransferSignature")
	defer span.End()

	var requestData struct {
		XPubs    bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
		Index    uint32               `json:"index"`
		Contract string               `json:"contract" validate:"required"`
		AssetID  string               `json:"asset_id" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	dbConn := t.MasterDB.Copy()
	defer dbConn.Close()

	user, err := oracle.FetchUserByXPub(ctx, dbConn, requestData.XPubs)
	if err != nil {
		if errors.Cause(err) == oracle.ErrXPubNotFound {
			web.RespondError(ctx, log, w, err, http.StatusNotFound)
			return nil
		}
		return translate(errors.Wrap(err, "fetch user"))
	}

	approved := true
	var description string
	if t.Approver != nil {
		var approveErr error
		approved, description, approveErr = t.Approver.ApproveTransfer(ctx, requestData.Contract,
			requestData.AssetID, user.ID)
		if approveErr != nil {
			return translate(errors.Wrap(err, "approver"))
		}
	}

	expiration := uint64(time.Now().Add(time.Duration(t.TransferExpirationDurationSeconds) * time.Second).UnixNano())

	// Check that xpub is in DB. Check that entity associated xpub meets criteria for asset.
	sigHash, height, hash, err := oracle.CreateReceiveSignature(ctx, dbConn,
		t.BlockHandler, requestData.Contract, requestData.AssetID, requestData.XPubs,
		requestData.Index, expiration, approved)
	if err != nil {
		return translate(errors.Wrap(err, "approve transfer"))
	}

	sig, err := t.Key.Sign(sigHash[:])
	if err != nil {
		return translate(errors.Wrap(err, "sign"))
	}

	response := struct {
		Approved     bool              `json:"approved"`
		Description  string            `json:"description"`
		SigAlgorithm uint32            `json:"algorithm"`
		Sig          bitcoin.Signature `json:"signature"`
		BlockHeight  uint32            `json:"block_height"`
		BlockHash    bitcoin.Hash32    `json:"block_hash"`
		Expiration   uint64            `json:"expiration"`
	}{
		Approved:     approved,
		Description:  description,
		SigAlgorithm: 1,
		Sig:          sig,
		BlockHeight:  height,
		BlockHash:    hash,
		Expiration:   expiration,
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}
