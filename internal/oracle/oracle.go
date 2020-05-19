package oracle

import (
	"context"

	"github.com/tokenized/identity-oracle/internal/platform/config"
	"github.com/tokenized/identity-oracle/internal/platform/db"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

// ApproveTransfer determines if a transfer is approved.
// Returns:
//   []byte - signature hash
//   uint32 - block height of block hash included in signature hash
//   bitcoin.Hash32 - block hash included in signature hash
//   bool - true if transfer is approved
func ApproveTransfer(ctx context.Context, dbConn *db.DB, blockHandler *BlockHandler,
	contract, asset string, xpub bitcoin.ExtendedKeys, index uint32,
	quantity uint64) ([]byte, uint32, bitcoin.Hash32, bool, error) {

	_, assetCode, err := protocol.DecodeAssetID(asset)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, false, errors.Wrap(err, "decode asset id")
	}

	// TODO Get contract and asset

	xpubData, err := FetchXPubByXPub(ctx, dbConn, xpub)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, false, errors.Wrap(err, "fetch xpub")
	}

	_, err = FetchUserByXPub(ctx, dbConn, xpub)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, false, errors.Wrap(err, "fetch user")
	}

	// TODO Verify user meets criteria
	approved := true
	approveValue := uint8(1)

	// Dev reject testing
	testValues := config.ContextTestValues(ctx)
	if testValues.RejectQuantity != 0 && testValues.RejectQuantity == quantity {
		approved = false
		approveValue = 0
	}

	contractAddress, err := bitcoin.DecodeAddress(contract)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, false, errors.Wrap(err, "decode contract address")
	}
	contractRawAddress := bitcoin.NewRawAddressFromAddress(contractAddress)

	// Get block hash for tip - 4
	blockHash, height, err := blockHandler.SigHash(ctx)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, false, errors.Wrap(err, "get sig block hash")
	}

	// Generate address at index
	addressKey, err := xpub.ChildKeys(index)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, false, errors.Wrap(err, "generate address key")
	}

	receiveAddress, err := addressKey.RawAddress(xpubData.RequiredSigners)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, false, errors.Wrap(err, "generate address")
	}

	sigHash, err := protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, quantity, &blockHash, approveValue)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, false, errors.Wrap(err, "generate signature")
	}

	return sigHash, height, blockHash, approved, nil
}
