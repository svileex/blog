package storage

import "encoding/json"

type UserCredentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func GetCredentials(reqBody []byte) (*UserCredentials, error) {
	var userCredentials UserCredentials

	return &userCredentials, json.Unmarshal(reqBody, &userCredentials)
}