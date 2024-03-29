package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tokenized/identity-oracle/internal/mid"
	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/tests"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

func TestRegister(t *testing.T) {
	ctx := tests.Context()
	test := tests.New()

	oracleKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate oracle key : %s", err)
	}
	handler := &Oracle{
		Config:   test.WebConfig,
		MasterDB: test.MasterDB,
		Key:      oracleKey,
	}

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate user key : %s", err)
	}

	entity := actions.EntityField{
		Name: "Test Handler Entity Name",
	}

	requestData := struct {
		Entity    actions.EntityField `json:"entity" validate:"required"`     // hex protobuf
		PublicKey bitcoin.PublicKey   `json:"public_key" validate:"required"` // hex compressed
		Signature bitcoin.Signature   `json:"signature" validate:"required"`
	}{
		Entity:    entity,
		PublicKey: key.PublicKey(),
	}

	// Sign the entity
	s := sha256.New()
	if err := requestData.Entity.WriteDeterministic(s); err != nil {
		t.Fatalf("Failed to write entity : %s", err)
	}
	hash := sha256.Sum256(s.Sum(nil))

	requestData.Signature, err = key.Sign(hash)
	if err != nil {
		t.Fatalf("Failed to sign entity : %s", err)
	}

	b, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Failed to serialize request data : %s", err)
	}
	requestBuf := bytes.NewBuffer(b)
	request, err := http.NewRequest("POST", "http://test.com/register", requestBuf)
	if err != nil {
		t.Fatalf("Failed to create request : %s", err)
	}

	params := map[string]string{}

	response := &MockResponseWriter{
		header: http.Header{},
	}

	err = handler.Register(ctx, response, request, params)
	if err != nil {
		t.Fatalf("Failed to register : %s", err)
	}

	if response.StatusCode != 200 {
		t.Fatalf("Response is not success : %d", response.StatusCode)
	}

	var responseData struct {
		Data struct {
			Status string    `json:"status"`
			UserID uuid.UUID `json:"user_id"`
		}
	}

	if err := web.Unmarshal(&response.buffer, &responseData); err != nil {
		t.Fatalf("Failed to unmarshal response : %s", err)
	}

	t.Logf("Status  : %s", responseData.Data.Status)
	t.Logf("User ID : %s", responseData.Data.UserID)

	ruser, err := oracle.FetchUser(ctx, test.MasterDB, responseData.Data.UserID.String())
	if err != nil {
		t.Fatalf("Failed to fetch user : %s", err)
	}

	rentity := &actions.EntityField{}
	if err := proto.Unmarshal(ruser.Entity, rentity); err != nil {
		t.Fatalf("Failed to unmarshal entity : %s", err)
	}

	if !rentity.Equal(&requestData.Entity) {
		t.Errorf("Wrong entity : \n  got  %+v\n  want %+v", *rentity, requestData.Entity)
	}
}

func TestAddXPub(t *testing.T) {
	ctx := tests.Context()
	test := tests.New()

	oracleKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate oracle key : %s", err)
	}
	handler := &Oracle{
		Config:   test.WebConfig,
		MasterDB: test.MasterDB,
		Key:      oracleKey,
	}

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate user key : %s", err)
	}

	entity := actions.EntityField{
		Name:        "Test Entity Name",
		CountryCode: "AUS",
	}

	entityBytes, err := proto.Marshal(&entity)
	if err != nil {
		t.Fatalf("Failed to serialize user entity : %s", err)
	}

	userID := uuid.New()

	user := &oracle.User{
		ID:           userID.String(),
		Entity:       entityBytes,
		PublicKey:    key.PublicKey(),
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		IsDeleted:    false,
	}

	if err := oracle.CreateUser(ctx, test.MasterDB, user); err != nil {
		t.Fatalf("Failed to create user : %s", err)
	}

	xkey, err := bitcoin.GenerateMasterExtendedKey()
	if err != nil {
		t.Fatalf("Failed to generate xkey : %s", err)
	}

	xkeys := bitcoin.ExtendedKeys{xkey}

	requestData := struct {
		UserID          string               `json:"user_id" validate:"required"`
		XPubs           bitcoin.ExtendedKeys `json:"xpubs" validate:"required"` // hex
		RequiredSigners int                  `json:"required_signers" validate:"required"`
		Signature       bitcoin.Signature    `json:"signature" validate:"required"` // hex signature of user id and xpub with users public key
	}{
		UserID:          user.ID,
		XPubs:           xkeys.ExtendedPublicKeys(),
		RequiredSigners: 1,
	}

	s := sha256.New()
	s.Write(userID[:])
	s.Write(requestData.XPubs.Bytes())
	if err := binary.Write(s, binary.LittleEndian, uint32(requestData.RequiredSigners)); err != nil {
		t.Fatalf("Failed to hash required signers : %s", err)
	}
	hash := sha256.Sum256(s.Sum(nil))

	requestData.Signature, err = key.Sign(hash)
	if err != nil {
		t.Fatalf("Failed to generate signature : %s", err)
	}

	b, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Failed to serialize request data : %s", err)
	}
	requestBuf := bytes.NewBuffer(b)
	request, err := http.NewRequest("POST", "http://test.com/addXPub", requestBuf)
	if err != nil {
		t.Fatalf("Failed to create request : %s", err)
	}

	params := map[string]string{}

	response := &MockResponseWriter{
		header: http.Header{},
	}

	// Insert XPub
	err = handler.AddXPub(ctx, response, request, params)
	if err != nil {
		t.Fatalf("Failed to add xpub : %s", err)
	}

	if response.StatusCode != 200 {
		t.Fatalf("Response is not success : %d", response.StatusCode)
	}

	ruserID, err := oracle.FetchUserIDByXPub(ctx, test.MasterDB, xkeys.ExtendedPublicKeys())
	if err != nil {
		t.Fatalf("Failed to fetch xpub user : %s", err)
	}

	if *ruserID != userID.String() {
		t.Errorf("Wrong xpub user : got %s, want %s", *ruserID, userID)
	}
}

