package handler

import (
	"blog/internal/microblog/storage"
	"blog/internal/microblog/utils"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
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

var (
	registerLogger     = utils.NewErrorLogger("RegisterNewUser")
	loginLogger        = utils.NewErrorLogger("Login")
	addPostLogger      = utils.NewErrorLogger("AddPost")
	getPostLogger      = utils.NewErrorLogger("GetPost")
	getUserPostsLogger = utils.NewErrorLogger("GetUserPosts")

	passwordSalt = "abcdefgh12345"
)

// Todo: interface
func (h *Handler) RegisterNewUser(w http.ResponseWriter, req *http.Request) {
	reqBody, err := io.ReadAll(req.Body)

	if registerLogger.CheckError(err, w, "can't read body", http.StatusBadRequest) != nil {
		return
	}

	userCredentials, err := storage.GetCredentials(reqBody)

	if registerLogger.CheckError(err, w, "can't parse body", http.StatusBadRequest) != nil {
		return
	}

	pwdHash, _ := bcrypt.GenerateFromPassword([]byte(userCredentials.Password + passwordSalt), 10)
	newUser := storage.User{
		Login:        userCredentials.Login,
		PasswordHash: pwdHash,
	}

	validate := validator.New()
	validate.RegisterValidation("login", utils.ValidateLogin)
	if registerLogger.CheckError(validate.Struct(newUser), w, "wrong login fmt", http.StatusBadRequest) != nil {
		return
	}

	err = (*h.s).AddUser(context.Background(), &newUser)
	if registerLogger.CheckError(err, w, "something went wrong", http.StatusBadRequest) != nil {
		return
	}

	resp, _ := json.Marshal(map[string]string{"id": newUser.Id})
	utils.WriteJsonToResponse(w, http.StatusOK, resp)
}

func (h *Handler) AddPost(w http.ResponseWriter, req *http.Request) {
	reqBody, err := io.ReadAll(req.Body)

	if addPostLogger.CheckError(err, w, "can't read body", http.StatusBadRequest) != nil {
		return
	}

	reqUserId, exist := (*req).Header["System-Design-User-Id"]

	if !exist || len(reqUserId) != 1 {
		log.Print("addPost: wrong id format")
		utils.WriteErrorToResponse(w, http.StatusUnauthorized, "wrong id format")
		return
	}

	user, err := (*h.s).GetUserById(context.Background(), reqUserId[0])

	if addPostLogger.CheckError(err, w, "user not found", http.StatusUnauthorized) != nil {
		return
	}

	var post storage.Post
	err = json.Unmarshal(reqBody, &post)

	if addPostLogger.CheckError(err, w, "wrong format", http.StatusBadRequest) != nil {
		return
	}

	post.AuthorId = user.Id
	(*h.s).AddPost(context.Background(), &post)

	resp, _ := json.Marshal(post)

	utils.WriteJsonToResponse(w, http.StatusOK, resp)
}

// нужен ли указатель?
func (h *Handler) GetPost(w http.ResponseWriter, req *http.Request) {
	postId, ex := mux.Vars(req)["postId"]

	if !ex {
		log.Print("getPost: bad post id")
		utils.WriteErrorToResponse(w, http.StatusNotFound, "bad post id")
		return
	}

	post, err := (*h.s).GetPost(context.Background(), postId)

	if getPostLogger.CheckError(err, w, "post not found", http.StatusNotFound) != nil {
		return
	}

	resp, _ := json.Marshal(post)
	utils.WriteJsonToResponse(w, http.StatusOK, resp)
}

func (h *Handler) GetUserPosts(w http.ResponseWriter, req *http.Request) {
	page := req.FormValue("page")
	size, err := strconv.Atoi(req.FormValue("size"))

	if req.FormValue("size") == "" {
		size = 10
	} else if getUserPostsLogger.CheckError(err, w, "size must be numer", http.StatusBadRequest) != nil {
		return
	}

	if size < 0 || size > 100 {
		log.Print("getUserPosts: bad size value")
		utils.WriteErrorToResponse(w, http.StatusBadRequest, "0 <= size <= 100")
		return
	}

	userId, ex := mux.Vars(req)["userId"]
	if !ex {
		log.Print("getUserPosts: wrong url format")
		utils.WriteErrorToResponse(w, http.StatusBadRequest, "wrong url format")
		return
	}

	var posts []storage.Post
	var nextPageToken string

	if page == "" {
		posts, nextPageToken, err = (*h.s).GetFirstPosts(context.Background(), userId, size)
	} else {
		posts, nextPageToken, err = (*h.s).GetPostsFrom(context.Background(), page, userId, size)
	}

	if getUserPostsLogger.CheckError(err, w, "wrong url format", http.StatusBadRequest) != nil {
		return
	}

	mapForResponse := map[string]interface{}{"posts": posts}

	if nextPageToken != "" {
		mapForResponse["nextPage"] = nextPageToken
	}

	resp, _ := json.Marshal(mapForResponse)
	utils.WriteJsonToResponse(w, http.StatusOK, resp)
}

// TODO: messages in errors
func (h *Handler) Login(w http.ResponseWriter, req *http.Request) {
	reqBody, err := io.ReadAll(req.Body)

	if loginLogger.CheckError(err, w, "can't read body", http.StatusBadRequest) != nil {
		return
	}

	userCredentials, err := storage.GetCredentials(reqBody)

	if loginLogger.CheckError(err, w, "can't can't getCredentials", http.StatusBadRequest) != nil {
		return
	}

	user, err := (*h.s).GetUserByLogin(context.Background(), userCredentials.Login)

	if loginLogger.CheckError(err, w, "user not found", http.StatusBadRequest) != nil {
		return
	}

	err = bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(userCredentials.Password))
	if loginLogger.CheckError(err, w, "wrong password", http.StatusBadRequest) != nil {
		return
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    user.Login,
		ExpiresAt: time.Now().Add(time.Hour * 1).Unix(),
	})

	token, err := claims.SignedString([]byte("secretKeycxvsdfdsfsdsdffsdsdfdsfsdfsdfsfdfsfdssfd"))

	if loginLogger.CheckError(err, w, "wrong password", http.StatusBadRequest) != nil {
		return
	}

	response, _ := json.Marshal(map[string]string{"token": token})

	utils.WriteJsonToResponse(w, http.StatusOK, response)
}
