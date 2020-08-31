package db

import (
	"context"
	sqldb "database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/tokenized/pkg/storage"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

var (
	// ErrInvalidDBProvided is returned in the event that an uninitialized db is
	// used to perform actions against.
	ErrInvalidDBProvided = errors.New("Invalid DB provided")

	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Entity not found")
)

// DB is a collection of support for different DB technologies. Currently
// only PgSql has been implemented. We want to be able to access the raw
// database support for the given DB so an interface does not work. Each
// database is too different.
type DB struct {
	database  Database
	storage   storage.Storage
	session   Database
	sessionTx DatabaseTx
}

// StorageConfig is geared towards "bucket" style storage, where you have a
// specific root (the Bucket).
type StorageConfig struct {
	Bucket string
	Root   string
}

// DBConfig geared towards relational database.
type DBConfig struct {
	Driver string
	URL    string
}

// New returns a new DB value for use with PgSql based on a registered
// master session.
func New(dbc *DBConfig, sc *StorageConfig) (*DB, error) {

	// PostgreSQL Database
	var newDB Database
	if dbc != nil {
		pgsql, err := sqlx.Connect(dbc.Driver, dbc.URL)
		if err != nil {
			return nil, err
		}

		if err = pgsql.Ping(); err != nil {
			return nil, err
		}

		newDB = &db{pgsql}
	}

	// S3 Storage
	var store storage.Storage
	if sc != nil {
		storeConfig := storage.NewConfig(sc.Bucket, sc.Root)
		if strings.ToLower(sc.Bucket) == "standalone" {
			store = storage.NewFilesystemStorage(storeConfig)
		} else {
			store = storage.NewS3Storage(storeConfig)
		}
	}

	db := DB{
		database:  newDB,
		storage:   store,
		session:   nil,
		sessionTx: nil,
	}

	return &db, nil
}

// StatusCheck validates the DB status good.
func (db *DB) StatusCheck(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "platform.DB.StatusCheck")
	defer span.End()

	if db.database != nil {
		if err := db.database.Ping(); err != nil {
			return err
		}
	}

	if db.storage != nil {
		// Generate a random key that is almost certain not to exist.
		uid, _ := uuid.NewRandom()
		ts := time.Now().UnixNano()
		k := fmt.Sprintf("healthcheck/%v/%v", uid, ts)

		// We should receive a "not found" error for a non-existant key.
		if _, err := db.Fetch(ctx, k); err != ErrNotFound {
			return err
		}
	}

	return nil
}

// Close closes a DB value being used with PgSql. If a session is available
// this should be closed instead of the master instance, instead we flag it
// as closed as a signal to prevent further incorrect use.
func (db *DB) Close() {
	if db.session != nil {
		db.session = nil
		return
	}

	if db.database != nil {
		db.database.Close()
	}
}

// Copy returns a new DB value for use within the app based on master session.
// The session is only needed for use with transactions for now but we will
// set up the interface to allow support any generic database type.
func (db *DB) Copy() *DB {
	newDB := DB{
		database:  db.database,
		storage:   db.storage,
		session:   db.database,
		sessionTx: nil,
	}

	return &newDB
}

// GetActiveDB returns a database object dependant on whether a transaction is active.
func (db *DB) GetActiveDB() Querier {
	if db.sessionTx != nil {
		return db.sessionTx
	}

	if db.session != nil {
		return db.session
	}

	return db.database
}

// SetStorage replaces the storage value. This is needed for unit testing.
func (db *DB) SetStorage(storage storage.Storage) {
	db.storage = storage
}

// GetStorage returns the storage value.
func (db *DB) GetStorage() storage.Storage {
	return db.storage
}

// -------------------------------------------------------------------------
// Database

// binaryInterface is an interface for classes that should be put in the database as binary values.
type binaryInterface interface {
	Bytes() []byte
}

// prepareArguments checks if arguments should be converted to binary.
func prepareArguments(args []interface{}) []interface{} {
	result := make([]interface{}, 0, len(args))

	// Convert any "binary" values and check for nil
	for _, arg := range args {
		rv := reflect.ValueOf(arg)
		if rv.Kind() == reflect.Ptr && rv.IsNil() {
			result = append(result, nil)
			continue
		}

		b, ok := arg.(binaryInterface)
		if ok {
			result = append(result, b.Bytes())
			continue
		}

		result = append(result, arg)
	}

	return result
}

