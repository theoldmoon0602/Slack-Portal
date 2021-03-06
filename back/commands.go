package main

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/nlopes/slack"
	"golang.org/x/net/websocket"
)

type Session struct {
	Db       *sql.DB
	Ws       *websocket.Conn
	User     *User
	Commands map[string](chan string)
}

func (sess *Session) Register(username, password string) (string, error) {
	var err error
	sess.User, err = CreateUser(sess.Db, username, password)
	if err != nil {
		return "", err
	}
	sessionKey, err := StoreUser(sess.Db, sess.User.Username)
	if err != nil {
		return "", err
	}
	return sessionKey, nil
}
func (sess *Session) Resume(sessionKey string) (string, error) {
	var err error
	sess.User, err = GetUserByKey(sess.Db, sessionKey)
	if err != nil {
		return "", err
	}
	newSessionKey, err := UpdateSessionKey(sess.Db, sessionKey)
	if err != nil {
		return "", err
	}

	err = sess.RestoreListeningTokens()
	if err != nil {
		return "", err
	}

	return newSessionKey, nil
}

func (sess *Session) Login(username, password string) (string, error) {
	var err error
	sess.User, err = GetUser(sess.Db, username)
	if err != nil {
		return "", err
	}
	if sess.User.Password != password {
		return "", errors.New("Username and Password are mismatch")
	}

	sessionKey, err := StoreUser(sess.Db, sess.User.Username)
	if err != nil {
		return "", err
	}
	err = sess.RestoreListeningTokens()
	if err != nil {
		return "", err
	}
	return sessionKey, nil
}
func (sess *Session) TokenListen(token string) error {
	api := slack.New(token)
	teamInfo, err := api.GetTeamInfo()
	if err != nil {
		return err
	}

	err = InsertNewToken(sess.Db, sess.User.Username, teamInfo.Name, token)
	if err != nil {
		return err
	}

	sess.Commands[teamInfo.Name] = make(chan string, 2)
	go ListenSlack(sess.Ws, api, sess.Commands[teamInfo.Name])

	// 監視をはじめた通知
	type ListenTeam struct {
		Name   string `json:"name"`
		Domain string `json:"domain"`
	}
	websocket.JSON.Send(sess.Ws, Response{
		TypeName: "Listen",
		Value: ListenTeam{
			Name:   teamInfo.Name,
			Domain: teamInfo.Domain,
		},
	})

	return nil
}
func (sess *Session) NewToken(token string) error {
	if sess.User == nil {
		return errors.New("User is nil")
	}
	exist, err := IsTokenExists(sess.Db, sess.User.Username, token)
	if err != nil {
		return err
	}
	if exist {
		return errors.New("Token already registered")
	}

	err = sess.TokenListen(token)
	if err != nil {
		return err
	}
	return nil
}
func (sess *Session) RestoreListeningTokens() error {
	tokens, err := GetAllTokens(sess.Db, sess.User.Username)
	if err != nil {
		return err
	}
	fmt.Println(tokens)
	for _, token := range tokens {
		fmt.Println(token)
		err = sess.TokenListen(token)
		fmt.Println(err)
	}

	return nil
}

// StopListen 監視を終了します
func (sess *Session) StopListen(teamName string) error {
	sess.Commands[teamName] <- "stop"
	err := DeleteToken(sess.Db, sess.User.Username, teamName)
	if err != nil {
		return err
	}
	return nil
}
