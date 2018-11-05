package dbutil

import (
	"database/sql"

	"github.com/go-kit/kit/log"
)

// DBManager db管理器
type DBManager interface {
	Register(dbName, dataSourceName string, maxIdleConns, maxOpenConns int) error
	Query(dbName, sql string, args ...interface{}) ([]map[string]interface{}, error)
	Exec(dbName string, sql string, args ...interface{}) (sql.Result, error)
	Prepare(dbName string, sql string) (*sql.Stmt, error)
	Begin(dbName string) (*sql.Tx, error)
	DB(dbName string) (db *sql.DB, ok bool)
	SetLogger(logger log.Logger) error
}
