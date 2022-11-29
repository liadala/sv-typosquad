package database

import "log"

func (c *Client) DeleteMessage(id string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	stmt, err := c.db.Prepare(`UPDATE messages SET deleted = 1 WHERE id = ?;`)
	if err != nil {
		log.Println(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(id)
	if err != nil {
		log.Println(err)
	}
}
