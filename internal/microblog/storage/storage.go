package storage

import "context"

type Storage interface {
	AddPost(context.Context, *Post) error
	AddUser(context.Context, *User) error
	GetPost(context.Context, string) (*Post, error)
	GetUserByLogin(context.Context, string) (*User, error)
	GetUserById(context.Context, string) (*User, error)
	GetPostsFrom(ctx context.Context, postId string, userId string, size int) ([]Post, string, error)
	GetFirstPosts(ctx context.Context, userId string, size int) ([]Post, string, error)
}