func TestAddXPubNoUser(t *testing.T) {
	test := tests.New()

	oracleKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate oracle key : %s", err)
	}
	handler := &Oracle{
		Config:   test.WebConfig,
		MasterDB: test.MasterDB,
		Key:      oracleKey,
	}

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate user key : %s", err)
	}

	userID := uuid.New()

	xkey, err := bitcoin.GenerateMasterExtendedKey()
	if err != nil {
		t.Fatalf("Failed to generate xkey : %s", err)
	}

	xkeys := bitcoin.ExtendedKeys{xkey}

	requestData := struct {
		UserID          string               `json:"user_id" validate:"required"`
		XPubs           bitcoin.ExtendedKeys `json:"xpubs" validate:"required"` // hex
		RequiredSigners int                  `json:"required_signers" validate:"required"`
		Signature       bitcoin.Signature    `json:"signature" validate:"required"` // hex signature of user id and xpub with users public key
	}{
		UserID:          userID.String(),
		XPubs:           xkeys.ExtendedPublicKeys(),
		RequiredSigners: 1,
	}

	s := sha256.New()
	s.Write(userID[:])
	s.Write(requestData.XPubs.Bytes())
	if err := binary.Write(s, binary.LittleEndian, uint32(requestData.RequiredSigners)); err != nil {
		t.Fatalf("Failed to hash required signers : %s", err)
	}
	hash := sha256.Sum256(s.Sum(nil))

	requestData.Signature, err = key.Sign(hash)
	if err != nil {
		t.Fatalf("Failed to generate signature : %s", err)
	}

	b, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Failed to serialize request data : %s", err)
	}
	requestBuf := bytes.NewBuffer(b)
	request, err := http.NewRequest("POST", "/oracle/addXPub", requestBuf)
	if err != nil {
		t.Fatalf("Failed to create request : %s", err)
	}

	app := web.New(test.WebConfig, mid.ErrorHandler, mid.CORS)
	app.Handle("POST", "/oracle/addXPub", handler.AddXPub)

	requestLogger := mid.NewRequestLoggingMiddleware(logger.NewConfig(false, false, ""))
	webHandler := requestLogger.Handler(app)

	// create a ResponseRecorder to record the response.
	rr := httptest.NewRecorder()

	webHandler.ServeHTTP(rr, request)

	if rr.Code != http.StatusNotFound {
		t.Errorf("got %v want %v with body %s", rr.Code, http.StatusNotFound, rr.Body.Bytes())
	}
}

