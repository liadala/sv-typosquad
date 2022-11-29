package database

import (
	"database/sql"
	"log"
	"sync"

	_ "modernc.org/sqlite"
)

type Client struct {
	db   *sql.DB
	lock sync.Mutex
}

func InitDB() Client {
	c := Client{}
	c.lock.Lock()
	defer c.lock.Unlock()

	d, err := sql.Open("sqlite", "semperurl.db")
	if err != nil {
		log.Fatal(err)
	}
	_, err = d.Exec(`PRAGMA journal_mode=WAL;`)
	if err != nil {
		log.Fatal(err)
	}
	c.db = d

	var querys []string = []string{
		`CREATE TABLE IF NOT EXISTS"messages" (
			"id"	TEXT NOT NULL,
			"timestamp"	INTEGER NOT NULL,
			"userid"	TEXT NOT NULL,
			"username"	TEXT NOT NULL,
			"message"	TEXT NOT NULL,
			"deleted"	INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY("id")
		);`,
	}

	err = bulkExec(c.db, querys)
	if err != nil {
		log.Fatal(err)
	}

	return Client{
		db: d,
	}
}

func bulkExec(db *sql.DB, in []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, v := range in {
		_, err = tx.Exec(v)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
