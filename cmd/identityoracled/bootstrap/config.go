package bootstrap

import "time"

// Config is used to hold all runtime configuration.
//
// The same Config is used regardless of if the server is a regualar HTTP
// server, or a Lambda function.
type Config struct {
	Env    string `envconfig:"ENV" json:"ENV"`
	Oracle struct {
		Key                               string `envconfig:"KEY" json:"KEY" masked:"true"`
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
		AccessKey string `envconfig:"STORAGE_ACCESS_KEY" json:"STORAGE_ACCESS_KEY" masked:"true"`
		Secret    string `envconfig:"STORAGE_SECRET" json:"STORAGE_SECRET" masked:"true"`
		Bucket    string `default:"standalone" envconfig:"STORAGE_BUCKET" json:"STORAGE_BUCKET"`
		Root      string `default:"./tmp" envconfig:"STORAGE_ROOT" json:"STORAGE_ROOT"`
	}
}
