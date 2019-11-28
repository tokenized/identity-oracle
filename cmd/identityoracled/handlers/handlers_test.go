package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/tests"
	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

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
		Entity: actions.EntityField{
			Name: "Test Oracle",
		},
	}

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate user key : %s", err)
	}

	entity := actions.EntityField{
		Name: "Test Handler Entity Name",
	}

	entityBytes, err := proto.Marshal(&entity)
	if err != nil {
		t.Fatalf("Failed to serialize user entity : %s", err)
	}

	requestData := struct {
		Entity       string `json:"entity" validate:"required"`     // hex protobuf
		PublicKey    string `json:"public_key" validate:"required"` // hex compressed
		Jurisdiction string `json:"jurisdiction"`
	}{
		Entity:       hex.EncodeToString(entityBytes),
		PublicKey:    hex.EncodeToString(key.PublicKey().Bytes()),
		Jurisdiction: "AUS",
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

	err = handler.Register(ctx, test.Log, response, request, params)
	if err != nil {
		t.Fatalf("Failed to register : %s", err)
	}

	if response.StatusCode != 200 {
		t.Fatalf("Response is not success : %d", response.StatusCode)
	}

	var responseData struct {
		Status string `json:"status"`
		UserID string `json:"user_id"`
	}

	if err := web.Unmarshal(&response.buffer, &responseData); err != nil {
		t.Fatalf("Failed to unmarshal response : %s", err)
	}

	t.Logf("Status  : %s", responseData.Status)
	t.Logf("User ID : %s", responseData.UserID)
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
		Entity: actions.EntityField{
			Name: "Test Oracle",
		},
	}

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate user key : %s", err)
	}

	entity := actions.EntityField{
		Name: "Test Entity Name",
	}

	entityBytes, err := proto.Marshal(&entity)
	if err != nil {
		t.Fatalf("Failed to serialize user entity : %s", err)
	}

	user := oracle.User{
		ID:           uuid.New().String(),
		Entity:       entityBytes,
		PublicKey:    key.PublicKey(),
		Jurisdiction: "AUS",
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
		UserID          string `json:"user_id" validate:"required"`
		XPub            string `json:"xpub" validate:"required"` // hex
		RequiredSigners int    `json:"required_signers" validate:"required"`
		Signature       string `json:"signature" validate:"required"` // hex signature of user id and xpub with users public key
	}{
		UserID:          user.ID,
		XPub:            xkeys.ExtendedPublicKeys().String(),
		RequiredSigners: 1,
	}

	hash := bitcoin.DoubleSha256([]byte(user.ID + requestData.XPub))

	sig, err := key.Sign(hash)
	if err != nil {
		t.Fatalf("Failed to generate signature : %s", err)
	}
	requestData.Signature = hex.EncodeToString(sig.Bytes())

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
	err = handler.AddXPub(ctx, test.Log, response, request, params)
	if err != nil {
		t.Fatalf("Failed to add xpub : %s", err)
	}

	if response.StatusCode != 200 {
		t.Fatalf("Response is not success : %d", response.StatusCode)
	}
}
