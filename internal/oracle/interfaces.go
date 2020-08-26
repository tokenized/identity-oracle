package oracle

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/google/uuid"
)

type ApproverInterface interface {
	// ApproveRegistration approves the registration of a new user.
	// 0, nil means approved.
	// anything except zero will be returned to the user as an http status code with the error text.
	ApproveRegistration(ctx context.Context, entity actions.EntityField,
		publicKey bitcoin.PublicKey) (int, error)

	// ApproveTransfer approves the receive of a token.
	// Returns:
	//   bool - approved
	//   string - description of approval or rejection
	//   error - error
	// An error aborts the process and returns an error to the user. If error is nil then a
	// signature will be returned to the user, though it won't indicate approval unless specified.
	ApproveTransfer(ctx context.Context, contract, assetID string,
		userID uuid.UUID) (bool, string, error)
}
