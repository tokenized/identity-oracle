package oracle

import (
	"testing"
	"time"

	"github.com/tokenized/identity-oracle/internal/platform/tests"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

func TestUsers(t *testing.T) {
	ctx := tests.Context()
	test := tests.New()

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

	user := &User{
		ID:           uuid.New().String(),
		Entity:       entityBytes,
		PublicKey:    key.PublicKey(),
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		IsDeleted:    false,
	}

	if err := CreateUser(ctx, test.MasterDB, user); err != nil {
		t.Fatalf("Failed to create user : %s", err)
	}

	fuser, err := FetchUser(ctx, test.MasterDB, user.ID)
	if err != nil {
		t.Fatalf("Failed to fetch user : %s", err)
	}

	if fuser.ID != user.ID {
		t.Fatalf("Invalid user id")
	}
}

func TestXPub(t *testing.T) {
	ctx := tests.Context()
	test := tests.New()

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

	user := &User{
		ID:           uuid.New().String(),
		Entity:       entityBytes,
		PublicKey:    key.PublicKey(),
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		IsDeleted:    false,
	}

	if err := CreateUser(ctx, test.MasterDB, user); err != nil {
		t.Fatalf("Failed to create user : %s", err)
	}

	xp, err := bitcoin.GenerateMasterExtendedKey()
	if err != nil {
		t.Fatalf("Failed to create xpub : %s", err)
	}

	xpubs := bitcoin.ExtendedKeys{xp}

	xpub := &XPub{
		UserID:          user.ID,
		XPub:            xpubs,
		RequiredSigners: 1,
		DateCreated:     time.Now(),
	}

	if err := CreateXPub(ctx, test.MasterDB, xpub); err != nil {
		t.Fatalf("Failed to create Xpub : %s", err)
	}

	fxpub, err := FetchXPubByXPub(ctx, test.MasterDB, xpubs)
	if err != nil {
		t.Fatalf("Failed to fetch xpub : %s", err)
	}

	if !fxpub.XPub.Equal(xpubs) {
		t.Fatalf("Invalid fetched xpubs")
	}

	userid, err := FetchUserIDByXPub(ctx, test.MasterDB, xpubs)
	if err != nil {
		t.Fatalf("Failed to fetch xpub user id : %s", err)
	}

	if *userid != user.ID {
		t.Fatalf("Invalid xpub user id")
	}

	fuser, err := FetchUserByXPub(ctx, test.MasterDB, xpubs)
	if err != nil {
		t.Fatalf("Failed to fetch user by xpubs : %s", err)
	}

	if fuser.ID != user.ID {
		t.Fatalf("Invalid user id")
	}
}
