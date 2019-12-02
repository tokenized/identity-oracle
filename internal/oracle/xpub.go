package oracle

import (
	"context"
	"errors"

	"github.com/tokenized/identity-oracle/internal/platform/db"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
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
func CreateXPub(ctx context.Context, dbConn *db.DB, xpub XPub) error {
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
			xpubs
		WHERE
			xpubs.xpub = ?`

	result := XPub{}
	err := dbConn.Get(ctx, &result, sql, xpub)
	if err == db.ErrNotFound {
		err = ErrXPubNotFound
	}
	return result, err
}
