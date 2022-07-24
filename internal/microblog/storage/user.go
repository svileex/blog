package storage

type User struct {
	Login        string
	Id           string // objId in hex
	PasswordHash []byte
}
