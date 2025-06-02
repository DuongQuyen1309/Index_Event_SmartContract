package db

import (
	"database/sql"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

var DB *bun.DB

func ConnectDB() {
	dns := os.Getenv("DNS_DATABASE")
	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dns)))
	DB = bun.NewDB(pgdb, pgdialect.New())
	DB.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
}
