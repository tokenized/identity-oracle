package handlers

import (
	"context"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"

	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

// Oracle provides support for identity checks.
type Oracle struct {
	Config   *web.Config
	MasterDB *db.DB
	Key      bitcoin.Key
}

// Identity returns identity information about the oracle.
func (o *Oracle) Identity(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.Identity")
	defer span.End()

	response := struct {
		Entity    string `json:"entity"`
		URL       string `json:"url"`
		PublicKey string `json:"public_key"`
	}{
		URL:       o.Config.RootURL,
		PublicKey: hex.EncodeToString(o.Key.PublicKey().Bytes()),
	}

	entityBytes, err := proto.Marshal(&o.Config.Entity)
	if err != nil {
		return errors.Wrap(err, "serialize entity")
	}
	response.Entity = hex.EncodeToString(entityBytes)

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}

// Register adds a new user to the system
func (o *Oracle) Register(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.Register")
	defer span.End()

	// TODO Add birth date
	var requestData struct {
		Entity    string `json:"entity" validate:"required"`     // hex protobuf
		PublicKey string `json:"public_key" validate:"required"` // hex compressed
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	entityBytes, err := hex.DecodeString(requestData.Entity)
	if err != nil {
		return translate(errors.Wrap(err, "decode entity hex"))
	}

	entity := &actions.EntityField{}
	if err := proto.Unmarshal(entityBytes, entity); err != nil {
		return translate(errors.Wrap(err, "unmarshal entity"))
	}

	pubKey, err := bitcoin.PublicKeyFromStr(requestData.PublicKey)
	if err != nil {
		return translate(errors.Wrap(err, "decode public key"))
	}

	// Insert user
	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	user := oracle.User{
		ID:           uuid.New().String(),
		Entity:       entityBytes,
		PublicKey:    pubKey,
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		Approved:     true, // TODO Add approval step
		IsDeleted:    false,
	}

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

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}

// AddXPub adds a new xpub to the system.
func (o *Oracle) AddXPub(ctx context.Context, log logger.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.AddXPub")
	defer span.End()

	var requestData struct {
		UserID          string `json:"user_id" validate:"required"`
		XPub            string `json:"xpub" validate:"required"` // hex
		RequiredSigners int    `json:"required_signers" validate:"required"`
		Signature       string `json:"signature" validate:"required"` // hex signature of user id and xpub with users public key
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

	xpub := oracle.XPub{
		ID:              uuid.New().String(),
		UserID:          requestData.UserID,
		RequiredSigners: requestData.RequiredSigners,
		DateCreated:     time.Now(),
	}

	xpub.XPub, err = bitcoin.ExtendedKeysFromStr(requestData.XPub)
	if err != nil {
		return translate(errors.Wrap(err, "decode xpub"))
	}

	signature, err := bitcoin.SignatureFromStr(requestData.Signature)
	if err != nil {
		return translate(errors.Wrap(err, "decode sig"))
	}

	hash := bitcoin.DoubleSha256([]byte(requestData.UserID + requestData.XPub))

	// Fetch User
	user, err := oracle.FetchUser(ctx, dbConn, requestData.UserID)
	if err != nil {
		return translate(errors.Wrap(err, "fetch user"))
	}

	// Verify signature is valid for user's public key
	if !signature.Verify(hash, user.PublicKey) {
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
		XPub string `json:"xpub" validate:"required"` // hex
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

	xpub, err := bitcoin.ExtendedKeysFromStr(requestData.XPub)
	if err != nil {
		return translate(errors.Wrap(err, "decode xpub"))
	}

	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

	// Check user ID
	userID, err := oracle.FetchUserIDByXPub(ctx, dbConn, xpub)
	if err != nil {
		if err == oracle.ErrXPubNotFound {
			return web.ErrNotFound // XPub doesn't exist
		}
		return translate(errors.Wrap(err, "fetch user"))
	}

	response := struct {
		UserID string `json:"user_id"`
	}{
		UserID: userID,
	}

	web.RespondData(ctx, log, w, response, http.StatusOK)
	return nil
}

// TODO Change Entity Data?
