package handlers

import (
	"context"
	"net/http"

	"github.com/tokenized/identity-oracle/internal/mid"
	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/pkg/bitcoin"
)

// API returns a handler for a set of routes.
func API(ctx context.Context, config *web.Config, masterDB *db.DB, key bitcoin.Key,
	contractAddress bitcoin.RawAddress, headers oracle.Headers, contracts oracle.Contracts,
	transferExpirationDurationSeconds, identityExpirationDurationSeconds int,
	approver oracle.ApproverInterface) http.Handler {

	app := web.New(config, mid.ErrorHandler, mid.CORS)

	// Register OPTIONS fallback handler for preflight requests.
	app.HandleOptions(mid.CORSHandler)

	hh := Health{
		MasterDB: masterDB,
	}
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
	app.Handle("POST", "/oracle/updateIdentity", oh.UpdateIdentity)

	th := Transfers{
		Config:                            config,
		MasterDB:                          masterDB,
		Key:                               key,
		Headers:                           headers,
		TransferExpirationDurationSeconds: transferExpirationDurationSeconds,
		Approver:                          approver,
	}
	app.Handle("POST", "/transfer/approve", th.TransferSignature)

	vh := Verify{
		Config:                            config,
		MasterDB:                          masterDB,
		Key:                               key,
		Headers:                           headers,
		Contracts:                         contracts,
		IdentityExpirationDurationSeconds: identityExpirationDurationSeconds,
		Approver:                          approver,
	}
	app.Handle("POST", "/identity/verifyPubKey", vh.PubKeySignature)
	app.Handle("POST", "/identity/verifyXPub", vh.XPubSignature)
	app.Handle("POST", "/identity/verifyAdmin", vh.AdminCertificate)

	return app
}
