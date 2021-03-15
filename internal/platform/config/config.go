package config

import (
	"encoding/json"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config is used to hold all runtime configuration.
//
// The same Config is used regardless of if the server is a regualar HTTP
// server, or a Lambda function.
type Config struct {
	Env    string `envconfig:"ENV" json:"ENV"`
	Oracle struct {
		Key                               string `envconfig:"KEY" json:"KEY"`
		ContractAddress                   string `envconfig:"CONTRACT_ADDRESS" json:"CONTRACT_ADDRESS"`
		TransferExpirationDurationSeconds int    `default:"21600" envconfig:"TRANSFER_EXPIRATION_DURATION_SECONDS" json:"TRANSFER_EXPIRATION_DURATION_SECONDS"`
		IdentityExpirationDurationSeconds int    `default:"21600" envconfig:"IDENTITY_EXPIRATION_DURATION_SECONDS" json:"IDENTITY_EXPIRATION_DURATION_SECONDS"`
	}
	Web struct {
		RootURL         string        `envconfig:"ROOT_URL" json:"ROOT_URL"`
		APIHost         string        `default:"0.0.0.0:8080" envconfig:"API_HOST" json:"API_HOST"`
		ReadTimeout     time.Duration `default:"5s" envconfig:"READ_TIMEOUT" json:"READ_TIMEOUT"`
		WriteTimeout    time.Duration `default:"5s" envconfig:"WRITE_TIMEOUT" json:"WRITE_TIMEOUT"`
		ShutdownTimeout time.Duration `default:"5s" envconfig:"SHUTDOWN_TIMEOUT" json:"SHUTDOWN_TIMEOUT"`
	}
	Bitcoin struct {
		Network string `default:"mainnet" envconfig:"BITCOIN_CHAIN" json:"BITCOIN_CHAIN"`
		IsTest  bool   `default:"true" envconfig:"IS_TEST" json:"IS_TEST"`
	}
	Db struct {
		Driver string `default:"postgres" envconfig:"DB_DRIVER" json:"DB_DRIVER"`
		URL    string `default:"user=foo dbname=bar sslmode=disable" envconfig:"DB_URL" json:"DB_URL"`
	}
	Storage struct {
		Region    string `default:"ap-southeast-2" envconfig:"STORAGE_REGION" json:"STORAGE_REGION"`
		AccessKey string `envconfig:"STORAGE_ACCESS_KEY" json:"STORAGE_ACCESS_KEY"`
		Secret    string `envconfig:"STORAGE_SECRET" json:"STORAGE_SECRET"`
		Bucket    string `default:"standalone" envconfig:"STORAGE_BUCKET" json:"STORAGE_BUCKET"`
		Root      string `default:"./tmp" envconfig:"STORAGE_ROOT" json:"STORAGE_ROOT"`
	}
	SpyNode struct {
		Address        string `default:"127.0.0.1:8333" envconfig:"NODE_ADDRESS"`
		UserAgent      string `default:"/TokenizedOracle:0.1.0/" envconfig:"NODE_USER_AGENT"`
		StartHash      string `envconfig:"START_HASH"`
		UntrustedNodes int    `default:"8" envconfig:"UNTRUSTED_NODES"`
		SafeTxDelay    int    `default:"2000" envconfig:"SAFE_TX_DELAY"`
		ShotgunCount   int    `default:"100" envconfig:"SHOTGUN_COUNT"`
	}
	NodeStorage struct {
		Region    string `default:"ap-southeast-2" envconfig:"NODE_STORAGE_REGION"`
		AccessKey string `envconfig:"NODE_STORAGE_ACCESS_KEY"`
		Secret    string `envconfig:"NODE_STORAGE_SECRET"`
		Bucket    string `default:"standalone" envconfig:"NODE_STORAGE_BUCKET"`
		Root      string `default:"./tmp" envconfig:"NODE_STORAGE_ROOT"`
	}
	RpcNode struct {
		Host       string `envconfig:"RPC_HOST" json:"RPC_HOST"`
		Username   string `envconfig:"RPC_USERNAME" json:"RPC_USERNAME"`
		Password   string `envconfig:"RPC_PASSWORD" json:"RPC_PASSWORD"`
		MaxRetries int    `default:"10" envconfig:"RPC_MAX_RETRIES"`
		RetryDelay int    `default:"2000" envconfig:"RPC_RETRY_DELAY"`
	}
}

// unmarshalNested applies JSON configuration
func unmarshalNested(data []byte, cfg *Config) error {
	var err error

	// Unmarshal each item
	err = json.Unmarshal(data, &cfg)
	err = json.Unmarshal(data, &cfg.Oracle)
	err = json.Unmarshal(data, &cfg.Web)
	err = json.Unmarshal(data, &cfg.Bitcoin)
	err = json.Unmarshal(data, &cfg.Db)
	err = json.Unmarshal(data, &cfg.Storage)
	err = json.Unmarshal(data, &cfg.SpyNode)
	err = json.Unmarshal(data, &cfg.NodeStorage)
	err = json.Unmarshal(data, &cfg.RpcNode)

	if err != nil {
		return err
	}

	return nil
}

// SafeConfig masks sensitive config values
func SafeConfig(cfg Config) *Config {
	cfgSafe := cfg

	if len(cfgSafe.Oracle.Key) > 0 {
		cfgSafe.Oracle.Key = "*** Masked ***"
	}
	if len(cfgSafe.Storage.Secret) > 0 {
		cfgSafe.Storage.Secret = "*** Masked ***"
	}
	if len(cfgSafe.NodeStorage.Secret) > 0 {
		cfgSafe.NodeStorage.Secret = "*** Masked ***"
	}
	if len(cfgSafe.RpcNode.Password) > 0 {
		cfgSafe.RpcNode.Password = "*** Masked ***"
	}

	return &cfgSafe
}

// Environment returns configuration sourced from environment variables
func Environment() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("API", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}