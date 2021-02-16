package oracle

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
	"github.com/tokenized/spynode/pkg/client"
)

const (
	// contractsStorageKey is the path to the contract formations.
	contractsStorageKey = "contract_formations"

	chainstateStorageKey = "chainstate"
	chainstateVersion    = uint8(0)
)

type Headers interface {
	// RecentSigHash returns a header hash and height for the current tip -4
	RecentSigHash(context.Context) (*bitcoin.Hash32, uint32, error)
}

type Contracts interface {
	// GetContractFormation returns the most recent contract formation for the specified contract
	// address.
	GetContractFormation(context.Context, bitcoin.RawAddress) (*actions.ContractFormation, error)
}

type Listener struct {
	spyNode client.Client
	dbConn  *db.DB
	net     bitcoin.Network
	isTest  bool
	offset  int

	hashes     []bitcoin.Hash32
	height     uint32
	hashesLock sync.Mutex
}

func NewListener(spyNode client.Client, dbConn *db.DB, net bitcoin.Network, isTest bool) *Listener {
	return &Listener{
		spyNode: spyNode,
		dbConn:  dbConn,
		net:     net,
		isTest:  isTest,
		offset:  4,
	}
}

func (l *Listener) RecentSigHash(ctx context.Context) (*bitcoin.Hash32, uint32, error) {
	l.hashesLock.Lock()
	defer l.hashesLock.Unlock()

	if len(l.hashes) < l.offset {
		return nil, 0, errors.New("Not enough headers")
	}

	return &l.hashes[len(l.hashes)-l.offset-1], l.height - uint32(l.offset), nil
}

func (l *Listener) GetContractFormation(ctx context.Context,
	ra bitcoin.RawAddress) (*actions.ContractFormation, error) {

	key := strings.Join([]string{contractsStorageKey, hex.EncodeToString(ra.Bytes())}, "/")

	b, err := l.dbConn.Fetch(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err, "fetch contract formation")
	}

	action, err := protocol.Deserialize(b, l.isTest)
	if err != nil {
		return nil, errors.Wrap(err, "deserialize contract formation")
	}

	result, ok := action.(*actions.ContractFormation)
	if !ok {
		return nil, errors.New("Not contract formation")
	}

	return result, nil

}

func (l *Listener) HandleTx(ctx context.Context, tx *client.Tx) {
	// Only look for contract formations and save them.
	if len(tx.Outputs) == 0 {
		return
	}

	// Address of first input
	ra, err := bitcoin.RawAddressFromLockingScript(tx.Outputs[0].PkScript)
	if err != nil || ra.IsEmpty() {
		return
	}

	for _, output := range tx.Tx.TxOut {
		action, err := protocol.Deserialize(output.PkScript, l.isTest)
		if err != nil {
			continue
		}

		formation, ok := action.(*actions.ContractFormation)
		if !ok {
			continue
		}

		if err := l.SaveContractFormation(ctx, ra, formation, output.PkScript); err != nil {
			logger.Error(ctx, "Failed to save contract formation : %s", err)
		}
	}
}

func (l *Listener) HandleTxUpdate(ctx context.Context, update *client.TxUpdate) {}

func (l *Listener) HandleHeaders(ctx context.Context, headers *client.Headers) {
	count := len(headers.Headers)
	if count == 0 {
		return
	}
	newHeight := headers.StartHeight + uint32(count) - 1
	logger.Info(ctx, "New headers (%d) to height %d : %s", count, newHeight,
		headers.Headers[count-1].BlockHash())

	l.hashesLock.Lock()

	l.height = newHeight

	currentCount := len(l.hashes)
	if currentCount > 0 {
		last := l.hashes[currentCount-1]
		if !headers.Headers[0].PrevBlock.Equal(&last) {
			// not consecutive headers so dump current headers
			logger.Info(ctx, "Not next header : current latest %s, new previous %s", last,
				headers.Headers[0].PrevBlock)
			l.hashes = nil
			currentCount = 0
		}
	}

	if currentCount == 0 || count >= l.offset {
		// Either no current headers or the new set of headers is longer than we need so overwrite
		// with the current set.
		if count > l.offset {
			// Only keep the last "offset" count
			headers.Headers = headers.Headers[count-l.offset-1:]
		}

		l.hashes = make([]bitcoin.Hash32, len(headers.Headers))
		for i, header := range headers.Headers {
			l.hashes[i] = *header.BlockHash()
		}
	} else {
		newLength := currentCount + count
		if newLength > l.offset {
			// Trim oldest current headers
			trimCount := newLength - l.offset
			l.hashes = l.hashes[trimCount:]
		}

		// Append new headers
		for _, header := range headers.Headers {
			l.hashes = append(l.hashes, *header.BlockHash())
		}
	}

	currentCount = len(l.hashes)
	l.hashesLock.Unlock()

	if currentCount < l.offset {
		logger.Info(ctx, "Re-initializing headers")
		if err := l.InitializeHeaders(ctx); err != nil {
			logger.Error(ctx, "Failed to re-initialize headers : %s", err)
		}
	}
}

func (l *Listener) HandleInSync(ctx context.Context) {
	l.hashesLock.Lock()
	defer l.hashesLock.Unlock()

	if len(l.hashes) == 0 {
		logger.Error(ctx, "No headers")
	} else {
		logger.Info(ctx, "Latest header of %d at height %d : %s", len(l.hashes), l.height,
			l.hashes[len(l.hashes)-1])
	}
}

