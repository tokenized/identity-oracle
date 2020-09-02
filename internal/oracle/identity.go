package oracle

import (
	"context"

	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

func VerifyPubKey(ctx context.Context, dbConn *db.DB, blockHandler *BlockHandler,
	entity *actions.EntityField, xpub bitcoin.ExtendedKey, index uint32) (*SignatureHash, error) {

	user, err := FetchUserByXPub(ctx, dbConn, bitcoin.ExtendedKeys{xpub})
	if err != nil {
		return nil, errors.Wrap(err, "fetch user")
	}

	userEntity := &actions.EntityField{}
	if err := proto.Unmarshal(user.Entity, userEntity); err != nil {
		return nil, errors.Wrap(err, "unmarshal user entity")
	}

	// Verify the entity matches that registered to the user.
	approved := true
	approve := uint8(1)
	var description string
	if err := VerifyEntityIsSubset(entity, userEntity); err != nil {
		description = err.Error()
		approved = false
		approve = 0
	}

	// Get block hash for tip - 4
	blockHash, height, err := blockHandler.SigHash(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get sig block hash")
	}

	// Generate public key at index
	xpubKey, err := xpub.ChildKey(index)
	if err != nil {
		return nil, errors.Wrap(err, "generate public key")
	}

	pubKey := xpubKey.PublicKey()

	hash, err := protocol.EntityPubKeyOracleSigHash(ctx, entity, pubKey, blockHash, approve)
	if err != nil {
		return nil, errors.Wrap(err, "generate signature")
	}

	return &SignatureHash{
		Hash:        hash,
		BlockHeight: height,
		Approved:    approved,
		Description: description,
	}, nil
}

func VerifyXPub(ctx context.Context, dbConn *db.DB, blockHandler *BlockHandler,
	entity *actions.EntityField, xpub bitcoin.ExtendedKeys) (*SignatureHash, error) {

	user, err := FetchUserByXPub(ctx, dbConn, xpub)
	if err != nil {
		return nil, errors.Wrap(err, "fetch user")
	}

	userEntity := &actions.EntityField{}
	if err := proto.Unmarshal(user.Entity, userEntity); err != nil {
		return nil, errors.Wrap(err, "unmarshal user entity")
	}

	// Verify the entity matches that registered to the user.
	approved := true
	approve := uint8(1)
	var description string
	if err := VerifyEntityIsSubset(entity, userEntity); err != nil {
		description = err.Error()
		approved = false
		approve = 0
	}

	// Get block hash for tip - 4
	blockHash, height, err := blockHandler.SigHash(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get sig block hash")
	}

	hash, err := protocol.EntityXPubOracleSigHash(ctx, entity, xpub, blockHash, approve)
	if err != nil {
		return nil, errors.Wrap(err, "generate signature")
	}

	return &SignatureHash{
		Hash:        hash,
		BlockHeight: height,
		Approved:    approved,
		Description: description,
	}, nil
}

// CreateAdminCertificate creates an admin certificate for contract offers.
// Returns:
//   []byte - signature hash
//   uint32 - block height of block hash included in signature hash
//   bitcoin.Hash32 - block hash included in signature hash
//   bool - true if approved
func CreateAdminCertificate(ctx context.Context, dbConn *db.DB, net bitcoin.Network, isTest bool,
	blockHandler *BlockHandler, xpubs bitcoin.ExtendedKeys, index uint32,
	issuer actions.EntityField, entityContract bitcoin.RawAddress,
	expiration uint64) (*SignatureHash, error) {

	user, err := FetchUserByXPub(ctx, dbConn, xpubs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch user")
	}

	userEntity := &actions.EntityField{}
	if err := proto.Unmarshal(user.Entity, userEntity); err != nil {
		return nil, errors.Wrap(err, "unmarshal user entity")
	}

	xpubData, err := FetchXPubByXPub(ctx, dbConn, xpubs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch xpub")
	}

	adminKey, err := xpubs.ChildKeys(index)
	if err != nil {
		return nil, errors.Wrap(err, "generate address key")
	}

	adminAddress, err := adminKey.RawAddress(xpubData.RequiredSigners)
	if err != nil {
		return nil, errors.Wrap(err, "generate address")
	}

	// Get block hash for tip - 4
	blockHash, height, err := blockHandler.SigHash(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get sig block hash")
	}

	logger.Info(ctx, "Admin Address : %s",
		bitcoin.NewAddressFromRawAddress(adminAddress, net).String())

	approved := true
	approve := uint8(1)
	var description string
	var entity interface{}
	var checkEntity *actions.EntityField
	if entityContract.IsEmpty() {
		entity = &issuer // Must be a pointer
		checkEntity = &issuer
		logger.Info(ctx, "Issuer : %+v", issuer)
	} else {
		entity = entityContract
		logger.Info(ctx, "Entity Contract : %s",
			bitcoin.NewAddressFromRawAddress(entityContract, net).String())

		// Verify the contract belongs to the user.
		cf, err := GetContractFormation(ctx, dbConn, entityContract, isTest)
		if err != nil {
			return nil, errors.Wrap(err, "get contract formation")
		}

		checkEntity = cf.Issuer
	}

	// Verify the entity matches that registered to the user.
	if err := VerifyEntityIsSubset(checkEntity, userEntity); err != nil {
		description = err.Error()
		approved = false
		approve = 0
	}

	logger.Info(ctx, "Block Hash : %s", blockHash.String())
	logger.Info(ctx, "Expiration : %d", expiration)
	logger.Info(ctx, "Approved : %d", approve)

	hash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, adminAddress, entity, blockHash,
		expiration, approve)
	if err != nil {
		return nil, errors.Wrap(err, "generate sig hash")
	}

	logger.Info(ctx, "Sig Hash : %x", hash)

	return &SignatureHash{
		Hash:        hash,
		BlockHeight: height,
		Approved:    approved,
		Description: description,
	}, nil
}
