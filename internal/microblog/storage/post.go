package storage

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Post struct {
	// Hex value of _id in mongo
	Id   string
	Text string
	// Hex value of user _id in mongo
	AuthorId string
	Time     string
}

func NewPost() *Post {
	return &Post{}
}

type storageDbTranferObject struct {
	Id       primitive.ObjectID `bson:"_id,omitempty"`
	Text     string             `bson:"text"`
	AuthorId primitive.ObjectID `bson:"authorId"`
}

type FrontendHandlerTransferObject struct {
	Id       string `json:"id"`
	Text     string `json:"text"`
	AuthorId string `json:"authorId"`
	Time     string `json:"createdAt"`
}

func NewFrontendDto() *FrontendHandlerTransferObject {
	return &FrontendHandlerTransferObject{}
}

// TODO: лучше сделать специальный методы? Так непредсказуемый эффект, делаешь json.Marshal, а там ...
func (p Post) MarshalBSON() ([]byte, error) {
	authorId, err := primitive.ObjectIDFromHex(p.AuthorId)

	if err != nil {
		return make([]byte, 0), err
	}

	return bson.Marshal(storageDbTranferObject{Text: p.Text, AuthorId: authorId})
}

func (p Post) MarshalJSON() ([]byte, error) {
	return json.Marshal(FrontendHandlerTransferObject{
		Id:       base64.URLEncoding.EncodeToString([]byte(p.Id)),
		Text:     p.Text,
		AuthorId: p.AuthorId,
		Time:     p.Time,
	})
}

func (p *Post) UnmarshalJSON(data []byte) error {
	var tmp FrontendHandlerTransferObject
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if p.Id != "" {
		postId, err := base64.URLEncoding.DecodeString(tmp.Id)

		if err != nil {
			return err
		}

		p.Id = string(postId)
	}
	p.Text = tmp.Text
	p.AuthorId = tmp.AuthorId
	p.Time = tmp.Time

	return nil
}

func (p *Post) UnmarshalBSON(data []byte) error {
	var tmp storageDbTranferObject
	if err := bson.Unmarshal(data, &tmp); err != nil {
		return err
	}

	p.Id = tmp.Id.Hex()
	p.AuthorId = tmp.AuthorId.Hex()
	p.Text = tmp.Text
	p.Time = tmp.Id.Timestamp().UTC().Format(time.RFC3339)

	return nil
}
