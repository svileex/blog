package storage

type User struct {
	Login        string
	Id           string `bson:"_id,omitempty"`
	PasswordHash []byte
}
