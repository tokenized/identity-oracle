package oracle

import (
	"context"

	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

// CreateReceiveSignature creates a token receive signature.
// Returns:
//   []byte - signature hash
//   uint32 - block height of block hash included in signature hash
//   bitcoin.Hash32 - block hash included in signature hash
//   bool - true if transfer is approved
func CreateReceiveSignature(ctx context.Context, dbConn *db.DB, headers Headers,
	contract, asset string, xpubs bitcoin.ExtendedKeys, index uint32, expiration uint64,
	approved bool) ([]byte, uint32, bitcoin.Hash32, error) {

	_, assetCode, err := protocol.DecodeAssetID(asset)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "decode asset id")
	}

	// TODO Get contract and asset

	xpubData, err := FetchXPubByXPub(ctx, dbConn, xpubs)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "fetch xpub")
	}

	approveValue := uint8(1)
	if !approved {
		approveValue = 0
	}

	contractAddress, err := bitcoin.DecodeAddress(contract)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "decode contract address")
	}
	contractRawAddress := bitcoin.NewRawAddressFromAddress(contractAddress)

	// Get block hash for tip - 4
	blockHash, height, err := headers.RecentSigHash(ctx)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "get sig block hash")
	}

	// Generate address at index
	addressKey, err := xpubs.ChildKeys(index)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate address key")
	}

	receiveAddress, err := addressKey.RawAddress(xpubData.RequiredSigners)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate address")
	}

	sigHash, err := protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, *blockHash, expiration, approveValue)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate signature")
	}

	return sigHash, height, *blockHash, nil
}
