package oracle

import (
	"context"
	"errors"

	"github.com/tokenized/identity-oracle/internal/platform/db"
)

const (
	XPubColumns = `
		xp.id,
		xp.xpub,
		xp.signers,
		xp.placeholder,
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
			date_created
		)
		VALUES (?, ?, ?, ?)`

	if err := dbConn.Execute(ctx, sql,
		xpub.ID,
		xpub.UserID,
		xpub.XPub,
		xpub.DateCreated); err != nil {
		return err
	}

	return nil
}
