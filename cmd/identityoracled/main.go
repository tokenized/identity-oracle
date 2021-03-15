package main

import (
	"context"
	"os"
	"strings"

	"github.com/tokenized/identity-oracle/cmd/identityoracled/bootstrap"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/rpcnode"
	"github.com/tokenized/pkg/spynode"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	buildUser    = "unknown"
)

func main() {

	// ---------------------------------------------------------------------------------------------
	// Logging

	logPath := os.Getenv("LOG_FILE_PATH")

	logConfig := logger.NewConfig(strings.ToUpper(os.Getenv("DEVELOPMENT")) == "TRUE",
		strings.ToUpper(os.Getenv("LOG_FORMAT")) == "TEXT", logPath)

	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(spynode.SubSystem)

	ctx := logger.ContextWithLogConfig(context.Background(), logConfig)

	logger.Info(ctx, "Build %v (%v on %v)", buildVersion, buildUser, buildDate)

	bootstrap.Run(ctx, nil)
}
