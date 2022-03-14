package oracle

import (
	"math/rand"
	"testing"
	"time"

	"github.com/tokenized/identity-oracle/internal/platform/tests"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/spynode/pkg/client"
)

func TestNewHeader(t *testing.T) {
	ctx := tests.Context()

	listener := &Listener{
		offset: 4,
	}

	// Generate some headers
	headers := make([]*wire.BlockHeader, 10)
	var prevHash bitcoin.Hash32
	rand.Read(prevHash[:])
	for i := range headers {
		var merkleRoot bitcoin.Hash32
		rand.Read(merkleRoot[:])
		headers[i] = &wire.BlockHeader{
			Version:    1,
			PrevBlock:  prevHash,
			MerkleRoot: merkleRoot,
			Timestamp:  uint32(time.Now().Unix()),
			Bits:       rand.Uint32(),
			Nonce:      rand.Uint32(),
		}

		prevHash = *headers[i].BlockHash()
	}

	// Start with full hashes and add one
	listener.height = 5
	listener.hashes = []bitcoin.Hash32{
		*headers[0].BlockHash(),
		*headers[1].BlockHash(),
		*headers[2].BlockHash(),
		*headers[3].BlockHash(),
	}

	clientHeaders := &client.Headers{
		RequestHeight: -1,
		StartHeight:   6,
		Headers: []*wire.BlockHeader{
			headers[4],
		},
	}

	listener.HandleHeaders(ctx, clientHeaders)

	// Check that one hash was added
	if len(listener.hashes) != listener.offset {
		t.Fatalf("Wrong number of hashes : got %d, want %d", len(listener.hashes), listener.offset)
	}

	if listener.height != 6 {
		t.Fatalf("Wrong hash height : got %d, want %d", listener.height, 6)
	}

	if !listener.hashes[0].Equal(headers[1].BlockHash()) {
		t.Fatalf("Wrong oldest hash : got %s, want %s", listener.hashes[0],
			headers[1].BlockHash())
	}

	if !listener.hashes[listener.offset-1].Equal(headers[4].BlockHash()) {
		t.Fatalf("Wrong latest hash : got %s, want %s", listener.hashes[listener.offset],
			headers[4].BlockHash())
	}

	// Append another
	clientHeaders = &client.Headers{
		RequestHeight: -1,
		StartHeight:   7,
		Headers: []*wire.BlockHeader{
			headers[5],
		},
	}

	listener.HandleHeaders(ctx, clientHeaders)

	// Check that one hash was added
	if len(listener.hashes) != listener.offset {
		t.Fatalf("Wrong number of hashes : got %d, want %d", len(listener.hashes), listener.offset)
	}

	if listener.height != 7 {
		t.Fatalf("Wrong hash height : got %d, want %d", listener.height, 6)
	}

	if !listener.hashes[0].Equal(headers[2].BlockHash()) {
		t.Fatalf("Wrong oldest hash : got %s, want %s", listener.hashes[0],
			headers[2].BlockHash())
	}

	if !listener.hashes[listener.offset-1].Equal(headers[5].BlockHash()) {
		t.Fatalf("Wrong latest hash : got %s, want %s", listener.hashes[listener.offset],
			headers[5].BlockHash())
	}

	// Reorg 3
	clientHeaders = &client.Headers{
		RequestHeight: -1,
		StartHeight:   4,
		Headers: []*wire.BlockHeader{
			headers[3],
			headers[4],
			headers[5],
			headers[6],
		},
	}

	listener.HandleHeaders(ctx, clientHeaders)

	// Check that one hash was added
	if len(listener.hashes) != listener.offset {
		t.Fatalf("Wrong number of hashes : got %d, want %d", len(listener.hashes), listener.offset)
	}

	if listener.height != 8 {
		t.Fatalf("Wrong hash height : got %d, want %d", listener.height, 8)
	}

	if !listener.hashes[0].Equal(headers[3].BlockHash()) {
		t.Fatalf("Wrong oldest hash : got %s, want %s", listener.hashes[0],
			headers[3].BlockHash())
	}

	if !listener.hashes[listener.offset-1].Equal(headers[6].BlockHash()) {
		t.Fatalf("Wrong latest hash : got %s, want %s", listener.hashes[listener.offset],
			headers[6].BlockHash())
	}
}
