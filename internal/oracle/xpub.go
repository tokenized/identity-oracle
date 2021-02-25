package oracle

import (
	"context"

	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/pkg/bitcoin"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	XPubColumns = `
		xp.id,
		xp.user_id,
		xp.xpub,
		xp.required_signers,
		xp.date_created
	`
)

// CreateXPub inserts an extended public key into the database.
func CreateXPub(ctx context.Context, dbConn *db.DB, xpub *XPub) error {
	sql := `INSERT
		INTO xpubs (
			id,
			user_id,
			xpub,
			required_signers,
			date_created
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT ON CONSTRAINT xpubs_unique DO NOTHING`

	xpub.ID = uuid.New().String()

	if err := dbConn.Execute(ctx, sql,
		xpub.ID,
		xpub.UserID,
		xpub.XPub,
		xpub.RequiredSigners,
		xpub.DateCreated); err != nil {
		return err
	}

	return nil
}

func FetchXPubByXPub(ctx context.Context, dbConn *db.DB,
	xpubs bitcoin.ExtendedKeys) (*XPub, error) {

	sql := `SELECT ` + XPubColumns + `
		FROM
			xpubs xp
		WHERE
			xp.xpub = ?`

	result := &XPub{}
	if err := dbConn.Get(ctx, result, sql, xpubs); err != nil {
		if errors.Cause(err) == db.ErrNotFound {
			return nil, errors.Wrap(ErrXPubNotFound, xpubs.String())
		}
		return nil, err
	}
	return result, nil
}

func FetchUserIDByXPub(ctx context.Context, dbConn *db.DB,
	xpubs bitcoin.ExtendedKeys) (*string, error) {

	sql := `SELECT user_id
		FROM
			xpubs
		WHERE
			xpubs.xpub = ?`

	var result string
	if err := dbConn.Get(ctx, &result, sql, xpubs); err != nil {
		if errors.Cause(err) == db.ErrNotFound {
			return nil, errors.Wrap(ErrXPubNotFound, xpubs.String())
		}
		return nil, err
	}
	return &result, nil
}
