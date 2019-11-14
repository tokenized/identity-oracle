package db

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type Database interface {
	Querier
	Ping() error
	Beginx() (DatabaseTx, error)
	Close() error
}

type DatabaseTx interface {
	Querier
	Commit() error
	Rollback() error
}

type Querier interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	Queryx(query string, args ...interface{}) (*sqlx.Rows, error)
	Prepare(query string) (*sql.Stmt, error)
	Rebind(query string) string
	PrepareNamed(query string) (*sqlx.NamedStmt, error)
}

type db struct {
	*sqlx.DB
}

func (db *db) Beginx() (DatabaseTx, error) {
	return db.DB.Beginx()
}

func (db *db) Close() error {
	return db.DB.Close()
}
