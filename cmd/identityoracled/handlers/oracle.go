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
	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
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
func (o *Oracle) Identity(ctx context.Context, w http.ResponseWriter, r *http.Request,
	params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.Identity")
	defer span.End()

	response := struct {
		ContractAddress bitcoin.RawAddress `json:"contract_address"`
		PublicKey       bitcoin.PublicKey  `json:"public_key"`
	}{
		ContractAddress: o.ContractAddress,
		PublicKey:       o.Key.PublicKey(),
	}

	web.RespondData(ctx, w, response, http.StatusOK)
	return nil
}

// Register adds a new user to the system
func (o *Oracle) Register(ctx context.Context, w http.ResponseWriter, r *http.Request,
	params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.Register")
	defer span.End()

	var requestData struct {
		Entity    actions.EntityField `json:"entity" validate:"required"`
		PublicKey bitcoin.PublicKey   `json:"public_key" validate:"required"`
		Signature bitcoin.Signature   `json:"signature" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	logger.InfoWithFields(ctx, []logger.Field{
		logger.Stringer("public_key", requestData.PublicKey),
	}, "Creating user")

	// Verify signature is valid for user's public key
	s := sha256.New()
	if err := requestData.Entity.WriteDeterministic(s); err != nil {
		return translate(errors.Wrap(err, "write entity"))
	}
	hash := sha256.Sum256(s.Sum(nil))

	if !requestData.Signature.Verify(hash, requestData.PublicKey) {
		return translate(oracle.ErrInvalidSignature)
	}

	entityBytes, err := proto.Marshal(&requestData.Entity)
	if err != nil {
		return translate(errors.Wrap(err, "protobuf marshal entity"))
	}

	userID := uuid.New().String()

	if o.Approver != nil {
		if approved, description, err := o.Approver.ApproveRegistration(ctx, userID,
			requestData.Entity, requestData.PublicKey); err != nil {
			return translate(errors.Wrap(err, "approve registration"))
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

	// Insert user
	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	user := &oracle.User{
		ID:           userID,
		Entity:       entityBytes,
		PublicKey:    requestData.PublicKey,
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		IsDeleted:    false,
	}

	logger.Info(ctx, "Created user : %s", userID)

	if err := oracle.CreateUser(ctx, dbConn, user); err != nil {
		return translate(errors.Wrap(err, "create user"))
	}

	response := struct {
		Status string `json:"status"`
		UserID string `json:"user_id"`
	}{
		Status: "User Created",
		UserID: user.ID,
	}

	web.RespondData(ctx, w, response, http.StatusOK)
	return nil
}

// AddXPub adds a new xpub to the system.
func (o *Oracle) AddXPub(ctx context.Context, w http.ResponseWriter, r *http.Request,
	params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.AddXPub")
	defer span.End()

	var requestData struct {
		UserID          string               `json:"user_id" validate:"required"`
		XPubs           bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
		RequiredSigners int                  `json:"required_signers" validate:"required"`
		Signature       bitcoin.Signature    `json:"signature" validate:"required"`
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

	logger.InfoWithFields(ctx, []logger.Field{
		logger.String("user_id", requestData.UserID),
		logger.Stringer("xpubs", requestData.XPubs),
	}, "Adding xpub")

	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	// Fetch User
	user, err := oracle.FetchUser(ctx, dbConn, requestData.UserID)
	if err != nil {
		return translate(errors.Wrap(err, "fetch user"))
	}

	userid, err := uuid.Parse(requestData.UserID)
	if err != nil {
		return translate(errors.Wrap(err, "parse user id"))
	}

	// Verify signature is valid for user's public key
	s := sha256.New()
	s.Write(userid[:])
	s.Write(requestData.XPubs.Bytes())
	if err := binary.Write(s, binary.LittleEndian, uint32(requestData.RequiredSigners)); err != nil {
		return translate(errors.Wrap(err, "hash signers"))
	}
	hash := sha256.Sum256(s.Sum(nil))

	if !requestData.Signature.Verify(hash, user.PublicKey) {
		return translate(oracle.ErrInvalidSignature)
	}

	xpub := &oracle.XPub{
		UserID:          requestData.UserID,
		XPub:            requestData.XPubs,
		RequiredSigners: requestData.RequiredSigners,
		DateCreated:     time.Now(),
	}

	// Insert xpub
	if err := oracle.CreateXPub(ctx, dbConn, xpub); err != nil {
		return translate(errors.Wrap(err, "create xpub"))
	}

	logger.Info(ctx, "Created xpub")

	web.Respond(ctx, w, nil, http.StatusOK)
	return nil
}

// User returns the user id associated with an xpub.
func (o *Oracle) User(ctx context.Context, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.User")
	defer span.End()

	var requestData struct {
		XPubs bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	logger.InfoWithFields(ctx, []logger.Field{
		logger.Stringer("xpubs", requestData.XPubs),
	}, "Finding user")

	for _, xpub := range requestData.XPubs {
		if xpub.IsPrivate() {
			web.Respond(ctx, w, "private keys not allowed", http.StatusUnprocessableEntity)
			return nil
		}
	}

	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	// Check user ID
	userID, err := oracle.FetchUserIDByXPub(ctx, dbConn, requestData.XPubs)
	if err != nil {
		return translate(errors.Wrap(err, "fetch user"))
	}

	response := struct {
		UserID string `json:"user_id"`
	}{
		UserID: *userID,
	}

	web.RespondData(ctx, w, response, http.StatusOK)
	return nil
}

// UpdateIdentity updates the users identity information.
func (o *Oracle) UpdateIdentity(ctx context.Context, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.UpdateIdentity")
	defer span.End()

	var requestData struct {
		UserID    string              `json:"user_id" validate:"required"`
		Entity    actions.EntityField `json:"entity" validate:"required"`
		Signature bitcoin.Signature   `json:"signature" validate:"required"`
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	logger.InfoWithFields(ctx, []logger.Field{
		logger.String("user_id", requestData.UserID),
	}, "Updating identity")

	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	// Fetch User
	user, err := oracle.FetchUser(ctx, dbConn, requestData.UserID)
	if err != nil {
		if err == db.ErrNotFound {
			return web.ErrNotFound // User doesn't exist
		}
		return translate(errors.Wrap(err, "fetch user"))
	}

	// Verify signature is valid for user's public key
	s := sha256.New()
	userID, err := uuid.Parse(requestData.UserID)
	if err != nil {
		return translate(errors.Wrap(err, "parse user id"))
	}
	if _, err := s.Write(userID[:]); err != nil {
		return translate(errors.Wrap(err, "write user id"))
	}
	if err := requestData.Entity.WriteDeterministic(s); err != nil {
		return translate(errors.Wrap(err, "write entity"))
	}
	hash := sha256.Sum256(s.Sum(nil))

	if !requestData.Signature.Verify(hash, user.PublicKey) {
		return translate(oracle.ErrInvalidSignature)
	}

	if o.Approver != nil {
		if approved, description, err := o.Approver.UpdateIdentity(ctx, user.ID,
			requestData.Entity); err != nil {
			return translate(errors.Wrap(err, "approve update entity"))
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

	// Update user in database
	entityBytes, err := proto.Marshal(&requestData.Entity)
	if err != nil {
		return translate(errors.Wrap(err, "protobuf marshal entity"))
	}
	user.Entity = entityBytes

	if err := oracle.UpdateUser(ctx, dbConn, user); err != nil {
		return translate(errors.Wrap(err, "update user"))
	}

	web.Respond(ctx, w, nil, http.StatusOK)
	return nil
}
