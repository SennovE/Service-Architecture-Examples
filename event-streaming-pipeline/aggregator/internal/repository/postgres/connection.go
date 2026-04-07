package postgres

import (
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func Connect(host string, port int, user string, password string, dbName string) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", user, password, host, port, dbName)
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}
