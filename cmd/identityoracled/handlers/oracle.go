package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"net/http"
	"time"

	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// Oracle provides support for identity checks.
type Oracle struct {
	Config          *web.Config
	MasterDB        *db.DB
	Approver        oracle.ApproverInterface
	Key             bitcoin.Key
	ContractAddress bitcoin.RawAddress
}

// Identity returns identity information about the oracle.
func (o *Oracle) Identity(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.Identity")
	defer span.End()

	response := struct {
		ContractAddress bitcoin.RawAddress `json:"contract_address"`
		PublicKey       bitcoin.PublicKey  `json:"public_key"`
	}{
		ContractAddress: o.ContractAddress,
		PublicKey:       o.Key.PublicKey(),
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}

// Register adds a new user to the system
func (o *Oracle) Register(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.Register")
	defer span.End()

	var requestData struct {
		Entity    actions.EntityField `json:"entity" validate:"required"`
		PublicKey bitcoin.PublicKey   `json:"public_key" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	entityBytes, err := proto.Marshal(&requestData.Entity)
	if err != nil {
		return translate(errors.Wrap(err, "protobuf marshal entity"))
	}

	// Insert user
	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	if o.Approver != nil {
		if status, err := o.Approver.ApproveRegistration(ctx, requestData.Entity,
			requestData.PublicKey); err != nil {
			web.RespondError(ctx, log, w, err, status)
			return nil
		}
	}

	user := &oracle.User{
		Entity:       entityBytes,
		PublicKey:    requestData.PublicKey,
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		Approved:     true, // TODO Add approval step
		IsDeleted:    false,
	}

	if err := oracle.CreateUser(ctx, dbConn, user); err != nil {
		return translate(errors.Wrap(err, "create user"))
	}

	response := struct {
		Status string    `json:"status"`
		UserID uuid.UUID `json:"user_id"`
	}{
		Status: "User Created",
		UserID: user.ID,
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}

// AddXPub adds a new xpub to the system.
func (o *Oracle) AddXPub(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.AddXPub")
	defer span.End()

	var requestData struct {
		UserID          uuid.UUID            `json:"user_id" validate:"required"`
		XPubs           bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
		RequiredSigners int                  `json:"required_signers" validate:"required"`
		Signature       bitcoin.Signature    `json:"signature" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	// Check user ID
	_, err := oracle.FetchUser(ctx, dbConn, requestData.UserID)
	if err != nil {
		if err == db.ErrNotFound {
			return web.ErrNotFound // User doesn't exist
		}
		return translate(errors.Wrap(err, "fetch user"))
	}

	xpub := &oracle.XPub{
		UserID:          requestData.UserID,
		XPub:            requestData.XPubs,
		RequiredSigners: requestData.RequiredSigners,
		DateCreated:     time.Now(),
	}

	// Fetch User
	user, err := oracle.FetchUser(ctx, dbConn, requestData.UserID)
	if err != nil {
		return translate(errors.Wrap(err, "fetch user"))
	}

	// Verify signature is valid for user's public key
	s := sha256.New()
	s.Write(requestData.UserID[:])
	s.Write(requestData.XPubs.Bytes())
	if err := binary.Write(s, binary.LittleEndian, uint32(requestData.RequiredSigners)); err != nil {
		return translate(errors.Wrap(err, "hash signers"))
	}
	hash := sha256.Sum256(s.Sum(nil))

	if !requestData.Signature.Verify(hash[:], user.PublicKey) {
		return translate(errors.Wrap(err, "validate sig"))
	}

	// Insert xpub
	if err := oracle.CreateXPub(ctx, dbConn, xpub); err != nil {
		return translate(errors.Wrap(err, "create xpub"))
	}

	web.Respond(ctx, log, w, nil, http.StatusOK)
	return nil
}

// User returns the user id associated with an xpub.
func (o *Oracle) User(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.User")
	defer span.End()

	var requestData struct {
		XPubs bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	// Check user ID
	userID, err := oracle.FetchUserIDByXPub(ctx, dbConn, requestData.XPubs)
	if err != nil {
		if err == oracle.ErrXPubNotFound {
			return web.ErrNotFound // XPub doesn't exist
		}
		return translate(errors.Wrap(err, "fetch user"))
	}

	response := struct {
		UserID uuid.UUID `json:"user_id"`
	}{
		UserID: userID,
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}

// TODO Change Entity Data?
