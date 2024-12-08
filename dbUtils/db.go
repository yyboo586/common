package dbUtils

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	User   string
	Passwd string
	Host   string
	Port   int
	DBName string
}

// NewDB returns a new database pool. only supports mysql
func NewDB(config *Config) (dbPool *sql.DB, err error) {
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=true", config.User, config.Passwd, config.Host, config.Port, config.DBName)

	if dbPool, err = sql.Open("mysql", dsn); err != nil {
		return nil, fmt.Errorf("NewDB(): failed to open database, error: %w", err)
	}

	if err = dbPool.Ping(); err != nil {
		return nil, fmt.Errorf("NewDB(): failed to ping database, error: %w", err)
	}

	return dbPool, nil
}
