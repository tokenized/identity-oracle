package oracle

import (
	"context"
	"errors"

	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/pkg/bitcoin"

	"github.com/google/uuid"
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

var (
	ErrXPubNotFound = errors.New("Extended Public Key Not Found")
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

	xpub.ID = uuid.New()

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

func FetchXPubByXPub(ctx context.Context, dbConn *db.DB, xpub bitcoin.ExtendedKeys) (XPub, error) {
	sql := `SELECT ` + XPubColumns + `
		FROM
			xpubs xp
		WHERE
			xp.xpub = ?`

	result := XPub{}
	err := dbConn.Get(ctx, &result, sql, xpub)
	if err == db.ErrNotFound {
		err = ErrXPubNotFound
	}
	return result, err
}

func FetchUserIDByXPub(ctx context.Context, dbConn *db.DB, xpub bitcoin.ExtendedKeys) (uuid.UUID, error) {
	sql := `SELECT user_id
		FROM
			xpubs
		WHERE
			xpubs.xpub = ?`

	var result uuid.UUID
	err := dbConn.Get(ctx, &result, sql, xpub)
	if err == db.ErrNotFound {
		err = ErrXPubNotFound
	}
	return result, err
}
