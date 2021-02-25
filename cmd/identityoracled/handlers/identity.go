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
	Headers                           oracle.Headers
	Contracts                         oracle.Contracts
	Approver                          oracle.ApproverInterface
	IdentityExpirationDurationSeconds int
}

// PubKeySignature returns an approve/deny signature for an association between an entity and a
//   public key.
func (v *Verify) PubKeySignature(ctx context.Context, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Verify.PubKeySignature")
	defer span.End()

	var requestData struct {
		XPub   bitcoin.ExtendedKey `json:"xpub" validate:"required"`
		Index  uint32              `json:"index"`
		Entity actions.EntityField `json:"entity" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	if requestData.XPub.IsPrivate() {
		web.Respond(ctx, w, "private keys not allowed", http.StatusUnprocessableEntity)
		return nil
	}

	ctx = logger.ContextWithLogFields(ctx, []logger.Field{
		logger.Stringer("xpub", requestData.XPub),
		logger.Uint32("index", requestData.Index),
	})

	dbConn := v.MasterDB.Copy()
	defer dbConn.Close()

	user, err := oracle.FetchUserByXPub(ctx, dbConn, bitcoin.ExtendedKeys{requestData.XPub})
	if err != nil {
		return translate(errors.Wrap(err, "fetch user"))
	}

	if v.Approver != nil {
		if approved, description, err := v.Approver.ApproveIdentity(ctx, user.ID); err != nil {
			return translate(errors.Wrap(err, "approve identity"))
		} else if !approved {
			response := struct {
				Status string `json:"status"`
			}{
				Status: description,
			}
			web.Respond(ctx, w, response, http.StatusForbidden)
			return nil
		}
	}

	// Verify that the public key is associated with the entity.
	sigHash, err := oracle.VerifyPubKey(ctx, user, v.Headers, &requestData.Entity,
		requestData.XPub, requestData.Index)
	if err != nil {
		return translate(errors.Wrap(err, "verify pub key"))
	}

	sig, err := v.Key.Sign(sigHash.Hash[:])
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
		Approved:     sigHash.Approved,
		Description:  sigHash.Description,
		SigAlgorithm: 1,
		Signature:    sig,
		BlockHeight:  sigHash.BlockHeight,
	}

	web.RespondData(ctx, w, response, http.StatusOK)
	return nil
}

// XPubSignature returns an approve/deny signature for an association between an entity and an
//   extended public key.
func (v *Verify) XPubSignature(ctx context.Context, w http.ResponseWriter,
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
			web.Respond(ctx, w, "private keys not allowed", http.StatusUnprocessableEntity)
			return nil
		}
	}

	ctx = logger.ContextWithLogFields(ctx, []logger.Field{
		logger.Stringer("xpubs", requestData.XPubs),
	})

	dbConn := v.MasterDB.Copy()
	defer dbConn.Close()

	user, err := oracle.FetchUserByXPub(ctx, dbConn, requestData.XPubs)
	if err != nil {
		return translate(errors.Wrap(err, "fetch user"))
	}

	if v.Approver != nil {
		if approved, description, err := v.Approver.ApproveIdentity(ctx, user.ID); err != nil {
			return translate(errors.Wrap(err, "approve identity"))
		} else if !approved {
			response := struct {
				Status string `json:"status"`
			}{
				Status: description,
			}
			web.Respond(ctx, w, response, http.StatusForbidden)
			return nil
		}
	}

	// Verify that the public key is associated with the entity.
	sigHash, err := oracle.VerifyXPub(ctx, user, v.Headers, &requestData.Entity,
		requestData.XPubs)
	if err != nil {
		return translate(errors.Wrap(err, "verify xpub"))
	}

	sig, err := v.Key.Sign(sigHash.Hash[:])
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
		Approved:     sigHash.Approved,
		Description:  sigHash.Description,
		SigAlgorithm: 1,
		Signature:    sig,
		BlockHeight:  sigHash.BlockHeight,
	}

	web.RespondData(ctx, w, response, http.StatusOK)
	return nil
}

// AdminCertificate returns a certificate verifying that the contract admin address belongs to the
// Issuer entity or entity contract address.
func (v *Verify) AdminCertificate(ctx context.Context, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Verify.AdminCertificate")
	defer span.End()

	var requestData struct {
		XPubs    bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
		Index    uint32               `json:"index"`
		Issuer   actions.EntityField  `json:"issuer"`
		Contract bitcoin.RawAddress   `json:"entity_contract"`
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

	dbConn := v.MasterDB.Copy()
	defer dbConn.Close()

	user, err := oracle.FetchUserByXPub(ctx, dbConn, requestData.XPubs)
	if err != nil {
		return translate(errors.Wrap(err, "fetch user"))
	}

	if v.Approver != nil {
		if approved, description, err := v.Approver.ApproveIdentity(ctx, user.ID); err != nil {
			return translate(errors.Wrap(err, "approve identity"))
		} else if !approved {
			response := struct {
				Status string `json:"status"`
			}{
				Status: description,
			}
			web.Respond(ctx, w, response, http.StatusForbidden)
			return nil
		}
	}

	expiration := uint64(time.Now().Add(time.Duration(v.IdentityExpirationDurationSeconds) *
		time.Second).UnixNano())

	// Verify that the public key is associated with the entity.
	sigHash, err := oracle.CreateAdminCertificate(ctx, dbConn, user, v.Config.Net, v.Config.IsTest,
		v.Headers, v.Contracts, requestData.XPubs, requestData.Index, requestData.Issuer,
		requestData.Contract, expiration)
	if err != nil {
		return translate(errors.Wrap(err, "verify admin"))
	}

	sig, err := v.Key.Sign(sigHash.Hash[:])
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
		Approved:    sigHash.Approved,
		Description: sigHash.Description,
		Signature:   sig,
		BlockHeight: sigHash.BlockHeight,
		Expiration:  expiration,
	}

	web.RespondData(ctx, w, response, http.StatusOK)
	return nil
}
