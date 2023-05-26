package tests

import (
	"context"
	"time"

	"github.com/tokenized/config"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/storage"
)

// Success and failure markers.
const (
	Success = "\u2713"
	Failed  = "\u2717"
)

// Test owns state for running/shutting down tests.
type Test struct {
	MasterDB  *db.DB
	WebConfig *web.Config
}

// New is the entry point for tests.
func New() *Test {

	// ============================================================
	// Configuration

	ctx := logger.ContextWithLogger(context.Background(), false, false, "")

	cfg := &Config{}
	// load config using sane fallbacks
	if err := config.LoadConfig(ctx, cfg); err != nil {
		logger.Fatal(ctx, "main : LoadConfig : %v", err)
	}

	// ============================================================
	// Start Database

	masterDB, err := db.New(&db.DBConfig{
		Driver: cfg.Db.Driver,
		URL:    cfg.Db.URL,
	}, nil)
	if err != nil {
		logger.Fatal(ctx, "main : Register DB : %v", err)
	}

	mockStorage := storage.NewMockStorage()
	masterDB.SetStorage(mockStorage)

	// ============================================================
	// Web Config

	webConfig := &web.Config{
		RootURL: "http://localhost:8081",
		Net:     bitcoin.MainNet,
		IsTest:  true,
	}

	return &Test{masterDB, webConfig}
}

// TearDown is used for shutting down tests. Calling this should be
// done in a defer immediately after calling New.
func (t *Test) TearDown() {
	t.MasterDB.Close()
}

// Context returns an app level context for testing.
func Context() context.Context {
	values := web.Values{
		Now: time.Now(),
	}

	ctx := context.WithValue(context.Background(), web.KeyValues, &values)

	return web.ContextWithValues(ctx, bitcoin.MainNet, true)
}

type Config struct {
	Db struct {
		Driver string `default:"postgres" envconfig:"DB_DRIVER" json:"DB_DRIVER"`
		URL    string `default:"user=foo dbname=bar sslmode=disable" envconfig:"DB_URL" json:"DB_URL"`
	}
}
