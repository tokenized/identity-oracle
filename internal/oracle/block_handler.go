package oracle

import (
	"bytes"
	"context"
	"encoding/binary"
	"sync"

	"github.com/tokenized/identity-oracle/internal/platform/db"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/spynode/handlers"
	"github.com/tokenized/pkg/wire"

	"github.com/pkg/errors"
)

type BlockHandler struct {
	Log          logger.Logger
	InSync       bool
	LatestHeight uint32
	LatestBlocks []bitcoin.Hash32
	Lock         sync.Mutex
}

// SigHash returns the block hash to be signed against at height tip - 4
func (bh *BlockHandler) SigHash(ctx context.Context) (bitcoin.Hash32, uint32, error) {
	bh.Lock.Lock()
	defer bh.Lock.Unlock()

	if len(bh.LatestBlocks) < 4 {
		return bitcoin.Hash32{}, 0, errors.New("Not enough blocks")
	}

	return bh.LatestBlocks[len(bh.LatestBlocks)-4], bh.LatestHeight - 3, nil
}

func (bh *BlockHandler) Save(ctx context.Context, dbConn *db.DB) error {
	bh.Lock.Lock()
	defer bh.Lock.Unlock()

	var buf bytes.Buffer

	if err := binary.Write(&buf, binary.LittleEndian, bh.LatestHeight); err != nil {
		return err
	}

	if err := binary.Write(&buf, binary.LittleEndian, uint8(len(bh.LatestBlocks))); err != nil {
		return err
	}

	for _, hash := range bh.LatestBlocks {
		buf.Write(hash[:])
	}

	if err := dbConn.Put(ctx, "blocks", buf.Bytes()); err != nil {
		return errors.Wrap(err, "put blocks in storage")
	}

	return nil
}

func (bh *BlockHandler) Load(ctx context.Context, dbConn *db.DB) error {
	bh.Lock.Lock()
	defer bh.Lock.Unlock()

	b, err := dbConn.Fetch(ctx, "blocks")
	if err != nil {
		if err == db.ErrNotFound {
			return nil
		}

		return errors.Wrap(err, "fetch blocks from storage")
	}

	buf := bytes.NewBuffer(b)

	if err := binary.Read(buf, binary.LittleEndian, &bh.LatestHeight); err != nil {
		return err
	}

	var count uint8
	if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
		return err
	}

	bh.LatestBlocks = make([]bitcoin.Hash32, int(count))
	for i := 0; i < int(count); i++ {
		buf.Read(bh.LatestBlocks[i][:])
	}

	return nil
}

/************************ Implement the SpyNode Listener interface. *******************************/

func (bh *BlockHandler) HandleBlock(ctx context.Context, msgType int, block *handlers.BlockMessage) error {
	bh.Lock.Lock()
	defer bh.Lock.Unlock()

	switch msgType {
	case handlers.ListenerMsgBlock:
		bh.Log.Printf("New Block (%d) : %s\n", block.Height, block.Hash.String())
		bh.LatestHeight = uint32(block.Height)
		bh.LatestBlocks = append(bh.LatestBlocks, block.Hash)
		if len(bh.LatestBlocks) > 10 {
			bh.LatestBlocks = bh.LatestBlocks[len(bh.LatestBlocks)-10:]
		}
	case handlers.ListenerMsgBlockRevert:
		bh.Log.Printf("Reverted Block (%d) : %s\n", block.Height, block.Hash.String())
		bh.LatestHeight = uint32(block.Height)
		if len(bh.LatestBlocks) > 0 {
			if bh.LatestBlocks[len(bh.LatestBlocks)-1].Equal(&block.Hash) {
				bh.LatestBlocks = bh.LatestBlocks[:len(bh.LatestBlocks)-1]
			}
		}
	}
	return nil
}

func (bh *BlockHandler) HandleTx(ctx context.Context, tx *wire.MsgTx) (bool, error) {
	bh.Log.Printf("Tx : %s\n", tx.TxHash().String())
	return true, nil
}

func (bh *BlockHandler) HandleTxState(ctx context.Context, msgType int, txid bitcoin.Hash32) error {
	switch msgType {
	case handlers.ListenerMsgTxStateSafe:
		bh.Log.Printf("Tx safe : %s\n", txid.String())

	case handlers.ListenerMsgTxStateConfirm:
		bh.Log.Printf("Tx confirm : %s\n", txid.String())

	case handlers.ListenerMsgTxStateCancel:
		bh.Log.Printf("Tx cancel : %s\n", txid.String())

	case handlers.ListenerMsgTxStateUnsafe:
		bh.Log.Printf("Tx unsafe : %s\n", txid.String())

	case handlers.ListenerMsgTxStateRevert:
		bh.Log.Printf("Tx revert : %s\n", txid.String())

	}
	return nil
}

func (bh *BlockHandler) HandleInSync(ctx context.Context) error {
	bh.Lock.Lock()
	defer bh.Lock.Unlock()

	bh.Log.Printf("Node is in sync\n")
	bh.InSync = true

	bh.Log.Printf("Latest blocks :\n")
	height := bh.LatestHeight
	for i := len(bh.LatestBlocks) - 1; i >= 0; i-- {
		bh.Log.Printf("  %d %s\n", height, bh.LatestBlocks[i].String())
		height--
	}

	return nil
}
