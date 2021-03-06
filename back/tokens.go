package main

import (
	"database/sql"
)

func IsTokenExists(db *sql.DB, username, token string) (bool, error) {
	var cnt int
	err := db.QueryRow("select count(*) from apiTokens where apiToken=? and username=?", token, username).Scan(&cnt)
	if err != nil {
		return false, nil
	}

	return cnt > 0, nil
}

func InsertNewToken(db *sql.DB, username, teamName, token string) error {
	tokenExists, err := IsTokenExists(db, username, token)
	if err != nil {
		return err
	}
	if tokenExists {
		return nil
	}
	_, err = db.Exec("insert into apiTokens(apiToken, teamName, username) values (?, ?, ?)", token, teamName, username)
	if err != nil {
		return err
	}
	return nil
}

func GetAllTokens(db *sql.DB, username string) ([]string, error) {
	rows, err := db.Query("select apiToken from apiTokens where username=?", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]string, 0, 10)
	for rows.Next() {
		var token string
		if err = rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func DeleteToken(db *sql.DB, username string, teamName string) error {
	_, err := db.Exec("delete from apiTokens where username=? and teamName=?", username, teamName)
	return err
}