func (l *Listener) HandleMessage(ctx context.Context, payload client.MessagePayload) {
	switch msg := payload.(type) {
	case *client.AcceptRegister:
		logger.Info(ctx, "SpyNode registration accepted")

		if l.spyNode != nil {
			// Subscribe to contracts to get all contract formations automatically.
			if err := l.spyNode.SubscribeContracts(ctx); err != nil {
				logger.Error(ctx, "Failed to subscribe to contracts : %s", err)
			}

			// Subscribe to headers to get new headers automatically.
			if err := l.spyNode.SubscribeHeaders(ctx); err != nil {
				logger.Error(ctx, "Failed to subscribe to headers : %s", err)
			}

			var nextMessageID uint64
			if msg.MessageCount == 0 {
				nextMessageID = 1 // either first startup or server reset
			} else {
				nextID, err := l.GetNextMessageID(ctx)
				if err != nil {
					logger.Error(ctx, "Failed to get next message id : %s", err)
					return
				}
				nextMessageID = *nextID
			}

			if err := l.spyNode.Ready(ctx, nextMessageID); err != nil {
				logger.Error(ctx, "Failed to notify spynode ready : %s", err)
				return
			}

			logger.Info(ctx, "SpyNode client ready at next message %d", nextMessageID)

			if err := l.InitializeHeaders(ctx); err != nil {
				logger.Error(ctx, "Failed to initialize headers : %s", err)
			}
		}
	}
}

func (l *Listener) InitializeHeaders(ctx context.Context) error {
	headers, err := l.spyNode.GetHeaders(ctx, -1, l.offset)
	if err != nil {
		return errors.Wrap(err, "get headers")
	}

	count := len(headers.Headers)
	if count == 0 {
		logger.Info(ctx, "No headers found")
		return nil // no headers yet
	}

	l.hashesLock.Lock()

	l.height = headers.StartHeight + uint32(count) - 1

	l.hashes = make([]bitcoin.Hash32, len(headers.Headers))
	for i, header := range headers.Headers {
		l.hashes[i] = *header.BlockHash()
	}

	l.hashesLock.Unlock()

	logger.Info(ctx, "Pulled initial headers (%d) to height %d : %s", count, l.height,
		headers.Headers[count-1].BlockHash())

	return nil
}

// SaveContractFormation saves a contract formation to storage.
func (l *Listener) SaveContractFormation(ctx context.Context, ra bitcoin.RawAddress,
	formation *actions.ContractFormation, script []byte) error {

	key := strings.Join([]string{contractsStorageKey, hex.EncodeToString(ra.Bytes())}, "/")

	// Check for pre-existing
	b, err := l.dbConn.Fetch(ctx, key)
	if err != nil {
		if errors.Cause(err) != db.ErrNotFound {
			return errors.Wrap(err, "read contract formation")
		}

		// ErrNotFound, this is the first version of this contract formation
		logger.Info(ctx, "Saving contract formation : %s : %x",
			bitcoin.NewAddressFromRawAddress(ra, l.net), script)
		if err := l.dbConn.Put(ctx, key, script); err != nil {
			return errors.Wrap(err, "write contract formation")
		}

		return nil
	}

	// Check timestamp vs current version to ensure we keep the latest.
	action, err := protocol.Deserialize(b, l.isTest)
	if err != nil {
		// Overwrite invalid contract formation
		logger.Warn(ctx, "Overwrite invalid contract formation : %s : %x",
			bitcoin.NewAddressFromRawAddress(ra, l.net), script)
		if err := l.dbConn.Put(ctx, key, script); err != nil {
			return errors.Wrap(err, "write contract formation")
		}

		return nil
	}

	current, ok := action.(*actions.ContractFormation)
	if !ok {
		// Overwrite invalid contract formation
		logger.Warn(ctx, "Overwrite non contract formation : %s : %x",
			bitcoin.NewAddressFromRawAddress(ra, l.net), script)
		if err := l.dbConn.Put(ctx, key, script); err != nil {
			return errors.Wrap(err, "write contract formation")
		}

		return nil
	}

	if current.Timestamp > formation.Timestamp {
		return nil // already have a later version
	}

	logger.Info(ctx, "Updating contract formation : %s : %x",
		bitcoin.NewAddressFromRawAddress(ra, l.net), script)
	if err := l.dbConn.Put(ctx, key, script); err != nil {
		return errors.Wrap(err, "write contract formation")
	}

	return nil
}

func (l *Listener) GetNextMessageID(ctx context.Context) (*uint64, error) {
	b, err := l.dbConn.Fetch(ctx, chainstateStorageKey)
	if err != nil {
		if errors.Cause(err) == db.ErrNotFound {
			result := uint64(1)
			return &result, nil
		}
		return nil, errors.Wrap(err, "fetch")
	}

	r := bytes.NewReader(b)

	var version uint8
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, errors.Wrap(err, "version")
	}

	if version != 0 {
		return nil, errors.New("Wrong version")
	}

	var result uint64
	if err := binary.Read(r, binary.LittleEndian, &result); err != nil {
		return nil, errors.Wrap(err, "next message id")
	}

	return &result, nil
}

func (l *Listener) SaveNextMessageID(ctx context.Context, nextMessageID uint64) error {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, chainstateVersion); err != nil {
		return errors.Wrap(err, "version")
	}

	if err := binary.Write(&buf, binary.LittleEndian, nextMessageID); err != nil {
		return errors.Wrap(err, "next message id")
	}

	if err := l.dbConn.Put(ctx, chainstateStorageKey, buf.Bytes()); err != nil {
		return errors.Wrap(err, "put")
	}

	return nil
}
