package main

import (
	"net/http"

	"github.com/tokenized/identity-oracle/cmd/identityoracled/handlers"
	"github.com/tokenized/identity-oracle/internal/mid"
	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
)

// API returns a handler for a set of routes.
func API(log logger.Logger, config *web.Config, masterDB *db.DB, key bitcoin.Key,
	blockHandler *oracle.BlockHandler) http.Handler {

	app := web.New(config, log, mid.Metrics, mid.ErrorHandler, mid.CORS)

	// Register OPTIONS fallback handler for preflight requests.
	app.HandleOptions(mid.CORSHandler)

	hh := handlers.Health{}
	app.Handle("GET", "/health", hh.Health)

	// We don't need to log health requests, so add this middleware after the health request.
	app.AddMiddleWare(mid.RequestLogger)

	oh := handlers.Oracle{
		Config:   config,
		MasterDB: masterDB,
		Key:      key,
	}
	app.Handle("GET", "/oracle/id", oh.Identity)
	app.Handle("POST", "/oracle/register", oh.Register)
	app.Handle("POST", "/oracle/addXPub", oh.AddXPub)
	app.Handle("POST", "/oracle/user", oh.User)

	th := handlers.Transfers{
		Config:       config,
		MasterDB:     masterDB,
		Key:          key,
		BlockHandler: blockHandler,
	}
	app.Handle("POST", "/transfer/approve", th.TransferSignature)

	vh := handlers.Verify{
		Config:       config,
		MasterDB:     masterDB,
		Key:          key,
		BlockHandler: blockHandler,
	}
	app.Handle("POST", "/identity/verifyPubKey", vh.PubKeySignature)
	app.Handle("POST", "/identity/verifyXPub", vh.XPubSignature)

	return app
}
