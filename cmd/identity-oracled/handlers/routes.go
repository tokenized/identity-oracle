package handlers

import (
	"log"
	"net/http"

	"github.com/tokenized/identity-oracle/internal/mid"
	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

// API returns a handler for a set of routes.
func API(log *log.Logger, config *web.Config, masterDB *db.DB, key bitcoin.Key,
	blockHandler *oracle.BlockHandler) http.Handler {

	app := web.New(config, log, mid.RequestLogger, mid.Metrics, mid.ErrorHandler, mid.CORS)

	// Register OPTIONS fallback handler for preflight requests.
	app.HandleOptions(mid.CORSHandler)

	oh := Oracle{
		Config:   config,
		MasterDB: masterDB,
		Key:      key,
		// Entity: // TODO Set entity for oracle
	}
	app.Handle("GET", "/oracle/id", oh.Identity)
	app.Handle("GET", "/oracle/register", oh.Register)

	th := Transfers{
		Config:       config,
		MasterDB:     masterDB,
		Key:          key,
		BlockHandler: blockHandler,
	}
	app.Handle("POST", "/transfer/approve", th.TransferSignature)

	return app
}
