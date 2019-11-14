package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"log"

	"github.com/pkg/errors"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type BlockHandler struct {
	Log          *log.Logger
	InSync       bool
	LatestBlocks []bitcoin.Hash32
}

func (bh *BlockHandler) Save(ctx context.Context, dbConn *db.DB) error {
	var buf bytes.Buffer

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
	b, err := dbConn.Fetch(ctx, "blocks")
	if err != nil {
		if err == db.ErrNotFound {
			return nil
		}

		return errors.Wrap(err, "fetch blocks from storage")
	}

	buf := bytes.NewBuffer(b)

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
	switch msgType {
	case handlers.ListenerMsgBlock:
		bh.Log.Printf("New Block (%d) : %s\n", block.Height, block.Hash.String())
		bh.LatestBlocks = append(bh.LatestBlocks, block.Hash)
		if len(bh.LatestBlocks) > 10 {
			bh.LatestBlocks = bh.LatestBlocks[len(bh.LatestBlocks)-10:]
		}
	case handlers.ListenerMsgBlockRevert:
		bh.Log.Printf("Reverted Block (%d) : %s\n", block.Height, block.Hash.String())
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
	bh.Log.Printf("Node is in sync\n")
	bh.InSync = true

	bh.Log.Printf("Latest blocks :\n")
	for i := len(bh.LatestBlocks) - 1; i >= 0; i-- {
		bh.Log.Printf("  %s\n", bh.LatestBlocks[i].String())
	}

	return nil
}
