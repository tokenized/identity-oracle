package oracle

import (
	"testing"
	"time"

	"github.com/tokenized/identity-oracle/internal/platform/tests"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

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
		Name: "Test Entity Name",
	}

	entityBytes, err := proto.Marshal(&entity)
	if err != nil {
		t.Fatalf("Failed to serialize user entity : %s", err)
	}

	user := &User{
		ID:           uuid.New().String(),
		Entity:       entityBytes,
		PublicKey:    key.PublicKey(),
		Jurisdiction: "AUS",
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		IsDeleted:    false,
	}

	if err := CreateUser(ctx, test.MasterDB, user); err != nil {
		t.Fatalf("Failed to create user : %s", err)
	}
}
