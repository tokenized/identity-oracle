package oracle

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tokenized/identity-oracle/internal/platform/tests"
)

func TestUsers(t *testing.T) {
	ctx := tests.Context()
	test := tests.New()

	user := &User{
		ID:           uuid.New().String(),
		Entity:       nil,
		Jurisdiction: "AUS",
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		IsDeleted:    false,
	}

	if err := CreateUser(ctx, test.MasterDB, user); err != nil {
		t.Fatalf("Failed to create user : %s", err)
	}
}
