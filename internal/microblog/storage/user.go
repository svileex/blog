package storage

type User struct {
	Login        string `validate:"login"`
	Id           string `bson:"_id,omitempty"`
	PasswordHash []byte
}

