package bootstrap

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/tokenized/identity-oracle/cmd/identityoracled/handlers"
	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/spynode/pkg/client"

	"github.com/pkg/errors"
)

type Oracle struct {
	cfg      *Config
	listener *oracle.Listener
	server   *http.Server
	spyNode  client.Client
	db       *db.DB
}

func Setup(ctx context.Context, cfg *Config, spyNode client.Client,
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

func (o *Oracle) Run(ctx context.Context, spyNodeErrors *chan error) error {
	defer o.db.Close()

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		logger.Info(ctx, "main : HTTP server Listening %s", o.cfg.Web.APIHost)
		result := o.server.ListenAndServe()
		if result != nil { // If there is no error, then it was requested closed by an interrupt
			logger.Info(ctx, "main : HTTP server finished : %s", result)
		} else {
			logger.Info(ctx, "main : HTTP server finished")
		}
		serverErrors <- result
	}()

	// ---------------------------------------------------------------------------------------------
	// Shutdown

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	// ---------------------------------------------------------------------------------------------
	// Stop API Service

	// Blocking main and waiting for shutdown.
	select {
	case err := <-*spyNodeErrors:
		if err != nil {
			logger.Error(ctx, "main : Spynode failed : %s", err)
		}

		// Asking listener to shutdown and load shed.
		if err := o.server.Shutdown(ctx); err != nil {
			logger.Info(ctx, "main : Graceful HTTP server shutdown did not complete in %v : %v",
				o.cfg.Web.ShutdownTimeout, err)
			if err := o.server.Close(); err != nil {
				logger.Error(ctx, "main : Could not stop HTTP server: %v", err)
			}
		}

	case err := <-serverErrors:
		if err != nil {
			logger.Error(ctx, "main : Server failed : %s", err)
		}

	case <-osSignals:
		logger.Info(ctx, "main : Start shutdown...")

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

	return nil
}

func (o *Oracle) Save(ctx context.Context) error {
	if err := o.listener.SaveNextMessageID(ctx, o.spyNode.NextMessageID()); err != nil {
		return errors.Wrap(err, "save next message id")
	}

	return nil
}
