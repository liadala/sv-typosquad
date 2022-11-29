package database

import (
	"time"
)

func (c *Client) AddMessage(id string, timestamp time.Time, userid string, username string, message string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	tx, err := c.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO messages (id, timestamp, userid, username, message)VALUES(?,?,?,?,?);`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	stmt.Exec(id, timestamp.UTC().Unix(), userid, username, message)

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
