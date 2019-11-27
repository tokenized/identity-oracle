package handlers

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/pkg/errors"
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

// Register adds a new user to the system
func (o *Oracle) Register(ctx context.Context, log *log.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.Register")
	defer span.End()

	var requestData struct {
		Entity       string `json:"entity" validate:"required"`     // hex protobuf
		PublicKey    string `json:"public_key" validate:"required"` // hex compressed
		Jurisdiction string `json:"jurisdiction"`
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

	user := &oracle.User{
		ID:           uuid.New().String(),
		Entity:       entityBytes,
		PublicKey:    pubKey,
		Jurisdiction: requestData.Jurisdiction,
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
func (o *Oracle) AddXPub(ctx context.Context, log *log.Logger, w http.ResponseWriter,
	r *http.Request, params map[string]string) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Oracle.AddXPub")
	defer span.End()

	var requestData struct {
		UserID    string `json:"user_id" validate:"required"`
		XPub      string `json:"xpub" validate:"required"`      // hex
		Signature string `json:"signature" validate:"required"` // hex signature of user id and xpub with users public key
	}

	if err := web.Unmarshal(r.Body, &requestData); err != nil {
		return translate(errors.Wrap(err, "unmarshal request"))
	}

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

	signature, err := bitcoin.SignatureFromStr(requestData.Signature)
	if err != nil {
		return translate(errors.Wrap(err, "decode sig"))
	}

	hash := bitcoin.DoubleSha256([]byte(requestData.UserID + requestData.XPub))

	dbConn := o.MasterDB.Copy()
	defer dbConn.Close()

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
