package hw1_milestone1

import (
	"blog/internal/microblog"
	"blog/internal/microblog/storage"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	openapi3_routers "github.com/getkin/kin-openapi/routers"
	openapi3_legacy "github.com/getkin/kin-openapi/routers/legacy"
	"github.com/stretchr/testify/suite"
)

type ApiSuite struct {
	suite.Suite

	client        http.Client
	apiSpecRouter openapi3_routers.Router
}

//go:embed microblog.yaml
var microblogApi []byte

var ctx = context.Background()

func (s *ApiSuite) SetupSuite() {
	srv := microblog.NewMicroblogServer(os.Getenv("MONGO_URL"))

	go func() {
		srv.StartNewMicrobologServer(8081)
	}()

	spec, err := openapi3.NewLoader().LoadFromData(microblogApi)
	s.Require().NoError(err)
	s.Require().NoError(spec.Validate(ctx))
	router, err := openapi3_legacy.NewRouter(spec)
	s.Require().NoError(err)
	s.apiSpecRouter = router
	s.client.Transport = s.specValidating(http.DefaultTransport)
}

func (s *ApiSuite) specValidating(transport http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		log.Println("Send HTTP request:")
		reqBody := s.printReq(req)

		// validate request
		route, params, err := s.apiSpecRouter.FindRoute(req)
		s.Require().NoError(err)
		reqDescriptor := &openapi3filter.RequestValidationInput{
			Request:     req,
			PathParams:  params,
			QueryParams: req.URL.Query(),
			Route:       route,
		}
		s.Require().NoError(openapi3filter.ValidateRequest(ctx, reqDescriptor))

		// do request
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
		resp, err := transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		log.Println("Got HTTP response:")
		respBody := s.printResp(resp)

		// Validate response against OpenAPI spec
		s.Require().NoError(openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{
			RequestValidationInput: reqDescriptor,
			Status:                 resp.StatusCode,
			Header:                 resp.Header,
			Body:                   io.NopCloser(bytes.NewReader(respBody)),
		}))

		return resp, nil
	})
}

