package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/tokenized/config"
	"github.com/tokenized/identity-oracle/cmd/identityoracled/bootstrap"
	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/rpcnode"
	"github.com/tokenized/pkg/storage"
	spynodeBootstrap "github.com/tokenized/spynode/cmd/spynoded/bootstrap"
	"github.com/tokenized/threads"
)

func main() {
	// ---------------------------------------------------------------------------------------------
	// Logging

	logPath := os.Getenv("LOG_FILE_PATH")

	logConfig := logger.NewConfig(strings.ToUpper(os.Getenv("DEVELOPMENT")) == "TRUE",
		strings.ToUpper(os.Getenv("LOG_FORMAT")) == "TEXT", logPath)

	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(spynodeBootstrap.SubSystem)

	ctx := logger.ContextWithLogConfig(context.Background(), logConfig)

	// ---------------------------------------------------------------------------------------------
	// App Starting

	logger.Info(ctx, "main : Started : Application Initializing")
	defer logger.Info(ctx, "main : Completed")

	// -------------------------------------------------------------------------
	// Config

	cfg := &Config{}
	// load config using sane fallbacks
	if err := config.LoadConfig(ctx, cfg); err != nil {
		logger.Fatal(ctx, "main : Load Config : %v", err)
	}

	config.DumpSafe(ctx, cfg)

	// -------------------------------------------------------------------------
	// RPC Node
	rpcConfig := &rpcnode.Config{
		Host:                   cfg.RpcNode.Host,
		Username:               cfg.RpcNode.Username,
		Password:               cfg.RpcNode.Password,
		MaxRetries:             cfg.RpcNode.MaxRetries,
		RetryDelay:             cfg.RpcNode.RetryDelay,
		IgnoreAlreadyInMempool: true, // multiple clients might send the same tx
	}

	rpcNode, err := rpcnode.NewNode(rpcConfig)
	if err != nil {
		logger.Fatal(ctx, "Create RPC : %s", err)
	}

	// -------------------------------------------------------------------------
	// SPY Node
	spyStorageConfig := storage.NewConfig(cfg.NodeStorage.Bucket, cfg.NodeStorage.Root)
	spyStorageConfig.SetupRetry(cfg.AWS.MaxRetries, cfg.AWS.RetryDelay)

	var spyStorage storage.Storage
	if strings.ToLower(spyStorageConfig.Bucket) == "standalone" {
		spyStorage = storage.NewFilesystemStorage(spyStorageConfig)
	} else {
		spyStorage = storage.NewS3Storage(spyStorageConfig)
	}

	net := bitcoin.NetworkFromString(cfg.Bitcoin.Network)

	spyConfig, err := spynodeBootstrap.NewConfig(net, cfg.Bitcoin.IsTest,
		cfg.SpyNode.Address, cfg.SpyNode.UserAgent, cfg.SpyNode.StartHash,
		cfg.SpyNode.UntrustedNodes, cfg.SpyNode.SafeTxDelay, cfg.SpyNode.ShotgunCount,
		cfg.SpyNode.MaxRetries, cfg.SpyNode.RetryDelay, cfg.SpyNode.RequestMempool)
	if err != nil {
		logger.Fatal(ctx, "Failed to create spynode config : %s", err)
	}

	spyNode := spynodeBootstrap.NewNode(spyConfig, spyStorage, rpcNode, rpcNode)

	server, err := bootstrap.Setup(ctx, logConfig, &cfg.Oracle, spyNode, nil)
	if err != nil {
		logger.Fatal(ctx, "Failed to setup server : %s", err)
	}

	// Start the service listening for requests.
	var wait sync.WaitGroup
	oracleThread, oracleComplete := threads.NewInterruptableThreadComplete("Oracle", server.Run,
		&wait)

	spyNodeThread, spyNodeComplete := threads.NewUninterruptableThreadComplete("SpyNode",
		spyNode.Run, &wait)

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	oracleThread.Start(ctx)
	spyNodeThread.Start(ctx)

	select {
	case err := <-oracleComplete:
		logger.Error(ctx, "Oracle completed : %s", err)

	case err := <-spyNodeComplete:
		logger.Error(ctx, "SpyNode completed : %s", err)

	case <-osSignals:
		logger.Info(ctx, "Shutdown requested")
	}

	oracleThread.Stop(ctx)
	spyNode.Stop(ctx)
	wait.Wait()
}

type Config struct {
	Bitcoin struct {
		Network string `default:"mainnet" envconfig:"BITCOIN_CHAIN" json:"BITCOIN_CHAIN"`
		IsTest  bool   `default:"true" envconfig:"IS_TEST" json:"IS_TEST"`
	}
	SpyNode struct {
		Address        string `default:"127.0.0.1:8333" envconfig:"NODE_ADDRESS" json:"NODE_ADDRESS"`
		UserAgent      string `default:"/Tokenized:0.1.0/" envconfig:"NODE_USER_AGENT" json:"NODE_USER_AGENT"`
		StartHash      string `envconfig:"START_HASH" json:"START_HASH"`
		UntrustedNodes int    `default:"25" envconfig:"UNTRUSTED_NODES" json:"UNTRUSTED_NODES"`
		SafeTxDelay    int    `default:"2000" envconfig:"SAFE_TX_DELAY" json:"SAFE_TX_DELAY"`
		ShotgunCount   int    `default:"100" envconfig:"SHOTGUN_COUNT" json:"SHOTGUN_COUNT"`
		MaxRetries     int    `default:"25" envconfig:"NODE_MAX_RETRIES" json:"NODE_MAX_RETRIES"`
		RetryDelay     int    `default:"5000" envconfig:"NODE_RETRY_DELAY" json:"NODE_RETRY_DELAY"`
		RequestMempool bool   `default:"true" envconfig:"REQUEST_MEMPOOL" json:"REQUEST_MEMPOOL"`
	}
	RpcNode struct {
		Host       string `envconfig:"RPC_HOST" json:"RPC_HOST"`
		Username   string `envconfig:"RPC_USERNAME" json:"RPC_USERNAME"`
		Password   string `envconfig:"RPC_PASSWORD" json:"RPC_PASSWORD" masked:"true"`
		MaxRetries int    `default:"10" envconfig:"RPC_MAX_RETRIES" json:"RPC_MAX_RETRIES"`
		RetryDelay int    `default:"2000" envconfig:"RPC_RETRY_DELAY" json:"RPC_RETRY_DELAY"`
	}
	NodeStorage struct {
		Bucket string `default:"standalone" envconfig:"NODE_STORAGE_BUCKET" json:"NODE_STORAGE_BUCKET"`
		Root   string `default:"./tmp" envconfig:"NODE_STORAGE_ROOT" json:"NODE_STORAGE_ROOT"`
	}
	AWS struct {
		Region          string `default:"ap-southeast-2" envconfig:"AWS_REGION" json:"AWS_REGION"`
		AccessKeyID     string `envconfig:"ACCESS_KEY_ID" json:"ACCESS_KEY_ID"`
		SecretAccessKey string `envconfig:"SECRET_ACCESS_KEY" json:"SECRET_ACCESS_KEY" masked:"true"`
		MaxRetries      int    `default:"10" envconfig:"AWS_MAX_RETRIES"`
		RetryDelay      int    `default:"2000" envconfig:"AWS_RETRY_DELAY"`
	}
	Oracle bootstrap.Config `envconfig:"ORACLE" json:"ORACLE"`
}
