package oracle

import (
	"context"

	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

// CreateReceiveSignature creates a token receive signature.
// Returns:
//   []byte - signature hash
//   uint32 - block height of block hash included in signature hash
//   bitcoin.Hash32 - block hash included in signature hash
//   bool - true if transfer is approved
func CreateReceiveSignature(ctx context.Context, dbConn *db.DB, blockHandler *BlockHandler,
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
	blockHash, height, err := blockHandler.SigHash(ctx)
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
		receiveAddress, blockHash, expiration, approveValue)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate signature")
	}

	return sigHash, height, blockHash, nil
}

// CreateAdminCertificate creates an admin certificate for contract offers.
// Returns:
//   []byte - signature hash
//   uint32 - block height of block hash included in signature hash
//   bitcoin.Hash32 - block hash included in signature hash
//   bool - true if approved
func CreateAdminCertificate(ctx context.Context, dbConn *db.DB, blockHandler *BlockHandler,
	xpubs bitcoin.ExtendedKeys, index uint32, issuer actions.EntityField,
	entityContract bitcoin.RawAddress, expiration uint64, approved bool) ([]byte, uint32, bitcoin.Hash32, error) {

	xpubData, err := FetchXPubByXPub(ctx, dbConn, xpubs)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "fetch xpub")
	}

	adminKey, err := xpubs.ChildKeys(index)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate address key")
	}

	adminAddress, err := adminKey.RawAddress(xpubData.RequiredSigners)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate address")
	}

	approveValue := uint8(1)
	if !approved {
		approveValue = 0
	}

	// Get block hash for tip - 4
	blockHash, height, err := blockHandler.SigHash(ctx)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "get sig block hash")
	}

	var entity interface{}
	if entityContract.IsEmpty() {
		entity = issuer
	} else {
		entity = entityContract
	}

	sigHash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, adminAddress, entity,
		blockHash, expiration, approveValue)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate sig hash")
	}

	return sigHash, height, blockHash, nil
}
