package storage

// TODO: all ids are strings, is it bad?

type Post struct {
	Id       string        `json:"id"`
	Text     string    `json:"text"`
	AuthorId string        `json:"authorId"`
	Time     string `json:"createdAt"`
}

func NewPost() *Post {
	return &Post{}
}