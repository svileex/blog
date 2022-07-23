package storage

type User struct {
	Login        string
	Id           string
	PasswordHash []byte
}
