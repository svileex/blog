package microblog

import (
	"encoding/json"
	"hw1-milestone1/internal/microblog/storage"
	"net/http"
)

func getCredentials(reqBody []byte) (*storage.UserCredentials, error) {
	var userCredentials storage.UserCredentials

	return &userCredentials, json.Unmarshal(reqBody, &userCredentials)
}

// нужен ли указатель?
func writeError(w http.ResponseWriter, statusCode int, respBody []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(respBody)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}