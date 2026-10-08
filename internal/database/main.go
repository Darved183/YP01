package database

import(
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func Load() (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite", "db.db")
	if err != nil {
		return nil,err
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	_,err = db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = ON;
		PRAGMA busy_timeout = 5111;
	`)
	if err != nil{
		return  nil,err
	}

	return db,nil
}