package tests

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/tokenized/config"
	"github.com/tokenized/identity-oracle/internal/platform/db"
	"github.com/tokenized/identity-oracle/internal/platform/web"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"

	"github.com/google/uuid"
)

// Success and failure markers.
const (
	Success = "\u2713"
	Failed  = "\u2717"
)

// Test owns state for running/shutting down tests.
type Test struct {
	Log       *log.Logger
	MasterDB  *db.DB
	WebConfig *web.Config
}

// New is the entry point for tests.
func New() *Test {

	// =========================================================================
	// Logging

	log := log.New(os.Stdout, "TEST : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// ============================================================
	// Configuration

	ctx := context.Background()

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
		log.Fatalf("main : Register DB : %v", err)
	}

	mockStorage := newMockStorage()
	masterDB.SetStorage(mockStorage)

	// ============================================================
	// Web Config

	webConfig := &web.Config{
		RootURL: "http://localhost:8081",
		Net:     bitcoin.MainNet,
		IsTest:  true,
	}

	return &Test{log, masterDB, webConfig}
}

// TearDown is used for shutting down tests. Calling this should be
// done in a defer immediately after calling New.
func (t *Test) TearDown() {
	t.MasterDB.Close()
}

// Context returns an app level context for testing.
func Context() context.Context {
	values := web.Values{
		TraceID: uuid.New().String(),
		Now:     time.Now(),
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
