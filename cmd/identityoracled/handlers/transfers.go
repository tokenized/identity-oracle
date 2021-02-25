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
	Headers                           oracle.Headers
	TransferExpirationDurationSeconds int

	Approver oracle.ApproverInterface
}

// TransferSignature returns an approve/deny signature for a transfer receiver.
func (t *Transfers) TransferSignature(ctx context.Context, w http.ResponseWriter,
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

	for _, xpub := range requestData.XPubs {
		if xpub.IsPrivate() {
			web.Respond(ctx, w, "private keys not allowed", http.StatusUnprocessableEntity)
			return nil
		}
	}

	ctx = logger.ContextWithLogFields(ctx, []logger.Field{
		logger.Stringer("xpubs", requestData.XPubs),
		logger.Uint32("index", requestData.Index),
	})

	dbConn := t.MasterDB.Copy()
	defer dbConn.Close()

	user, err := oracle.FetchUserByXPub(ctx, dbConn, requestData.XPubs)
	if err != nil {
		if errors.Cause(err) == oracle.ErrXPubNotFound {
			web.RespondError(ctx, w, err, http.StatusNotFound)
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
			return translate(errors.Wrap(err, "approve transfer"))
		}
	}

	expiration := uint64(time.Now().Add(time.Duration(t.TransferExpirationDurationSeconds) *
		time.Second).UnixNano())

	// Check that xpub is in DB. Check that entity associated xpub meets criteria for asset.
	sigHash, height, blockHash, err := oracle.CreateReceiveSignature(ctx, dbConn, t.Headers,
		requestData.Contract, requestData.AssetID, requestData.XPubs, requestData.Index, expiration,
		approved)
	if err != nil {
		return translate(errors.Wrap(err, "create signature"))
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
		BlockHash:    blockHash,
		Expiration:   expiration,
	}

	web.RespondData(ctx, w, response, http.StatusOK)
	return nil
}
