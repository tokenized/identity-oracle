package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tokenized/identity-oracle/cmd/identityoracled/handlers"
	"github.com/tokenized/identity-oracle/internal/oracle"
	"github.com/tokenized/identity-oracle/internal/platform/config"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"

	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"
)

func main() {

	// ---------------------------------------------------------------------------------------------
	// Logging

	var logConfig *logger.Config
	if strings.ToUpper(os.Getenv("DEVELOPMENT")) == "TRUE" {
		logConfig = logger.NewDevelopmentConfig()
	} else {
		logConfig = logger.NewProductionConfig()
	}

	logFileName := os.Getenv("LOG_FILE_PATH")
	if len(logFileName) > 0 {
		if err := logConfig.Main.AddFile(logFileName); err != nil {
			fmt.Printf("Failed to Address log file : %s\n", err)
			return
		}
	}
	logConfig.Main.Format |= logger.IncludeSystem

	if strings.ToUpper(os.Getenv("LOG_FORMAT")) == "TEXT" {
		logConfig.IsText = true
	}

	logConfig.EnableSubSystem(spynode.SubSystem)
	if strings.ToUpper(os.Getenv("LOG_FORMAT")) == "TEXT" {
		logConfig.IsText = true
	}
	ctx := logger.ContextWithLogConfig(context.Background(), logConfig)

	log := logger.NewLoggerObject(ctx)

	// ---------------------------------------------------------------------------------------------
	// Config

	cfg, err := config.Environment()
	if err != nil {
		log.Fatalf("main : Parsing Config : %v", err)
	}

	// ---------------------------------------------------------------------------------------------
	// App Starting

	log.Println("main : Started : Application Initializing")
	defer log.Println("main : Completed")

	// Mask sensitive values
	cfgSafe := config.SafeConfig(*cfg)
	cfgJSON, err := json.MarshalIndent(cfgSafe, "", "    ")
	if err != nil {
		log.Fatalf("main : Marshalling Config to JSON : %v", err)
	}
	log.Printf("main : Config : %v\n", string(cfgJSON))

	// ---------------------------------------------------------------------------------------------
	// Signing Key
	key, err := bitcoin.KeyFromStr(cfg.Oracle.Key)
	if err != nil {
		log.Fatalf("main : Reading key : %v", err)
	}

	// ---------------------------------------------------------------------------------------------
	// Entity

	entityFileName := os.Getenv("ENTITY_FILE")
	var entity actions.EntityField
	if len(entityFileName) > 0 {
		data, err := ioutil.ReadFile(filepath.FromSlash(entityFileName))
		if err != nil {
			log.Printf("Failed to read entity json file : %s\n", err)
			return
		}

		// Put json data into opReturn struct
		if err := json.Unmarshal(data, &entity); err != nil {
			log.Printf("Failed to unmarshal entity json file : %s\n", err)
			return
		}
	}

	// ---------------------------------------------------------------------------------------------
	// SPY Node
	spyStorageConfig := storage.NewConfig(cfg.NodeStorage.Bucket, cfg.NodeStorage.Root)

	var spyStorage storage.Storage
	if strings.ToLower(spyStorageConfig.Bucket) == "standalone" {
		spyStorage = storage.NewFilesystemStorage(spyStorageConfig)
	} else {
		spyStorage = storage.NewS3Storage(spyStorageConfig)
	}

	spyConfig, err := data.NewConfig(bitcoin.NetworkFromString(cfg.Bitcoin.Network),
		cfg.SpyNode.Address, cfg.SpyNode.UserAgent, cfg.SpyNode.StartHash,
		cfg.SpyNode.UntrustedNodes, cfg.SpyNode.SafeTxDelay, cfg.SpyNode.ShotgunCount)
	if err != nil {
		log.Printf("main : Failed to create spynode config : %s", err)
		return
	}

	spyNode := spynode.NewNode(spyConfig, spyStorage)
	spyNode.AddTxFilter(&TxFilter{})

	// ---------------------------------------------------------------------------------------------
	// Start Database / Storage

	log.Println("main : Started : Initialize Database")

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
		log.Fatalf("main : Register DB : %v", err)
	}
	defer masterDB.Close()

	blockHandler := &oracle.BlockHandler{Log: log}
	if err := blockHandler.Load(ctx, masterDB); err != nil {
		log.Fatalf("main : Load blocks : %v", err)
	}
	defer blockHandler.Save(ctx, masterDB)

	spyNode.RegisterListener(blockHandler)

	// ---------------------------------------------------------------------------------------------
	// Web Config

	webConfig := &web.Config{
		RootURL: cfg.Web.RootURL,
		Net:     bitcoin.NetworkFromString(cfg.Bitcoin.Network),
		IsTest:  cfg.Bitcoin.IsTest,
		Entity:  entity,
	}

	// ---------------------------------------------------------------------------------------------
	// Start API Service

	webHandler := handlers.API(log, webConfig, masterDB, key, blockHandler)

	api := http.Server{
		Addr:           cfg.Web.APIHost,
		Handler:        webHandler,
		ReadTimeout:    cfg.Web.ReadTimeout,
		WriteTimeout:   cfg.Web.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	go func() {
		result := spyNode.Run(ctx)
		if result != nil { // If there is no error, then it was requested closed by an interrupt
			log.Printf("main : Error starting spynode: %s", err)
			if err := api.Shutdown(ctx); err != nil {
				log.Printf("main : Graceful HTTP server shutdown did not complete in %v : %v",
					cfg.Web.ShutdownTimeout, err)
				if err := api.Close(); err != nil {
					log.Fatalf("main : Could not stop HTTP server: %v", err)
				}
			}
			serverErrors <- result
		} else {
			log.Printf("main : Spynode finished")
		}
	}()

	// Start the service listening for requests.
	go func() {
		log.Printf("main : HTTP server Listening %s", cfg.Web.APIHost)
		result := api.ListenAndServe()
		if result != nil { // If there is no error, then it was requested closed by an interrupt
			log.Printf("main : Error starting HTTP server: %s", result)
			if err := spyNode.Stop(ctx); err != nil {
				log.Printf("main : Graceful spynode shutdown did not complete in %v : %v",
					cfg.Web.ShutdownTimeout, err)
				if err := api.Close(); err != nil {
					log.Fatalf("main : Could not stop spynode: %v", err)
				}
			}
			serverErrors <- result
		} else {
			log.Printf("main : HTTP server finished")
		}
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
	case _ = <-serverErrors:
		log.Fatalf("main : Server stopped")

	case <-osSignals:
		log.Println("main : Start shutdown...")

		// Create context for Shutdown call.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := spyNode.Stop(ctx); err != nil {
			log.Printf("main : Graceful spynode shutdown did not complete in %v : %v",
				cfg.Web.ShutdownTimeout, err)
			if err := api.Close(); err != nil {
				log.Fatalf("main : Could not stop spynode: %v", err)
			}
		}

		// Asking listener to shutdown and load shed.
		if err := api.Shutdown(ctx); err != nil {
			log.Printf("main : Graceful HTTP server shutdown did not complete in %v : %v",
				cfg.Web.ShutdownTimeout, err)
			if err := api.Close(); err != nil {
				log.Fatalf("main : Could not stop HTTP server: %v", err)
			}
		}
	}
}

type TxFilter struct{}

func (filter *TxFilter) IsRelevant(ctx context.Context, tx *wire.MsgTx) bool {
	return false // We only care about block hashes
}
