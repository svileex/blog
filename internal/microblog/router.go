package microblog

import (
	"blog/internal/microblog/handler"
	"blog/internal/microblog/storage"
	"blog/internal/microblog/storage/mongostorage"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type MicroblogServer struct {
	r       *mux.Router
	storage *storage.Storage
}

func NewRouter(s *storage.Storage) *mux.Router {
	r := mux.NewRouter()
	h := handler.NewHandler(s)

	r.HandleFunc("/api/v1/register", h.RegisterNewUser).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/login", h.Login)
	r.HandleFunc("/api/v1/posts", h.AddPost).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/posts/{postId}", h.GetPost).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/users/{userId}/posts", h.GetUserPosts).Methods(http.MethodGet)

	return r
}

func NewMicroblogServer(mongoUrl string) *MicroblogServer {
	s, err := mongostorage.NewMongoStorage(mongoUrl)

	if err != nil {
		panic(fmt.Errorf("can't create mongo storage - %w", err))
	}

	return &MicroblogServer{r: NewRouter(&s), storage: &s}
}

func (srv *MicroblogServer) StartNewMicrobologServer(port int) {
	server := &http.Server{
		Handler:      srv.r,
		Addr:         "0.0.0.0:" + strconv.Itoa(port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	server.ListenAndServe()
}
