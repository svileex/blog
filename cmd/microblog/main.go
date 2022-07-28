package main

import (
	"blog/internal/microblog"
	"os"
)

func main() {
	srv := microblog.NewMicroblogServer(os.Getenv("MONGO_URL"))
	srv.StartNewMicrobologServer(8081)
}
