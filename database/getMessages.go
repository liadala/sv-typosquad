package database

import "regexp"

func (c *Client) CrawlMessagesByRegex(reg string) ([]string, error) {
	var reply []string = make([]string, 0)

	r, err := regexp.Compile(reg)
	if err != nil {
		return nil, err
	}

	rows, err := c.db.Query(`SELECT message FROM messages WHERE deleted = 0;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var message string
	for rows.Next() {
		err := rows.Scan(&message)
		if err != nil {
			return nil, err
		}
		reply = append(reply, r.FindAllString(message, -1)...)
	}

	return reply, nil
}
