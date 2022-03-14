package oracle

import (
	"context"

	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
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
	net bitcoin.Network, contract, instrument string, xpubs bitcoin.ExtendedKeys, index uint32,
	expiration uint64, approved bool) (*bitcoin.Hash32, uint32, bitcoin.Hash32, error) {

	_, instrumentCode, err := protocol.DecodeInstrumentID(instrument)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "decode instrument id")
	}

	// TODO Get contract and instrument

	xpubData, err := FetchXPubByXPub(ctx, dbConn, xpubs)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "fetch xpub")
	}

	approveValue := uint8(1)
	if !approved {
		approveValue = 0
	}
	fields := []logger.Field{
		logger.Uint8("approved", approveValue),
	}
	fields = append(fields, logger.Uint64("expiration", expiration))

	contractAddress, err := bitcoin.DecodeAddress(contract)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "decode contract address")
	}
	contractRawAddress := bitcoin.NewRawAddressFromAddress(contractAddress)
	fields = append(fields, logger.Stringer("contract_address",
		bitcoin.NewAddressFromRawAddress(contractRawAddress, net)))

	// Get block hash for tip - 4
	blockHash, height, err := headers.RecentSigHash(ctx)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "get sig block hash")
	}
	fields = append(fields, logger.Stringer("block_hash", blockHash))

	// Generate address at index
	addressKey, err := xpubs.ChildKeys(index)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate address key")
	}

	receiveAddress, err := addressKey.RawAddress(xpubData.RequiredSigners)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate address")
	}
	fields = append(fields, logger.Stringer("receive_address",
		bitcoin.NewAddressFromRawAddress(receiveAddress, net)))

	sigHash, err := protocol.TransferOracleSigHash(ctx, contractRawAddress, instrumentCode.Bytes(),
		receiveAddress, *blockHash, expiration, approveValue)
	if err != nil {
		return nil, 0, bitcoin.Hash32{}, errors.Wrap(err, "generate signature")
	}

	fields = append(fields, logger.Stringer("sig_hash", sigHash))

	logger.InfoWithFields(ctx, fields, "Transfer certificate")

	return sigHash, height, *blockHash, nil
}
