package utils

import (
	"log"
	"net/http"
)

type ErrorLogger struct {
	functionName string
}

func (l ErrorLogger) CheckError(err error, w http.ResponseWriter, msg string, status int) error {
	if err != nil {
		log.Print(l.functionName + msg)
		WriteErrorToResponse(w, status, msg)
	}

	return err
}

func NewErrorLogger(funcName string) *ErrorLogger {
	return &ErrorLogger{functionName: funcName + ": "}
}