package oracle

import (
	"context"
	"time"

	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/pkg/bitcoin"
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
		u.is_deleted`
)

// CreateUser inserts a user into the database.
func CreateUser(ctx context.Context, dbConn *db.DB, user *User) error {
	sql := `INSERT
		INTO users (
			id,
			entity,
			public_key,
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
		user.PublicKey,
		user.DateCreated,
		user.DateModified,
		user.IsDeleted); err != nil {
		return err
	}

	return nil
}

func FetchUser(ctx context.Context, dbConn *db.DB, id string) (*User, error) {
	sql := `SELECT ` + UserColumns + ` FROM users u WHERE u.id=? AND u.is_deleted=false`

	user := &User{}
	if err := dbConn.Get(ctx, user, sql, id); err != nil {
		if errors.Cause(err) == db.ErrNotFound {
			return nil, errors.Wrap(ErrUserNotFound, id)
		}
		return nil, err
	}
	return user, nil
}

func FetchUserByXPub(ctx context.Context, dbConn *db.DB, xpubs bitcoin.ExtendedKeys) (*User, error) {
	sql := `SELECT ` + UserColumns + `
		FROM
			users u,
			xpubs
		WHERE
			xpubs.xpub = ?
			AND xpubs.user_id=u.id
			AND u.is_deleted=false`

	user := &User{}
	if err := dbConn.Get(ctx, user, sql, xpubs); err != nil {
		if errors.Cause(err) == db.ErrNotFound {
			return nil, errors.Wrap(ErrXPubNotFound, xpubs.String())
		}
		return nil, err
	}
	return user, nil
}

func UpdateUser(ctx context.Context, dbConn *db.DB, user *User) error {
	sql := `UPDATE users SET entity=$2, date_modified=$3 WHERE id=$1`

	// Verify entity format
	entity := &actions.EntityField{}
	if err := proto.Unmarshal(user.Entity, entity); err != nil {
		return errors.Wrap(err, "deserialize entity")
	}

	user.DateModified = time.Now()
	if err := dbConn.Execute(ctx, sql,
		user.ID,
		user.Entity,
		user.DateModified); err != nil {
		return err
	}

	return nil
}
