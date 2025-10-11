package dbUtils

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// NewDB returns a new database pool. only supports mysql
func NewDB(userName, password, host string, port int, dbName string) (dbPool *sql.DB, err error) {
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=true&loc=Local", userName, password, host, port, dbName)

	if dbPool, err = sql.Open("mysql", dsn); err != nil {
		return nil, fmt.Errorf("NewDB(): failed to open database, error: %w", err)
	}

	if err = dbPool.Ping(); err != nil {
		return nil, fmt.Errorf("NewDB(): failed to ping database, error: %w", err)
	}

	return dbPool, nil
}
