package bootstrap

import (
	"context"
	"net/http"
	"sync"

	"github.com/tokenized/identity-oracle/cmd/identityoracled/handlers"
	"github.com/tokenized/identity-oracle/internal/mid"
	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/spynode/pkg/client"
	"github.com/tokenized/threads"

	"github.com/pkg/errors"
)

type Oracle struct {
	cfg      *Config
	listener *oracle.Listener
	server   *http.Server
	spyNode  client.Client
	db       *db.DB
}

func Setup(ctx context.Context, logConfig logger.Config, cfg *Config, spyNode client.Client,
	approver oracle.ApproverInterface) (*Oracle, error) {

	// ---------------------------------------------------------------------------------------------
	// Signing Key

	key, err := bitcoin.KeyFromStr(cfg.Oracle.Key)
	if err != nil {
		return nil, errors.Wrap(err, "server key")
	}

	// ---------------------------------------------------------------------------------------------
	// Contract Address

	contractAddress, err := bitcoin.DecodeAddress(cfg.Oracle.ContractAddress)
	if err != nil {
		return nil, errors.Wrap(err, "contract address")
	}

	// ---------------------------------------------------------------------------------------------
	// Start Database / Storage

	logger.Info(ctx, "main : Started : Initialize Database")

	masterDB, err := db.New(
		&db.DBConfig{
			Driver: cfg.Db.Driver,
			URL:    cfg.Db.URL,
		},
		&db.StorageConfig{
			Bucket: cfg.Storage.Bucket,
			Root:   cfg.Storage.Root,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "database")
	}

	// ---------------------------------------------------------------------------------------------
	// Web Config

	webConfig := &web.Config{
		RootURL: cfg.Web.RootURL,
		Net:     bitcoin.NetworkFromString(cfg.Bitcoin.Network),
		IsTest:  cfg.Bitcoin.IsTest,
	}

	// ---------------------------------------------------------------------------------------------
	// Listener - Collects headers for signature data and contract formations for service info.

	listener := oracle.NewListener(spyNode, masterDB, webConfig.Net, cfg.Bitcoin.IsTest)

	spyNode.RegisterHandler(listener)

	// ---------------------------------------------------------------------------------------------
	// Start API Service

	ra := bitcoin.NewRawAddressFromAddress(contractAddress)

	webHandler := handlers.API(ctx, webConfig, masterDB, key, ra, listener, listener,
		cfg.Oracle.TransferExpirationDurationSeconds, cfg.Oracle.IdentityExpirationDurationSeconds,
		approver)

	requestLogger := mid.NewRequestLoggingMiddleware(logConfig)
	webHandler = requestLogger.Handler(webHandler)

	api := &http.Server{
		Addr:           cfg.Web.APIHost,
		Handler:        webHandler,
		ReadTimeout:    cfg.Web.ReadTimeout,
		WriteTimeout:   cfg.Web.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	return &Oracle{
		cfg:      cfg,
		listener: listener,
		server:   api,
		spyNode:  spyNode,
		db:       masterDB,
	}, nil
}

func (o *Oracle) Run(ctx context.Context, interrupt <-chan interface{}) error {
	defer o.db.Close()

	var wait sync.WaitGroup

	listenThread, listenComplete := threads.NewUninterruptableThreadComplete("Listen HTTP",
		func(ctx context.Context) error {
			return o.server.ListenAndServe()
		}, &wait)

	listenThread.Start(ctx)

	// ---------------------------------------------------------------------------------------------
	// Stop API Service

	// Blocking main and waiting for shutdown.
	listenStopped := false
	select {
	case err := <-listenComplete:
		logger.Error(ctx, "main : Server completed : %s", err)
		listenStopped = true

	case <-interrupt:
	}

	if !listenStopped {
		// Create context for Shutdown call.
		ctx, cancel := context.WithTimeout(context.Background(), o.cfg.Web.ShutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and load shed.
		if err := o.server.Shutdown(ctx); err != nil {
			logger.Info(ctx, "main : Graceful HTTP server shutdown did not complete in %v : %v",
				o.cfg.Web.ShutdownTimeout, err)
			if err := o.server.Close(); err != nil {
				return errors.Wrap(err, "close server")
			}
		}
	}

	wait.Wait()

	if err := o.Save(ctx); err != nil {
		return errors.Wrap(err, "save")
	}

	return nil
}

func (o *Oracle) Save(ctx context.Context) error {
	if err := o.listener.SaveNextMessageID(ctx, o.spyNode.NextMessageID()); err != nil {
		return errors.Wrap(err, "save next message id")
	}

	return nil
}
