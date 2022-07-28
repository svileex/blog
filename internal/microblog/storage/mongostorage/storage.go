package mongostorage

import (
	"blog/internal/microblog/storage"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const dbName = "blog"

type mongoStorage struct {
	posts *mongo.Collection
	users *mongo.Collection
}

func NewMongoStorage(mongoUrl string) (storage.Storage, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoUrl))

	if err != nil {
		return nil, fmt.Errorf("agboba can't connect to mongo - %w", err)
	}

	posts := client.Database(dbName).Collection("posts")
	posts.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.D{{"authorId", 1}}})

	users := client.Database(dbName).Collection("users")
	users.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.D{{"login", 1}}})

	return &mongoStorage{
		posts: posts,
		users: users,
	}, nil
}

func (s *mongoStorage) AddPost(ctx context.Context, post *storage.Post) error {
	id, err := s.posts.InsertOne(ctx, *post)

	if err != nil {
		return fmt.Errorf("can't insert post - %w", err)
	}

	objId := id.InsertedID.(primitive.ObjectID)
	post.Id = objId.Hex()
	post.Time = objId.Timestamp().UTC().Format(time.RFC3339)

	return nil
}

func (s *mongoStorage) AddUser(ctx context.Context, user *storage.User) error {
	id, err := s.users.InsertOne(ctx, user)

	if err != nil {
		return fmt.Errorf("can't insert user - %w", err)
	}

	user.Id = id.InsertedID.(primitive.ObjectID).Hex()

	return nil
}

func (s *mongoStorage) GetPost(ctx context.Context, postIdBase64 string) (*storage.Post, error) {
	var findResult storage.Post
	postId, err := decodeBase64PostId(postIdBase64)

	if err != nil {
		return nil, fmt.Errorf("can't decode this id, id: %s - %w", postIdBase64, err)
	}

	err = s.posts.FindOne(ctx, bson.M{"_id": postId}).Decode(&findResult)

	if err != nil {
		return nil, fmt.Errorf("can't find post with id %s - %w", postIdBase64, err)
	}

	return &findResult, nil
}

func (s *mongoStorage) GetUserByLogin(ctx context.Context, login string) (*storage.User, error) {
	var findResult storage.User
	err := s.posts.FindOne(ctx, bson.M{"Login": login}).Decode(&findResult)

	if err != nil {
		return nil, fmt.Errorf("can't find user with login %s - %w", login, err)
	}

	return &findResult, nil
}

func (s *mongoStorage) GetUserById(ctx context.Context, idHex string) (*storage.User, error) {
	var findResult storage.User
	objId, err := primitive.ObjectIDFromHex(idHex)

	if err != nil {
		return nil, fmt.Errorf("bad user id - %w", err)
	}

	err = s.users.FindOne(ctx, bson.M{"_id": objId}).Decode(&findResult)

	if err != nil {
		return nil, fmt.Errorf("can't find user with id %s - %w", idHex, err)
	}

	return &findResult, nil
}

// TODO: если больше постов нет?
func (s *mongoStorage) GetPostsFrom(ctx context.Context, postId string, authorId string, size int) ([]storage.Post, string, error) {
	postIdObj, err := decodeBase64PostId(postId)

	if err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't decode postId: %w", err)
	}

	authorIdObj, err := primitive.ObjectIDFromHex(authorId)

	if err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't decode authorId: %w", err)
	}

	opts := options.Find().SetSort(bson.M{"_id": -1}).SetLimit(int64(size))
	cur, err := s.posts.Find(ctx, bson.M{"authorId": authorIdObj, "_id": bson.M{"$lte": postIdObj}}, opts)

	if err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't find posts: %w", err)
	}

	posts := make([]storage.Post, 0)
	if err := cur.All(ctx, &posts); err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't get data from cursor: %w", err)
	}

	if len(posts) == 0 {
		return posts, "", nil
	}

	nextPageToken, err := s.getNextPageToken(ctx, &posts, authorIdObj)

	if err != nil {
		return make([]storage.Post, 0), "", err
	}

	return posts, nextPageToken, nil
}

func (s *mongoStorage) GetFirstPosts(ctx context.Context, userIdHex string, size int) ([]storage.Post, string, error) {
	userId, err := primitive.ObjectIDFromHex(userIdHex)

	if err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't decode objId: %w", err)
	}

	opts := options.Find().SetSort(bson.M{"_id": -1}).SetLimit(int64(size))
	cur, err := s.posts.Find(ctx, bson.M{"authorId": userId}, opts)

	if err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't find posts: %w", err)
	}

	posts := make([]storage.Post, 0)
	if err := cur.All(ctx, &posts); err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't get data from cursor: %w", err)
	}

	if len(posts) == 0 {
		return posts, "", nil
	}

	nextPageToken, err := s.getNextPageToken(ctx, &posts, userId)

	if err != nil {
		return make([]storage.Post, 0), "", err
	}

	return posts, nextPageToken, nil
}

// len(posts) must be not zero
func (s *mongoStorage) getNextPageToken(ctx context.Context, posts *[]storage.Post, userId primitive.ObjectID) (string, error) {
	lastId, _ := primitive.ObjectIDFromHex((*posts)[len(*posts)-1].Id)

	nextPost, err := s.getNextPost(ctx, userId, lastId)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("can't get next post: %w", err)
	}

	return base64.URLEncoding.EncodeToString([]byte(nextPost.Id)), nil
}

func (s *mongoStorage) getNextPost(ctx context.Context, userId, postId primitive.ObjectID) (*storage.Post, error) {
	var nextPost storage.Post
	nextPostOpts := options.FindOne().SetSort(bson.M{"_id": -1})
	err := s.posts.FindOne(ctx,
		bson.M{
			"authorId": userId,
			"_id":      bson.M{"$lt": postId},
		}, nextPostOpts).Decode(&nextPost)

	return &nextPost, err
}

func decodeBase64PostId(id string) (*primitive.ObjectID, error) {
	objectIdBytes, err := base64.URLEncoding.DecodeString(id)

	if err != nil {
		log.Print("can't decode base64 id")
		return nil, err
	}

	objId, err := primitive.ObjectIDFromHex(string(objectIdBytes))

	return &objId, err
}