// Execute is used to execute PgSql commands.
func (db *DB) Execute(ctx context.Context, sql string, args ...interface{}) error {
	ctx, span := trace.StartSpan(ctx, "platform.DB.Execute")
	defer span.End()

	activeDB := db.GetActiveDB()
	if activeDB == nil {
		return errors.Wrap(ErrInvalidDBProvided, "database == nil")
	}

	stmt, err := activeDB.Prepare(activeDB.Rebind(sql))
	if err != nil {
		return err
	}

	if len(args) == 0 {
		// cannot pass empty args to Exec.
		_, err = stmt.Exec()
	} else {
		pargs := prepareArguments(args)
		_, err = stmt.Exec(pargs...)
	}

	return err
}

// Query provides a string version of the value
func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (*sqlx.Rows, error) {
	ctx, span := trace.StartSpan(ctx, "platform.DB.Query")
	defer span.End()

	activeDB := db.GetActiveDB()
	if activeDB == nil {
		return nil, errors.Wrap(ErrInvalidDBProvided, "database == nil")
	}

	pargs := prepareArguments(args)
	rows, err := activeDB.Queryx(activeDB.Rebind(sql), pargs...)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// Select using this DB. Any placeholder parameters are replaced with supplied args.
func (db *DB) Select(ctx context.Context, model interface{}, sql string, args ...interface{}) error {
	ctx, span := trace.StartSpan(ctx, "platform.DB.Select")
	defer span.End()

	activeDB := db.GetActiveDB()
	if activeDB == nil {
		return errors.Wrap(ErrInvalidDBProvided, "database == nil")
	}

	pargs := prepareArguments(args)
	if err := activeDB.Select(model, activeDB.Rebind(sql), pargs...); err != nil {
		if err == sqldb.ErrNoRows {
			err = ErrNotFound
		}

		return err
	}

	return nil
}

// SelectIn using a WHERE IN style query with this db.
func (db *DB) SelectIn(ctx context.Context, model interface{}, sql string, args ...interface{}) error {
	ctx, span := trace.StartSpan(ctx, "platform.DB.Select")
	defer span.End()

	activeDB := db.GetActiveDB()
	if activeDB == nil {
		return errors.Wrap(ErrInvalidDBProvided, "database == nil")
	}

	pargs := prepareArguments(args)
	inSql, inArgs, err := sqlx.In(sql, pargs...)
	if err != nil {
		return err
	}

	if err := activeDB.Select(model, activeDB.Rebind(inSql), inArgs...); err != nil {
		if err == sqldb.ErrNoRows {
			err = ErrNotFound
		}

		return err
	}

	return nil
}

// Get using this DB. Any placeholder parameters are replaced with supplied args. An error is returned if the result set is empty.
func (db *DB) Get(ctx context.Context, model interface{}, sql string, args ...interface{}) error {
	ctx, span := trace.StartSpan(ctx, "platform.DB.Get")
	defer span.End()

	activeDB := db.GetActiveDB()
	if activeDB == nil {
		return errors.Wrap(ErrInvalidDBProvided, "database == nil")
	}

	pargs := prepareArguments(args)
	if err := activeDB.Get(model, activeDB.Rebind(sql), pargs...); err != nil {
		if err == sqldb.ErrNoRows {
			err = ErrNotFound
		}

		return err
	}

	return nil
}

// -------------------------------------------------------------------------
// Database Transactions

// BeginTransaction starts a new database transaction.
func (db *DB) BeginTransaction() {
	if db.session == nil {
		panic("Attempt to perform transaction on master instance, you must create a Copy() first")
	}

	tx, err := db.session.Beginx()
	if err != nil {
		panic(err)
	}

	db.sessionTx = tx
}

// Commit the pending transaction to the database.
func (db *DB) Commit() error {
	err := db.sessionTx.Commit()
	db.sessionTx = nil
	return err
}

// Rollback the pending transaction.
func (db *DB) Rollback() error {
	err := db.sessionTx.Rollback()
	db.sessionTx = nil
	return err
}

// -------------------------------------------------------------------------
// Storage

// Put something in storage
func (db *DB) Put(ctx context.Context, key string, body []byte) error {
	if db.storage == nil {
		return errors.Wrap(ErrInvalidDBProvided, "storage == nil")
	}

	return db.storage.Write(ctx, key, body, nil)
}

// Fetch something from storage
func (db *DB) Fetch(ctx context.Context, key string) ([]byte, error) {
	if db.storage == nil {
		return nil, errors.Wrap(ErrInvalidDBProvided, "storage == nil")
	}

	b, err := db.storage.Read(ctx, key)
	if err != nil {
		if err == storage.ErrNotFound {
			err = ErrNotFound
		}

		return nil, err
	}

	return b, nil
}

// Remove something from storage
func (db *DB) Remove(ctx context.Context, key string) error {
	if db.storage == nil {
		return errors.Wrap(ErrInvalidDBProvided, "storage == nil")
	}

	return db.storage.Remove(ctx, key)
}
