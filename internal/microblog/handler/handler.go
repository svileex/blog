package handler

import (
	"blog/internal/microblog/storage"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	s *storage.Storage
}

func NewHandler(s *storage.Storage) *Handler {
	return &Handler{s: s}
}

// TODO: validation in storage
func validateUserLogin(user storage.User) error {
	matched, err := regexp.Match(`^[a-z]+$`, []byte(user.Login))

	if !matched || err != nil {
		return errors.New("login should match `^[a-Z]|[0-9]$`")
	}

	return nil
}

func writeJsonToResponse(w *http.ResponseWriter, statusCode int, json []byte) {
	(*w).Header().Set("Content-Type", "application/json")
	(*w).WriteHeader(statusCode)
	(*w).Write(json)
}

func writeErrorToResponse(w *http.ResponseWriter, statusCode int, errorMsg string) {
	resp, _ := json.Marshal(map[string]string{"error": errorMsg})
	writeJsonToResponse(w, statusCode, resp)
}

func (h *Handler) RegisterNewUser(w http.ResponseWriter, req *http.Request) {
	reqBody, err := io.ReadAll(req.Body)

	if err != nil {
		log.Print("RegisterNewUser: can't read body")
		writeErrorToResponse(&w, http.StatusBadRequest, "can't read body")
		return
	}

	userCredentials, err := storage.GetCredentials(reqBody)

	if err != nil {
		log.Print("register: can't parse body. Body: " + string(reqBody))
		writeErrorToResponse(&w, http.StatusBadRequest, "bad request body")
		return
	}

	pwdHash, _ := bcrypt.GenerateFromPassword([]byte(userCredentials.Password), 10)
	newUser := storage.User{
		Login:        userCredentials.Login,
		Id:           "",
		PasswordHash: pwdHash,
	}

	if err := validateUserLogin(newUser); err != nil {
		log.Print("register: bad login")
		writeErrorToResponse(&w, http.StatusBadRequest, "wrong login format")
		return
	}

	if err := (*h.s).AddUser(context.Background(), &newUser); err != nil {
		log.Print("register: can't add to user to storage")
		writeErrorToResponse(&w, http.StatusBadRequest, "ooops storage dead")
		return
	}

	resp, _ := json.Marshal(map[string]string{"id": newUser.Id})
	writeJsonToResponse(&w, http.StatusOK, resp)
}

func (h *Handler) Login(w http.ResponseWriter, req *http.Request) {
	reqBody, err := io.ReadAll(req.Body)

	if err != nil {
		log.Print("RegisterNewUser: can't read body")
		writeErrorToResponse(&w, http.StatusBadRequest, "can't read body")
		return
	}

	userCredentials, err := storage.GetCredentials(reqBody)

	if err != nil {
		log.Print("login: can't getCredentials. Body: " + string(reqBody))
		writeErrorToResponse(&w, http.StatusBadRequest, "bad request body")
		return
	}

	user, err := (*h.s).GetUserByLogin(context.Background(), userCredentials.Login)

	if err != nil {
		log.Print("login: can't find user by login")
		writeErrorToResponse(&w, http.StatusBadRequest, "no user with this login")
		return
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(userCredentials.Password)); err != nil {
		log.Print("login: can't compareHash")
		writeErrorToResponse(&w, http.StatusBadRequest, "wrong password")
		return
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    user.Login,
		ExpiresAt: time.Now().Add(time.Hour * 1).Unix(),
	})

	token, err := claims.SignedString([]byte("secretKeycxvsdfdsfsdsdffsdsdfdsfsdfsdfsfdfsfdssfd"))
	response, err := json.Marshal(map[string]string{"token": token})
	// TODO: check errors

	writeJsonToResponse(&w, http.StatusOK, response)
}

func (h *Handler) AddPost(w http.ResponseWriter, req *http.Request) {
	reqBody, err := io.ReadAll(req.Body)

	if err != nil {
		log.Print("RegisterNewUser: can't read body")
		writeErrorToResponse(&w, http.StatusBadRequest, "can't read body")
		return
	}

	reqUserId, exist := (*req).Header["System-Design-User-Id"]

	if !exist || len(reqUserId) != 1 {
		log.Print("addPost: wrong id format")
		writeErrorToResponse(&w, http.StatusUnauthorized, "wrong id format")
		return
	}

	user, err := (*h.s).GetUserById(context.Background(), reqUserId[0])

	if err != nil {
		log.Print("addPost: no user with this id")
		writeErrorToResponse(&w, http.StatusUnauthorized, "no user with this id")
		return
	}

	var post storage.Post

	if json.Unmarshal(reqBody, &post) != nil {
		log.Print("addPost: can't parse body")
		writeErrorToResponse(&w, http.StatusBadRequest, "wrong post format")
		return
	}

	post.AuthorId = user.Id
	post.Time = time.Now().UTC().Format(time.RFC3339)
	(*h.s).AddPost(context.Background(), &post)
	// TODO: time from database

	resp, _ := json.Marshal(post)

	writeJsonToResponse(&w, http.StatusOK, resp)
}

func (h *Handler) GetPost(w http.ResponseWriter, req *http.Request) {
	postId, ex := mux.Vars(req)["postId"]

	if !ex {
		log.Print("getPost: bad post id")
		writeErrorToResponse(&w, http.StatusNotFound, "bad post id")
		return
	}

	post, err := (*h.s).GetPost(context.Background(), postId)

	if err != nil {
		log.Print("getPost: post not found")
		writeErrorToResponse(&w, http.StatusNotFound, "post not found")
		return
	}

	resp, _ := json.Marshal(post)
	writeJsonToResponse(&w, http.StatusOK, resp)
}

func (h *Handler) GetUserPosts(w http.ResponseWriter, req *http.Request) {
	page := req.FormValue("page")
	size, err := strconv.Atoi(req.FormValue("size"))

	if req.FormValue("size") == "" {
		size = 10
	} else if err != nil {
		log.Print("getUserPosts: size must be number")
		writeErrorToResponse(&w, http.StatusBadRequest, "size must be numer")
		return
	}

	if size < 10 || size > 100 {
		log.Print("getUserPosts: bad size value")
		writeErrorToResponse(&w, http.StatusBadRequest, "10 <= size <= 100")
		return
	}

	userId, ex := mux.Vars(req)["userId"]
	if !ex {
		log.Print("getUserPosts: wrong url format")
		writeErrorToResponse(&w, http.StatusBadRequest, "wrong url format")
		return
	}

	var posts []storage.Post
	var nextPageToken string

	if page == "" {
		posts, nextPageToken, err = (*h.s).GetFirstPosts(context.Background(), userId, size)
	} else {
		posts, nextPageToken, err = (*h.s).GetPostsFrom(context.Background(), page, userId, size)
	}

	// TODO: logger который принимает ошибку и тд
	// TODO: specification
	if err != nil {
		log.Print("getUserPosts: can't get posts")
		writeErrorToResponse(&w, http.StatusBadRequest, "wrong url format")
		return
	}

	var mapForResponse map[string]interface{}
	if nextPageToken == "" {
		mapForResponse = map[string]interface{}{"posts": posts}
	} else {
		mapForResponse = map[string]interface{}{"posts": posts, "nextPage": nextPageToken}
	}

	resp, _ := json.Marshal(mapForResponse)
	writeJsonToResponse(&w, http.StatusOK, resp)
}
