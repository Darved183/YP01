package database

import (
	"os"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func Load() (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite", "db.db")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = ON;
		PRAGMA busy_timeout = 5111;
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	var userIDCol int
	err = db.Get(&userIDCol, `SELECT COUNT(*) FROM pragma_table_info('cart') WHERE lower(name) = 'user_id'`)
	if err != nil {
		db.Close()
		return nil, err
	}
	if userIDCol == 0 {
		if _, err = db.Exec(`DROP TABLE IF EXISTS cart`); err != nil {
			db.Close()
			return nil, err
		}
	}

	_, err = db.Exec(`
		DROP TABLE IF EXISTS OrderItem;
		DROP TABLE IF EXISTS "Order";
		DROP TABLE IF EXISTS Photo;
		DROP TABLE IF EXISTS User;
		DROP TABLE IF EXISTS Role;
		DROP TABLE IF EXISTS Product;
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	schema, err := os.ReadFile("internal/database/migrations/init.sql")
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err = db.Exec(string(schema)); err != nil {
		db.Close()
		return nil, err
	}

	roles, err := os.ReadFile("internal/database/migrations/AddRole.sql")
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err = db.Exec(string(roles)); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
