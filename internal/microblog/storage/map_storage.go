package storage

import (
	"errors"
	"strconv"
	"sync"
)

type mapStorage struct {
	usersMu sync.RWMutex
	usersId int
	users   []User
	postsMu sync.RWMutex
	postsId int
	posts   []Post
}

func (m *mapStorage) AddPost(post *Post) error {
	m.postsMu.Lock()
	post.Id = strconv.Itoa(m.postsId)
	m.posts = append(m.posts, *post)
	m.postsId++
	m.postsMu.Unlock()

	return nil
}

func (m *mapStorage) AddUser(newUser *User) error {
	m.usersMu.Lock() // TODO: lock guard? Normal usage of mutex

	for _, user := range m.users {
		if user.Login == newUser.Login {
			m.usersMu.Unlock()
			return errors.New("user with this login already exist")
		}
	}
	newUser.Id = strconv.Itoa(m.usersId) // TODO: id not string
	m.usersId++
	m.users = append(m.users, *newUser)
	m.usersMu.Unlock()

	return nil
}

func (m *mapStorage) GetPost(id string) (Post, error) {
	m.postsMu.RLock()
	for _, p := range m.posts {
		if p.Id == id {
			m.postsMu.RUnlock()
			return p, nil
		}
	}
	m.postsMu.RUnlock()

	return *NewPost(), errors.New("not found")
}

func (m *mapStorage) GetUser(id string) error {
	m.usersMu.RLock()
	for _, user := range m.users {
		if user.Id == id {
			m.usersMu.RUnlock()
		}
	}
	m.usersMu.RUnlock()
	return nil
}

func (m *mapStorage) ValidateUser(string, string) error {
	return nil
}

func NewMapStorage() *mapStorage {
	return &mapStorage{
		sync.RWMutex{},
		0,
		make([]User, 0),
		sync.RWMutex{},
		0,
		make([]Post, 0)}
}

func (m *mapStorage) GetUserByLogin(targetLogin string) (User, error) {
	m.postsMu.RLock()
	for _, user := range m.users {
		if user.Login == targetLogin {
			return user, nil
		}
	}
	m.postsMu.RUnlock()

	return User{"", "", make([]byte, 0)}, nil // return pointer?
}

// TODO: normal interface what is GetUser?
func (m *mapStorage) GetUserById(targetId string) (User, error) {
	m.usersMu.RLock()
	for _, user := range m.users {
		if string(user.Id) == targetId {
			m.usersMu.RUnlock()
			return user, nil
		}
	}
	m.usersMu.RUnlock()

	return User{"", "", make([]byte, 0)}, nil // return pointer?
}

func (m *mapStorage) GetAllPostsOfUser(id string) ([]Post, error) {
	m.postsMu.RLock()
	posts := make([]Post, 0)

	for i := len(m.posts) - 1; i >= 0; i-- {
		if m.posts[i].AuthorId == id { //TODO: id from string
			posts = append(posts, m.posts[i])
		}
	}
	m.postsMu.RUnlock()

	return posts, nil
}