func TestAddXPubBadSignature(t *testing.T) {
	ctx := tests.Context()
	test := tests.New()

	oracleKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate oracle key : %s", err)
	}
	handler := &Oracle{
		Config:   test.WebConfig,
		MasterDB: test.MasterDB,
		Key:      oracleKey,
	}

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate user key : %s", err)
	}

	entity := actions.EntityField{
		Name:        "Test Entity Name",
		CountryCode: "AUS",
	}

	entityBytes, err := proto.Marshal(&entity)
	if err != nil {
		t.Fatalf("Failed to serialize user entity : %s", err)
	}

	userID := uuid.New()

	user := &oracle.User{
		ID:           userID.String(),
		Entity:       entityBytes,
		PublicKey:    key.PublicKey(),
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		IsDeleted:    false,
	}

	if err := oracle.CreateUser(ctx, test.MasterDB, user); err != nil {
		t.Fatalf("Failed to create user : %s", err)
	}

	xkey, err := bitcoin.GenerateMasterExtendedKey()
	if err != nil {
		t.Fatalf("Failed to generate xkey : %s", err)
	}

	xkeys := bitcoin.ExtendedKeys{xkey}

	requestData := struct {
		UserID          string               `json:"user_id" validate:"required"`
		XPubs           bitcoin.ExtendedKeys `json:"xpubs" validate:"required"` // hex
		RequiredSigners int                  `json:"required_signers" validate:"required"`
		Signature       bitcoin.Signature    `json:"signature" validate:"required"` // hex signature of user id and xpub with users public key
	}{
		UserID:          userID.String(),
		XPubs:           xkeys.ExtendedPublicKeys(),
		RequiredSigners: 1,
	}

	s := sha256.New()
	s.Write(userID[:])
	s.Write(requestData.XPubs.Bytes())
	if err := binary.Write(s, binary.LittleEndian, uint32(requestData.RequiredSigners)); err != nil {
		t.Fatalf("Failed to hash required signers : %s", err)
	}
	hash := sha256.Sum256(s.Sum(nil))

	// Sign with wrong key
	otherKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate user key : %s", err)
	}

	requestData.Signature, err = otherKey.Sign(hash)
	if err != nil {
		t.Fatalf("Failed to generate signature : %s", err)
	}

	b, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Failed to serialize request data : %s", err)
	}
	requestBuf := bytes.NewBuffer(b)
	request, err := http.NewRequest("POST", "/oracle/addXPub", requestBuf)
	if err != nil {
		t.Fatalf("Failed to create request : %s", err)
	}

	app := web.New(test.WebConfig, mid.ErrorHandler, mid.CORS)
	app.Handle("POST", "/oracle/addXPub", handler.AddXPub)

	requestLogger := mid.NewRequestLoggingMiddleware(logger.NewConfig(false, false, ""))
	webHandler := requestLogger.Handler(app)

	// create a ResponseRecorder to record the response.
	rr := httptest.NewRecorder()

	webHandler.ServeHTTP(rr, request)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %v want %v with body %s", rr.Code, http.StatusNotFound, rr.Body.Bytes())
	}
}

func TestUpdateEntity(t *testing.T) {
	ctx := tests.Context()
	test := tests.New()

	oracleKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate oracle key : %s", err)
	}
	handler := &Oracle{
		Config:   test.WebConfig,
		MasterDB: test.MasterDB,
		Key:      oracleKey,
	}

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate user key : %s", err)
	}

	entity := &actions.EntityField{
		Name:        "Test Entity Name",
		CountryCode: "AUS",
	}

	entityBytes, err := proto.Marshal(entity)
	if err != nil {
		t.Fatalf("Failed to serialize user entity : %s", err)
	}

	userID := uuid.New()

	user := &oracle.User{
		ID:           userID.String(),
		Entity:       entityBytes,
		PublicKey:    key.PublicKey(),
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		IsDeleted:    false,
	}

	if err := oracle.CreateUser(ctx, test.MasterDB, user); err != nil {
		t.Fatalf("Failed to create user : %s", err)
	}

	requestData := struct {
		UserID    string              `json:"user_id" validate:"required"`
		Entity    actions.EntityField `json:"entity" validate:"required"`
		Signature bitcoin.Signature   `json:"signature" validate:"required"`
	}{
		UserID: user.ID,
		Entity: *entity,
	}

	requestData.Entity.CountryCode = "USA"

	// Sign entity
	s := sha256.New()
	if _, err := s.Write(userID[:]); err != nil {
		t.Fatalf("Failed to write user id : %s", err)
	}
	if err := requestData.Entity.WriteDeterministic(s); err != nil {
		t.Fatalf("Failed to write entity : %s", err)
	}
	hash := sha256.Sum256(s.Sum(nil))

	requestData.Signature, err = key.Sign(hash)
	if err != nil {
		t.Fatalf("Failed to generate signature : %s", err)
	}

	b, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Failed to serialize request data : %s", err)
	}
	requestBuf := bytes.NewBuffer(b)
	request, err := http.NewRequest("POST", "http://test.com/updateEntity", requestBuf)
	if err != nil {
		t.Fatalf("Failed to create request : %s", err)
	}

	params := map[string]string{}

	response := &MockResponseWriter{
		header: http.Header{},
	}

	// Update Identity
	err = handler.UpdateIdentity(ctx, response, request, params)
	if err != nil {
		t.Fatalf("Failed to update identity : %s", err)
	}

	if response.StatusCode != 200 {
		t.Fatalf("Response is not success : %d", response.StatusCode)
	}

	ruser, err := oracle.FetchUser(ctx, test.MasterDB, requestData.UserID)
	if err != nil {
		t.Fatalf("Failed to fetch user : %s", err)
	}

	rentity := &actions.EntityField{}
	if err := proto.Unmarshal(ruser.Entity, rentity); err != nil {
		t.Fatalf("Failed to unmarshal entity : %s", err)
	}

	if !rentity.Equal(&requestData.Entity) {
		t.Errorf("Wrong entity : \n  got  %+v\n  want %+v", *rentity, requestData.Entity)
	}

	if rentity.CountryCode != "USA" {
		t.Errorf("Wrong country code : got %s, want %s", rentity.CountryCode, "USA")
	}
}
