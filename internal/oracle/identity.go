package oracle

import (
	"context"

	"github.com/tokenized/identity-oracle/internal/platform/db"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

func VerifyPubKey(ctx context.Context, dbConn *db.DB, blockHandler *BlockHandler,
	entity *actions.EntityField, xpub bitcoin.ExtendedKeys, index uint32) ([]byte, uint32, bool, error) {

	user, err := FetchUserByXPub(ctx, dbConn, xpub)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "fetch user")
	}

	userEntity := &actions.EntityField{}
	if err := proto.Unmarshal(user.Entity, userEntity); err != nil {
		return nil, 0, false, errors.Wrap(err, "unmarshal user entity")
	}

	// Verify the entity matches that registered to the user.
	approve := uint8(1)
	if !entity.Equal(userEntity) {
		approve = 0
	}

	// Get block hash for tip - 4
	blockHash, height, err := blockHandler.SigHash(ctx)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "get sig block hash")
	}

	// Generate public key at index
	xpubKeys, err := xpub.ChildKeys(index)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "generate public key")
	}

	if len(xpubKeys) > 1 {
		return nil, 0, false, errors.Wrap(err, "multi-key not supported")
	}

	pubKey := xpubKeys[0].PublicKey()

	sig, err := protocol.EntityPubKeyOracleSigHash(ctx, entity, pubKey, &blockHash, approve)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "generate signature")
	}

	return sig, height, true, nil
}

func VerifyXPub(ctx context.Context, dbConn *db.DB, blockHandler *BlockHandler,
	entity *actions.EntityField, xpub bitcoin.ExtendedKeys) ([]byte, uint32, bool, error) {

	user, err := FetchUserByXPub(ctx, dbConn, xpub)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "fetch user")
	}

	userEntity := &actions.EntityField{}
	if err := proto.Unmarshal(user.Entity, userEntity); err != nil {
		return nil, 0, false, errors.Wrap(err, "unmarshal user entity")
	}

	// Verify the entity matches that registered to the user.
	approve := uint8(1)
	if !entity.Equal(userEntity) {
		approve = 0
	}

	// Get block hash for tip - 4
	blockHash, height, err := blockHandler.SigHash(ctx)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "get sig block hash")
	}

	sig, err := protocol.EntityXPubOracleSigHash(ctx, entity, xpub, &blockHash, approve)
	if err != nil {
		return nil, 0, false, errors.Wrap(err, "generate signature")
	}

	return sig, height, true, nil
}
