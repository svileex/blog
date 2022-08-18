package utils

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	userRegex = regexp.MustCompile("[a-z]+$")
)

func ValidateLogin(fl validator.FieldLevel) bool {
	return userRegex.Match([]byte(fl.Field().String()))
}

func WriteJsonToResponse(w http.ResponseWriter, statusCode int, json []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(json)
}

func WriteErrorToResponse(w http.ResponseWriter, statusCode int, errorMsg string) {
	resp, _ := json.Marshal(map[string]string{"error": errorMsg})
	WriteJsonToResponse(w, statusCode, resp)
}
