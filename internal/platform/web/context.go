package web

import (
	"context"
	"time"

	"github.com/tokenized/pkg/bitcoin"
)

// ctxKey represents the type of value for the context key.
type ctxKey int

// KeyValues is how request values or stored/retrieved.
const KeyValues ctxKey = 1

// Values represent state for each request.
type Values struct {
	Now        time.Time
	StatusCode int
	Error      bool
}

// netKey is where the bitcoin network value is stored.
const netKey ctxKey = 2

// ContextNetwork returns the bitcoin network associated with the context.
func ContextNetwork(ctx context.Context) bitcoin.Network {
	netValue := ctx.Value(netKey)
	if netValue == nil {
		return bitcoin.TestNet
	}

	net, ok := netValue.(bitcoin.Network)
	if !ok {
		return bitcoin.TestNet
	}
	return net
}

// testKey is where the test mode value is stored.
const testKey ctxKey = 3

// ContextTestMode returns true if the test mode associated with the context is active.
func ContextTestMode(ctx context.Context) bool {
	testValue := ctx.Value(testKey)
	if testValue == nil {
		return true
	}

	test, ok := testValue.(bool)
	if !ok {
		return true
	}
	return test
}

func ContextWithValues(ctx context.Context, net bitcoin.Network, isTest bool) context.Context {
	ctx = context.WithValue(ctx, netKey, net)
	return context.WithValue(ctx, testKey, isTest)
}
