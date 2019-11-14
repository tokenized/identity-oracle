package config

import (
	"os"

	"github.com/tokenized/nexus-api/pkg/lambdaproxy"
)

// Storage returns configuration sourced from file storage (S3)
func Storage(region string, bucket string, filename string) (*Config, error) {
	b, err := lambdaproxy.LoadConfig(region, bucket, filename)
	if err != nil {
		return nil, err
	}

	cfg, err := Environment()
	if err != nil {
		return nil, err
	}

	if err := unmarshalNested(b, cfg); err != nil {
		return nil, err
	}

	if cfg.Storage.AccessKey != os.Getenv("AWS_ACCESS_KEY") &&
		len(os.Getenv("AWS_ACCESS_KEY")) > 0 {

		// we are running in lambda, use the credentials in the environment
		cfg.Storage.AccessKey = os.Getenv("AWS_ACCESS_KEY")
		cfg.Storage.Secret = os.Getenv("AWS_SECRET_KEY")
	}

	return cfg, nil
}