func (s *ApiSuite) printReq(req *http.Request) []byte {
	body := s.readAll(req.Body)

	req.Body = io.NopCloser(bytes.NewReader(body))
	s.Require().NoError(req.Write(os.Stdout))
	fmt.Println()

	req.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func (s *ApiSuite) printResp(resp *http.Response) []byte {
	body := s.readAll(resp.Body)

	resp.Body = io.NopCloser(bytes.NewReader(body))
	s.Require().NoError(resp.Write(os.Stdout))
	fmt.Println()

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func (s *ApiSuite) readAll(in io.Reader) []byte {
	if in == nil {
		return nil
	}
	data, err := ioutil.ReadAll(in)
	s.Require().NoError(err)
	return data
}

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func registerUser(s *ApiSuite, login string) string {
	reqBody := io.NopCloser(strings.NewReader(
		fmt.Sprintf( /* language=json */ `{"login": "%s", "password": "%s"}`, login, "test")))
	resp, err := s.client.Post("http://localhost:8081/api/v1/register", "application/json", reqBody)

	s.Require().NoError(err)
	rawBody, err := ioutil.ReadAll(resp.Body)
	s.Require().NoError(err)

	s.Assert().Equal(200, resp.StatusCode)
	type registerResponse struct {
		Id string `json:"id"`
	}
	var response registerResponse
	s.Require().NoError(json.Unmarshal(rawBody, &response))
	s.Require().NotEmpty(response.Id)

	return response.Id
}

func addPost(s *ApiSuite, postText, userIdForCheck string) storage.FrontendHandlerTransferObject {
	post := storage.NewFrontendDto()
	post.Text = postText
	reqRawBody, _ := json.Marshal(map[string]string{"text": postText})
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8081/api/v1/posts", bytes.NewReader(reqRawBody))
	req.Header.Add("System-Design-User-Id", userIdForCheck)
	req.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	rawRespBody, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Require().NoError(json.Unmarshal(rawRespBody, &post))

	s.Require().Equal(post.AuthorId, userIdForCheck)
	s.Require().Equal(post.Text, postText)

	_, err = time.Parse(time.RFC3339, post.Time)

	s.Require().NoError(err)
	return *post
}

func (s *ApiSuite) TestRegisterAndCreatePost() {
	var userId string

	// TODO: adress to constant in server
	s.Run("registerUser", func() {
		userId = registerUser(s, "testregisterandcreatepost")
	})

	var regPost storage.FrontendHandlerTransferObject

	s.Run("addPost", func() {
		regPost = addPost(s, "not aboba", userId)
	})

	s.Run("getPost", func() {
		resp, err := s.client.Get("http://localhost:8081/api/v1/posts/" + regPost.Id)
		s.Require().NoError(err)

		rawRespBody, _ := io.ReadAll(resp.Body)
		postFromResp := storage.NewFrontendDto()

		s.Require().NoError(json.Unmarshal(rawRespBody, &postFromResp))
		s.Require().Equal(postFromResp.AuthorId, regPost.AuthorId)
		s.Require().Equal(postFromResp.Text, regPost.Text)
		s.Require().Equal(postFromResp.Time, regPost.Time)
		s.Require().Equal(postFromResp.Id, regPost.Id)
	})
}

func getLastPosts(s *ApiSuite, size int, page, url string) ([]storage.Post, string, int) {
	req, err := http.NewRequest(http.MethodGet, url, io.NopCloser(strings.NewReader("")))
	s.Require().NoError(err)
	params := req.URL.Query()
	params.Add("size", strconv.Itoa(size))

	if page != "" {
		params.Add("page", page)
	}

	req.URL.RawQuery = params.Encode()

	resp, err := s.client.Do(req)
	s.Require().NoError(err)

	rawBody, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	type usersPostAnswer struct {
		Posts    []storage.Post `json:"posts"`
		NextPage string         `json:"NextPage"`
	}
	var respBody usersPostAnswer

	s.Require().NoError(json.Unmarshal(rawBody, &respBody))

	return respBody.Posts, respBody.NextPage, resp.StatusCode
}

func (s *ApiSuite) TestUsersPosts() {
	var userId string
	s.Run("registerUser", func() {
		userId = registerUser(s, "testusersposts")
	})

	for i := 0; i < 10; i++ {
		s.Run("addPostInUsersPosts", func() {
			addPost(s, strconv.Itoa(i), userId)
		})
		time.Sleep(1 * time.Second)
		// TODO: время в бд слишком сильно округляется 
	}

	url := fmt.Sprintf("http://localhost:8081/api/v1/users/%s/posts", userId)

	s.Run("lastPosts3", func() {
		posts, nextPage, _ := getLastPosts(s, 3, "", url)
		s.Require().Equal(3, len(posts))
		s.Require().Equal(posts[0].Text, "9")
		s.Require().Equal(posts[1].Text, "8")
		s.Require().Equal(posts[2].Text, "7")
		s.Require().NotEqual(nextPage, "")

		posts, nextPage, _ = getLastPosts(s, 4, nextPage, url)
		s.Require().Equal(len(posts), 4)
		s.Require().Equal(posts[0].Text, "6")
		s.Require().Equal(posts[1].Text, "5")
		s.Require().Equal(posts[2].Text, "4")
		s.Require().Equal(posts[3].Text, "3")
		s.Require().NotEqual(nextPage, "")

		posts, nextPage, _ = getLastPosts(s, 4, nextPage, url)
		s.Require().Equal(3, len(posts))
		s.Require().Equal(posts[0].Text, "2")
		s.Require().Equal(posts[1].Text, "1")
		s.Require().Equal(posts[2].Text, "0")
		s.Require().Equal(nextPage, "")
	})

	s.Run("lastPostBadArgs", func() {
		_, _, code := getLastPosts(s, 1, "21211212", url)
		s.Require().Equal(400, code)
	})

}

func TestAPI(t *testing.T) {
	suite.Run(t, &ApiSuite{})
}
