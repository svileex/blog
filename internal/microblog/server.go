package microblog

import (
	"encoding/json"
	"errors"
	"hw1-milestone1/internal/microblog/storage"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type MicroblogServer struct {
	r       *mux.Router
	storage storage.Storage
}

type ParsedHttpRequest struct {
	Body   *[]byte
	Header *http.Header
	Url    *url.URL
}

func validateUserLogin(user storage.User) error {
	matched, err := regexp.Match(`^[a-z]+$`, []byte(user.Login))

	if !matched || err != nil {
		return errors.New("login should match `^[a-Z]|[0-9]$`")
	}

	return nil
}

func getHttpHandler(bodyHandler func(ParsedHttpRequest, *storage.Storage) ([]byte, int), s *storage.Storage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		reqBody, err := io.ReadAll(req.Body)

		if err != nil {
			log.Print("handleHttpRequset: can't read body")
			writeError(w, http.StatusBadRequest, errorToJson("can't read body"))
			return
		}

		respBody, statusCode := bodyHandler(ParsedHttpRequest{&reqBody, &req.Header, req.URL}, s)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(respBody)
	}
}

func errorToJson(errorBody string) []byte {
	resp, _ := json.Marshal(map[string]string{"error": errorBody})

	return resp
}

func registerNewUser(s *storage.Storage) func(w http.ResponseWriter, req *http.Request) {
	return getHttpHandler(func(req ParsedHttpRequest, s *storage.Storage) ([]byte, int) {
		userCredentials, err := getCredentials(*req.Body)

		if err != nil {
			log.Print("register: can't parse body. Body: " + string(*req.Body))
			return errorToJson("bad request body"), http.StatusBadRequest
		}

		pwdHash, _ := bcrypt.GenerateFromPassword([]byte(userCredentials.Password), 10)
		newUser := storage.User{
			Login:        userCredentials.Login,
			Id:           "",
			PasswordHash: pwdHash,
		}

		if err := validateUserLogin(newUser); err != nil {
			log.Print("register: bad login")
			return errorToJson("wrong login format"), http.StatusBadRequest
		}

		if err := (*s).AddUser(&newUser); err != nil {
			log.Print("register: can't add to user to storage")
			return errorToJson("ooops storage dead"), http.StatusBadRequest
		}

		resp, _ := json.Marshal(map[string]string{"id": newUser.Id})
		return resp, http.StatusOK
	}, s)
}

func login(s *storage.Storage) func(w http.ResponseWriter, req *http.Request) {
	return getHttpHandler(func(req ParsedHttpRequest, s *storage.Storage) ([]byte, int) {
		userCredentials, err := getCredentials(*req.Body)

		if err != nil {
			log.Print("login: can't getCredentials. Body: " + string(*req.Body))
			return errorToJson("bad request body"), 400
		}

		user, err := (*s).GetUserByLogin(userCredentials.Login)

		if err != nil {
			log.Print("login: can't find user by login")
			return errorToJson("no user with this login"), 400
		}

		if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(userCredentials.Password)); err != nil {
			log.Print("login: can't compareHash")
			return errorToJson("wrong password"), 400
		}

		claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
			Issuer:    user.Login,
			ExpiresAt: time.Now().Add(time.Hour * 1).Unix(),
		})

		token, err := claims.SignedString([]byte("secretKeycxvsdfdsfsdsdffsdsdfdsfsdfsdfsfdfsfdssfd"))
		response, err := json.Marshal(map[string]string{"token": token})
		// TODO: check errors
		return response, http.StatusAccepted
	}, s)
}

func addPost(s *storage.Storage) func(w http.ResponseWriter, req *http.Request) {
	return getHttpHandler(func(req ParsedHttpRequest, s *storage.Storage) ([]byte, int) {
		reqUserId, exist := (*req.Header)["System-Design-User-Id"]

		if !exist || len(reqUserId) != 1 {
			log.Print("addPost: wrong id format")
			return errorToJson("wrong id format"), 401
		}

		user, err := (*s).GetUserById(reqUserId[0])

		if err != nil {
			log.Print("addPost: no user with this id")
			return errorToJson("no user with this id"), 401
		}

		var post storage.Post

		if json.Unmarshal(*req.Body, &post) != nil {
			log.Print("addPost: can't parse body")
			return errorToJson("wrong post format"), 401
		}

		post.AuthorId = user.Id
		post.Time = time.Now().UTC().Format(time.RFC3339)
		(*s).AddPost(&post)

		resp, _ := json.Marshal(post)

		return resp, 200
	}, s)
}

func getPost(s *storage.Storage) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		postId, ex := mux.Vars(req)["postId"]

		if !ex {
			log.Print("getPost: bad post id")
			writeError(w, 404, errorToJson("bad post id"))
			return
		}

		post, err := (*s).GetPost(postId)

		if err != nil {
			log.Print("getPost: post not found")
			writeError(w, 404, errorToJson("post not found"))
			return
		}

		resp, _ := json.Marshal(post)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(resp)
	}
}

func getUserPosts(s *storage.Storage) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		userId, ex := mux.Vars(req)["userId"]

		if !ex {
			log.Print("getUserPosts: wrong url format")
			writeError(w, 400, errorToJson("wrong url format"))
			return
		}

		page, err := strconv.Atoi(req.FormValue("page"))

		if err != nil && req.FormValue("page") != "" {
			log.Print("getUserPosts: page must be number")
			writeError(w, 400, errorToJson("bad page identifier"))
			return
		}

		size, err := strconv.Atoi(req.FormValue("size"))

		if err != nil && req.FormValue("size") != "" {
			log.Print("getUserPosts: size must be number")
			writeError(w, 400, errorToJson("bad size identifier"))
			return
		} else if req.FormValue("size") == "" {
			size = 10
		}

		// TODO: logger with error and message
		// TODO: все посты плохо их слишком много
		// TODO: проверить, что size нормальный
		// TODO: хранить курсор в нормальном порядке
		// TODO: например id последнего поста который получили - курсор
		// TODO: get first ten and get next to
		posts, err := (*s).GetAllPostsOfUser(userId)

		if err != nil {
			log.Print("getUserPosts: getAllPosts")
			writeError(w, 400, errorToJson("bad size identifier"))
			return
		}

		if len(posts) < page {
			log.Print("getUserPosts: bad page")
			writeError(w, 400, errorToJson("bad page"))
			return
		}

		postsForResponse := posts[page:min(len(posts), page+size)]
		var mapForJson map[string]interface{}

		if len(posts) <= page+size {
			mapForJson = map[string]interface{}{"posts": postsForResponse}
		} else {
			mapForJson = map[string]interface{}{"posts": postsForResponse, "nextPage": strconv.Itoa(page + size)}
		}

		resp, _ := json.Marshal(mapForJson)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(resp)
	}
}

func NewMicroblogServer() *MicroblogServer {
	s := MicroblogServer{mux.NewRouter(), storage.NewMapStorage()}
	s.r.HandleFunc("/api/v1/register", registerNewUser(&s.storage)).Methods(http.MethodPost)
	s.r.HandleFunc("/api/v1/login", login(&s.storage))
	s.r.HandleFunc("/api/v1/posts", addPost(&s.storage)).Methods(http.MethodPost)
	s.r.HandleFunc("/api/v1/posts/{postId}", getPost(&s.storage)).Methods(http.MethodGet)
	s.r.HandleFunc("/api/v1/users/{userId}/posts", getUserPosts(&s.storage)).Methods(http.MethodGet)

	return &s
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
