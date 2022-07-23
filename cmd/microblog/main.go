package main

import (
	"hw1-milestone1/internal/microblog"
)

func main() {
	srv := microblog.NewMicroblogServer()
	srv.StartNewMicrobologServer(8081)
}
