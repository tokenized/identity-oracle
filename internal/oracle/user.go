package oracle

import (
	"context"

	"github.com/pkg/errors"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/golang/protobuf/proto"
)

const (
	UserColumns = `
		u.id,
		u.entity,
		u.jurisdiction,
		u.date_created,
		u.date_modified,
		u.is_deleted`
)

// CreateUser inserts a user into the database.
func CreateUser(ctx context.Context, dbConn *db.DB, user *User) error {
	sql := `INSERT
		INTO xpubs (
		    id
		    entity,
		    jurisdiction,
		    date_created,
		    date_modified,
		    is_deleted
		)
		VALUES (?, ?, ?, ?, ?, ?)`

	// Verify entity format
	entity := &actions.EntityField{}
	if err := proto.Unmarshal(user.Entity, entity); err != nil {
		return errors.Wrap(err, "deserialize entity")
	}

	if err := dbConn.Execute(ctx, sql,
		user.ID,
		user.Entity,
		user.Jurisdiction,
		user.DateCreated,
		user.DateModified,
		user.IsDeleted); err != nil {
		return err
	}

	return nil
}

func FetchUserByXPub(ctx context.Context, dbConn *db.DB, xpub bitcoin.ExtendedKey) (User, error) {
	sql := `SELECT ` + UserColumns + `
		FROM
			users u,
			xpubs
		WHERE
			xpubs.xpub = ?
			xpubs.user_id=u.id`

	user := User{}
	err := dbConn.Get(ctx, &user, sql, xpub)
	if err == db.ErrNotFound {
		err = ErrXPubNotFound
	}
	return user, err
}
