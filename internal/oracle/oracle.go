package oracle

import (
	"context"

	"github.com/tokenized/identity-oracle/internal/platform/db"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

func ApproveTransfer(ctx context.Context, dbConn *db.DB, blockHandler *BlockHandler,
	user User, contract, asset string,
	xpub bitcoin.ExtendedKey, index uint32, quantity uint64) ([]byte, uint32, bool, error) {

	_, assetCode, err := protocol.DecodeAssetID(asset)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "decode asset id")
	}

	// TODO Get contract and asset
	// TODO Verify user meets criteria

	contractAddress, err := bitcoin.DecodeAddress(contract)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "decode contract address")
	}
	contractRawAddress := bitcoin.NewRawAddressFromAddress(contractAddress)

	// Get block hash for tip - 4
	blockHash, height, err := blockHandler.SigHash(ctx)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "get sig block hash")
	}

	// Generate address at index
	addressKey, err := xpub.ChildKey(index)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "generate address key")
	}

	receiveAddress, err := addressKey.RawAddress()

	sig, err := protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, quantity, &blockHash, 1)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "generate signature")
	}

	return sig, height, true, nil
}
