package oracle

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/storage"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

const (
	// contractsStorageKey is the path to the contract formations.
	contractsStorageKey = "contract_formations"
)

// ContractsManager implements the ContractProcessor interface for the contracts package. It saves
// contract formation actions to storage.
type ContractsManager struct {
	st     storage.Storage
	net    bitcoin.Network
	isTest bool
}

// NewContractsManager creates a ContractsManager.
func NewContractsManager(st storage.Storage, net bitcoin.Network, isTest bool) *ContractsManager {
	return &ContractsManager{st, net, isTest}
}

// SaveContractFormation saves a contract formation to storage.
func (cm *ContractsManager) SaveContractFormation(ctx context.Context, ra bitcoin.RawAddress,
	script []byte) error {

	key := strings.Join([]string{contractsStorageKey, hex.EncodeToString(ra.Bytes())}, "/")

	// Check for pre-existing
	b, err := cm.st.Read(ctx, key)
	if err != nil {
		if errors.Cause(err) != storage.ErrNotFound {
			return errors.Wrap(err, "read contract formation")
		}

		// ErrNotFound, First version of this contract formation
		logger.Info(ctx, "Saving contract formation : %s : %x",
			bitcoin.NewAddressFromRawAddress(ra, cm.net).String(), script)
		if err := cm.st.Write(ctx, key, script, nil); err != nil {
			return errors.Wrap(err, "write contract formation")
		}
	}

	// Check timestamp vs current version to ensure we keep the latest.
	action, err := protocol.Deserialize(b, cm.isTest)
	if err != nil {
		// Overwrite invalid contract formation
		logger.Warn(ctx, "Overwrite invalid contract formation : %s : %x",
			bitcoin.NewAddressFromRawAddress(ra, cm.net).String(), script)
		if err := cm.st.Write(ctx, key, script, nil); err != nil {
			return errors.Wrap(err, "write contract formation")
		}

		return nil
	}

	current, ok := action.(*actions.ContractFormation)
	if !ok {
		// Overwrite invalid contract formation
		logger.Warn(ctx, "Overwrite non contract formation : %s : %x",
			bitcoin.NewAddressFromRawAddress(ra, cm.net).String(), script)
		if err := cm.st.Write(ctx, key, script, nil); err != nil {
			return errors.Wrap(err, "write contract formation")
		}

		return nil
	}

	action, err = protocol.Deserialize(script, cm.isTest)
	if err != nil {
		return errors.Wrap(err, "parse contract formation")
	}

	new, nok := action.(*actions.ContractFormation)
	if !nok {
		return errors.Wrap(err, "not contract formation")
	}

	if current.Timestamp > new.Timestamp {
		return nil // already have a later version
	}

	logger.Info(ctx, "Updating contract formation : %s : %x",
		bitcoin.NewAddressFromRawAddress(ra, cm.net).String(), script)
	if err := cm.st.Write(ctx, key, script, nil); err != nil {
		return errors.Wrap(err, "write contract formation")
	}

	return nil
}

// GetContractFormation retrieves a contract formation from storage.
func GetContractFormation(ctx context.Context, dbConn *db.DB, ra bitcoin.RawAddress,
	isTest bool) (*actions.ContractFormation, error) {

	key := strings.Join([]string{contractsStorageKey, hex.EncodeToString(ra.Bytes())}, "/")

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err, "fetch contract formation")
	}

	action, err := protocol.Deserialize(b, isTest)
	if err != nil {
		return nil, errors.Wrap(err, "deserialize contract formation")
	}

	result, ok := action.(*actions.ContractFormation)
	if !ok {
		return nil, errors.New("Not contract formation")
	}

	return result, nil
}
