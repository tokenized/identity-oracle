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
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// Verify provides support for providing signatures to prove identity.
type Verify struct {
	Config                            *web.Config
	MasterDB                          *db.DB
	Key                               bitcoin.Key
	BlockHandler                      *oracle.BlockHandler
	Approver                          oracle.ApproverInterface
	IdentityExpirationDurationSeconds int
}

// PubKeySignature returns an approve/deny signature for an association between an entity and a
//   public key.
func (v *Verify) PubKeySignature(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Verify.PubKeySignature")
	defer span.End()

	var requestData struct {
		XPub   bitcoin.ExtendedKey `json:"xpub" validate:"required"`
		Index  uint32              `json:"index" validate:"required"`
		Entity actions.EntityField `json:"entity" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	if requestData.XPub.IsPrivate() {
		web.Respond(ctx, log, w, "private keys not allowed", http.StatusUnprocessableEntity)
		return nil
	}

	dbConn := v.MasterDB.Copy()
	defer dbConn.Close()

	// Verify that the public key is associated with the entity.
	sigHash, height, approved, description, err := oracle.VerifyPubKey(ctx, dbConn, v.BlockHandler,
		&requestData.Entity, requestData.XPub, requestData.Index)
	if err != nil {
		return translate(errors.Wrap(err, "verify pub key"))
	}

	sig, err := v.Key.Sign(sigHash[:])
	if err != nil {
		return translate(errors.Wrap(err, "sign"))
	}

	response := struct {
		Approved     bool              `json:"approved"`
		Description  string            `json:"description"`
		SigAlgorithm uint32            `json:"algorithm"`
		Signature    bitcoin.Signature `json:"signature"`
		BlockHeight  uint32            `json:"block_height"`
	}{
		Approved:     approved,
		Description:  description,
		SigAlgorithm: 1,
		Signature:    sig,
		BlockHeight:  height,
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}

// XPubSignature returns an approve/deny signature for an association between an entity and an
//   extended public key.
func (v *Verify) XPubSignature(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Verify.XPubSignature")
	defer span.End()

	var requestData struct {
		XPubs  bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
		Entity actions.EntityField  `json:"entity" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	for _, xpub := range requestData.XPubs {
		if xpub.IsPrivate() {
			web.Respond(ctx, log, w, "private keys not allowed", http.StatusUnprocessableEntity)
			return nil
		}
	}

	dbConn := v.MasterDB.Copy()
	defer dbConn.Close()

	// Verify that the public key is associated with the entity.
	sigHash, height, approved, description, err := oracle.VerifyXPub(ctx, dbConn, v.BlockHandler,
		&requestData.Entity, requestData.XPubs)
	if err != nil {
		return translate(errors.Wrap(err, "verify xpub"))
	}

	sig, err := v.Key.Sign(sigHash[:])
	if err != nil {
		return translate(errors.Wrap(err, "sign"))
	}

	response := struct {
		Approved     bool              `json:"approved"`
		Description  string            `json:"description"`
		SigAlgorithm uint32            `json:"algorithm"`
		Sig          bitcoin.Signature `json:"signature"`
		BlockHeight  uint32            `json:"block_height"`
	}{
		Approved:     approved,
		Description:  description,
		SigAlgorithm: 1,
		Sig:          sig,
		BlockHeight:  height,
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}

// AdminCertificate returns a certificate verifying that the contract admin address belongs to the
// Issuer entity or entity contract address.
func (v *Verify) AdminCertificate(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Verify.AdminCertificate")
	defer span.End()

	var requestData struct {
		XPubs    bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
		Index    uint32               `json:"index" validate:"required"`
		Issuer   actions.EntityField  `json:"issuer"`
		Contract bitcoin.RawAddress   `json:"entity_contract"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	for _, xpub := range requestData.XPubs {
		if xpub.IsPrivate() {
			web.Respond(ctx, log, w, "private keys not allowed", http.StatusUnprocessableEntity)
			return nil
		}
	}

	dbConn := v.MasterDB.Copy()
	defer dbConn.Close()

	expiration := uint64(time.Now().Add(time.Duration(v.IdentityExpirationDurationSeconds) * time.Second).UnixNano())

	// Verify that the public key is associated with the entity.
	sigHash, height, approved, description, err := oracle.CreateAdminCertificate(ctx, dbConn,
		v.Config.IsTest, v.BlockHandler, requestData.XPubs, requestData.Index, requestData.Issuer,
		requestData.Contract, expiration)
	if err != nil {
		return translate(errors.Wrap(err, "verify admin"))
	}

	sig, err := v.Key.Sign(sigHash[:])
	if err != nil {
		return translate(errors.Wrap(err, "sign"))
	}

	response := struct {
		Approved    bool              `json:"approved"`
		Description string            `json:"description"`
		Signature   bitcoin.Signature `json:"signature"`
		BlockHeight uint32            `json:"block_height"`
		Expiration  uint64            `json:"expiration"`
	}{
		Approved:    approved,
		Description: description,
		Signature:   sig,
		BlockHeight: height,
		Expiration:  expiration,
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}
