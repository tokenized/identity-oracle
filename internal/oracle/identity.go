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

func VerifyPubKey(ctx context.Context, user *User, headers Headers,
	entity *actions.EntityField, xpub bitcoin.ExtendedKey, index uint32) (*SignatureHash, error) {

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
	blockHash, height, err := headers.RecentSigHash(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get sig block hash")
	}

	// Generate public key at index
	xpubKey, err := xpub.ChildKey(index)
	if err != nil {
		return nil, errors.Wrap(err, "generate public key")
	}

	pubKey := xpubKey.PublicKey()

	hash, err := protocol.EntityPubKeyOracleSigHash(ctx, entity, pubKey, *blockHash, approve)
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

func VerifyXPub(ctx context.Context, user *User, headers Headers,
	entity *actions.EntityField, xpub bitcoin.ExtendedKeys) (*SignatureHash, error) {

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
	blockHash, height, err := headers.RecentSigHash(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get sig block hash")
	}

	hash, err := protocol.EntityXPubOracleSigHash(ctx, entity, xpub, *blockHash, approve)
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
func CreateAdminCertificate(ctx context.Context, dbConn *db.DB, user *User, net bitcoin.Network,
	isTest bool, headers Headers, contracts Contracts, xpubs bitcoin.ExtendedKeys, index uint32,
	issuer actions.EntityField, entityContract bitcoin.RawAddress,
	expiration uint64) (*SignatureHash, error) {

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
	blockHash, height, err := headers.RecentSigHash(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get sig block hash")
	}

	fields := []logger.Field{
		logger.Stringer("admin_address", bitcoin.NewAddressFromRawAddress(adminAddress, net)),
	}

	approved := true
	approve := uint8(1)
	var description string
	var entity interface{}
	var checkEntity *actions.EntityField
	if entityContract.IsEmpty() {
		entity = &issuer // Must be a pointer
		checkEntity = &issuer
		fields = append(fields, logger.JSON("issuer", &issuer))
	} else {
		entity = entityContract
		fields = append(fields, logger.Stringer("entity_contract",
			bitcoin.NewAddressFromRawAddress(entityContract, net)))

		// Verify the contract belongs to the user.
		cf, err := contracts.GetContractFormation(ctx, entityContract)
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

	fields = append(fields, logger.Stringer("block_hash", blockHash))
	fields = append(fields, logger.Uint64("expiration", expiration))
	fields = append(fields, logger.Uint8("approved", approve))

	hash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, adminAddress, entity, *blockHash,
		expiration, approve)
	if err != nil {
		return nil, errors.Wrap(err, "generate sig hash")
	}

	hashObject, err := bitcoin.NewHash32(hash)
	if err == nil {
		fields = append(fields, logger.Stringer("sig_hash", hashObject))
	} else {
		fields = append(fields, logger.String("sig_hash", "invalid"))
	}

	logger.InfoWithFields(ctx, fields, "Admin certificate")

	return &SignatureHash{
		Hash:        hash,
		BlockHeight: height,
		Approved:    approved,
		Description: description,
	}, nil
}
