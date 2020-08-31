package oracle

import (
	"time"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID         `db:"id" json:"id"`
	Entity       []byte            `db:"entity" json:"entity"`
	PublicKey    bitcoin.PublicKey `db:"public_key" json:"public_key"`
	DateCreated  time.Time         `db:"date_created" json:"date_created"`
	DateModified time.Time         `db:"date_modified" json:"date_modified"`
	Approved     bool              `db:"approved" json:"approved"`
	IsDeleted    bool              `db:"is_deleted" json:"is_deleted"`
}

type XPub struct {
	ID              uuid.UUID            `db:"id" json:"id"`
	UserID          uuid.UUID            `db:"user_id" json:"user_id"`
	XPub            bitcoin.ExtendedKeys `db:"xpub" json:"xpub"`
	RequiredSigners int                  `json:"required_signers" db:"required_signers"`
	DateCreated     time.Time            `db:"date_created" json:"date_created"`
}

// SignatureHash is a simple struct for wrapping the common values returned from a function that
// calculates a signature hash.
type SignatureHash struct {
	Hash        []byte
	BlockHeight uint32
	Approved    bool
	Description string
}
