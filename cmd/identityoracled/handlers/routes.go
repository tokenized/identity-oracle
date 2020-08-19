package handlers

import (
	"net/http"

	"github.com/tokenized/identity-oracle/internal/mid"
	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
)

// API returns a handler for a set of routes.
func API(log logger.Logger, config *web.Config, masterDB *db.DB, key bitcoin.Key,
	contractAddress bitcoin.RawAddress, blockHandler *oracle.BlockHandler,
	transferExpirationDurationSeconds, identityExpirationDurationSeconds int,
	approver oracle.ApproverInterface) http.Handler {

	app := web.New(config, log, mid.RequestLogger, mid.Metrics, mid.ErrorHandler, mid.CORS)

	// Register OPTIONS fallback handler for preflight requests.
	app.HandleOptions(mid.CORSHandler)

	hh := Health{}
	app.Handle("GET", "/health", hh.Health)

	oh := Oracle{
		Config:          config,
		MasterDB:        masterDB,
		Approver:        approver,
		Key:             key,
		ContractAddress: contractAddress,
	}
	app.Handle("GET", "/oracle/id", oh.Identity)
	app.Handle("POST", "/oracle/register", oh.Register)
	app.Handle("POST", "/oracle/addXPub", oh.AddXPub)
	app.Handle("POST", "/oracle/user", oh.User)

	th := Transfers{
		Config:                            config,
		MasterDB:                          masterDB,
		Key:                               key,
		BlockHandler:                      blockHandler,
		TransferExpirationDurationSeconds: transferExpirationDurationSeconds,
		Approver:                          approver,
	}
	app.Handle("POST", "/transfer/approve", th.TransferSignature)

	vh := Verify{
		Config:                            config,
		MasterDB:                          masterDB,
		Key:                               key,
		BlockHandler:                      blockHandler,
		IdentityExpirationDurationSeconds: identityExpirationDurationSeconds,
		Approver:                          approver,
	}
	app.Handle("POST", "/identity/verifyPubKey", vh.PubKeySignature)
	app.Handle("POST", "/identity/verifyXPub", vh.XPubSignature)
	app.Handle("POST", "/identity/verifyAdmin", vh.AdminCertificate)

	return app
}
