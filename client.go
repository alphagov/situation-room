package main

import (
	"encoding/base64"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"

	calendar "google.golang.org/api/calendar/v3"
)

const scope = "https://www.googleapis.com/auth/calendar.readonly"
const authURL = "https://accounts.google.com/o/oauth2/auth"
const tokenURL = "https://accounts.google.com/o/oauth2/token"

type ApiClient struct {
	ClientId   string
	EncodedKey string
	Token      *oauth2.Token
}

func (c ApiClient) GetToken() *oauth2.Token {

	keyBytes, err := base64.StdEncoding.DecodeString(c.EncodedKey)
	if err != nil {
		log.Fatal("Error decoding private key:", err)
	}

	conf := &jwt.Config{
		Email:      c.ClientId,
		PrivateKey: keyBytes,
		TokenURL:   tokenURL,
		Scopes:     []string{scope},
	}

	log.Printf("Requesting new access token.\n")
	token, err := conf.TokenSource(oauth2.NoContext).Token()

	if err != nil {
		panic(err)
	}

	log.Printf("New access token acquired.\n")
	return token
}

func (c ApiClient) Client() *http.Client {
	config := &oauth2.Config{
		ClientID:     c.ClientId,
		ClientSecret: "notasecret",
		Scopes:       []string{scope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	return config.Client(oauth2.NoContext, c.Token)
}

func (c ApiClient) Api() *calendar.Service {
	client := c.Client()

	svc, _ := calendar.New(client)
	return svc
}
