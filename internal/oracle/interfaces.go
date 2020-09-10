package oracle

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
)

type ApproverInterface interface {
	// ApproveRegistration approves the registration of a new user.
	// userID is the user id the user will have if registration is approved.
	// Returns:
	//   bool - approved
	//   string - description of approval or rejection
	//   error - error
	// 0, nil means approved.
	// anything except zero will be returned to the user as an http status code with the error text.
	ApproveRegistration(ctx context.Context, userID string, entity actions.EntityField,
		publicKey bitcoin.PublicKey) (int, string, error)

	// ApproveTransfer approves the receive of a token.
	// Returns:
	//   bool - approved
	//   string - description of approval or rejection
	//   error - error
	// 0, nil means approved.
	// An error aborts the process and returns an error to the user. If error is nil then a
	// signature will be returned to the user, though it won't indicate approval unless specified.
	ApproveTransfer(ctx context.Context, contract, assetID string,
		userID string) (bool, string, error)

	// ApproveIdentity approves that an identity is verified and ready to use.
	// Returns:
	//   bool - approved
	//   string - description of approval or rejection
	//   error - error
	// 0, nil means approved.
	// anything except zero will be returned to the user as an http status code with the error text.
	ApproveIdentity(ctx context.Context, userID string) (int, string, error)
}
