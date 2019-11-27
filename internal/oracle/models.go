package oracle

import (
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

type User struct {
	ID           string            `db:"id" json:"id"`
	Entity       []byte            `db:"entity" json:"entity"`
	PublicKey    bitcoin.PublicKey `db:"public_key" json:"public_key"`
	Jurisdiction string            `db:"jurisdiction" json:"jurisdiction"`
	DateCreated  time.Time         `db:"date_created" json:"date_created"`
	DateModified time.Time         `db:"date_modified" json:"date_modified"`
	Approved     bool              `db:"approved" json:"approved"`
	IsDeleted    bool              `db:"is_deleted" json:"is_deleted"`
}

type XPub struct {
	ID          string              `db:"id" json:"id"`
	UserID      string              `db:"user_id" json:"user_id"`
	XPub        bitcoin.ExtendedKey `db:"xpub" json:"xpub"`
	DateCreated time.Time           `db:"date_created" json:"date_created"`
}
