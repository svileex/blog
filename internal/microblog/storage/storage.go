package storage

type Storage interface {
	AddPost(*Post) error
	AddUser(*User) error
	GetPost(string) (Post, error)
	GetUserByLogin(string) (User, error)
	ValidateUser(string, string) error
	GetUserById(string) (User, error)
	GetAllPostsOfUser(string) ([]Post, error)
}
