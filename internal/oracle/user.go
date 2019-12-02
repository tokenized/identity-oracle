package oracle

import (
	"context"

	"github.com/tokenized/identity-oracle/internal/platform/db"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	UserColumns = `
		u.id,
		u.entity,
		u.public_key,
		u.date_created,
		u.date_modified,
		u.approved,
		u.is_deleted`
)

// CreateUser inserts a user into the database.
func CreateUser(ctx context.Context, dbConn *db.DB, user User) error {
	sql := `INSERT
		INTO users (
		    id,
		    entity,
		    public_key,
		    date_created,
		    date_modified,
		    approved,
		    is_deleted
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	// Verify entity format
	entity := &actions.EntityField{}
	if err := proto.Unmarshal(user.Entity, entity); err != nil {
		return errors.Wrap(err, "deserialize entity")
	}

	if err := dbConn.Execute(ctx, sql,
		user.ID,
		user.Entity,
		user.PublicKey,
		user.DateCreated,
		user.DateModified,
		user.Approved,
		user.IsDeleted); err != nil {
		return err
	}

	return nil
}

func FetchUser(ctx context.Context, dbConn *db.DB, id string) (User, error) {
	sql := `SELECT ` + UserColumns + `
		FROM
			users u
		WHERE
			u.id=?`

	user := User{}
	err := dbConn.Get(ctx, &user, sql, id)
	return user, err
}

func FetchUserByXPub(ctx context.Context, dbConn *db.DB, xpub bitcoin.ExtendedKeys) (User, error) {
	sql := `SELECT ` + UserColumns + `
		FROM
			users u,
			xpubs
		WHERE
			xpubs.xpub = ?
			AND xpubs.user_id=u.id`

	user := User{}
	err := dbConn.Get(ctx, &user, sql, xpub)
	if err == db.ErrNotFound {
		err = ErrXPubNotFound
	}
	return user, err
}
