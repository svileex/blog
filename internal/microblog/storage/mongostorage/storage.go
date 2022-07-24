package mongostorage

import (
	"blog/internal/microblog/storage"
	"context"
	"encoding/base64"
	"fmt"
	"log"

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
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUrl))

	if err != nil {
		return nil, fmt.Errorf("can't connect to mongo - %w", err)
	}

	// TODO: validation for posts?
	// TODO: https://www.mongodb.com/docs/manual/reference/method/db.createCollection/#mongodb-method-db.createCollection
	// TODO: https://www.mongodb.com/docs/manual/core/schema-validation/
	posts := client.Database(dbName).Collection("posts")
	posts.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.D{{"AuthorId", 1}, {"Time", -1}}})

	users := client.Database(dbName).Collection("users")
	users.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.D{{"Login", 1}}})

	// TODO: разобраться с контекстом - goplayground, нужен ли тут контекст с таймаутом?
	return &mongoStorage{
		posts: posts,
		users: users,
	}, nil
}

func (s *mongoStorage) AddPost(ctx context.Context, post *storage.Post) error {
	id, err := s.posts.InsertOne(ctx, post)

	if err != nil {
		return fmt.Errorf("can't insert post - %w", err)
	}

	post.Id = encodeObjectIdToBase64(id.InsertedID.(primitive.ObjectID))

	return nil
}

func (s *mongoStorage) AddUser(ctx context.Context, user *storage.User) error {
	id, err := s.users.InsertOne(ctx, user)

	if err != nil {
		return fmt.Errorf("can't insert user - %w", err)
	}

	// TODO: set time from id timestamp
	// TODO: плохая идея, лучше заставить монгу саму ставить время
	user.Id = string(id.InsertedID.(primitive.ObjectID).Hex())

	return nil
}

// TODO: зачем это шаманство с индексами?
// TODO: indexes
// TODO: mongo guides and many to many with users and posts
func (s *mongoStorage) GetPost(ctx context.Context, id string) (*storage.Post, error) {
	var findResult storage.Post
	objectId, err := decodeBase64PostId(id)

	if err != nil {
		return nil, fmt.Errorf("can't decode this id, id: %s - %w", id, err)
	}

	err = s.posts.FindOne(ctx, bson.M{"_id": objectId}).Decode(&findResult)

	if err != nil {
		return nil, fmt.Errorf("can't find post with id %s - %w", id, err)
	}

	findResult.Id = encodeObjectIdToBase64(*objectId)
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

// TODO: copypaste
func (s *mongoStorage) GetUserById(ctx context.Context, id string) (*storage.User, error) {
	var findResult storage.User
	objId, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return nil, fmt.Errorf("bad user id - %w", err)
	}

	err = s.posts.FindOne(ctx, bson.M{"_id": objId}).Decode(&findResult)

	if err != nil {
		return nil, fmt.Errorf("can't find user with id %s - %w", id, err)
	}

	return &findResult, nil
}

// TODO: если больше постов нет?
func (s *mongoStorage) GetPostsFrom(ctx context.Context, postId string, authorId string, size int) ([]storage.Post, string, error) {
	objId, err := decodeBase64PostId(postId)

	if err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't decode objId: %w", err)
	}
	opts := options.Find().SetSort(bson.M{"Time": -1}).SetLimit(int64(size))
	cur, err := s.posts.Find(ctx, bson.M{"AuthorId": authorId, "_id": bson.M{"$gte": objId}}, opts)

	if err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't find posts: %w", err)
	}

	posts := make([]storage.Post, 0)
	if err := cur.All(ctx, &posts); err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't get data from cursor: %w", err)
	}

	lastId, _ := decodeBase64PostId(posts[len(posts)-1].Id)

	nextPost, err := s.getNextPost(ctx, *objId, *lastId)
	if err != nil {
		return posts, "", fmt.Errorf("something wrong with nextPostRaw: %w", err)
	}

	return posts, nextPost.Id, nil
}

// TODO: copypaste
func (s *mongoStorage) GetFirstPosts(ctx context.Context, userIdHex string, size int) ([]storage.Post, string, error) {
	userId, err := primitive.ObjectIDFromHex(userIdHex)

	if err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't decode objId: %w", err)
	}

	opts := options.Find().SetSort(bson.M{"Time": -1}).SetLimit(int64(size))
	cur, err := s.posts.Find(ctx, bson.M{"AuthorId": userId}, opts)

	if err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't find posts: %w", err)
	}

	posts := make([]storage.Post, 0)
	if err := cur.All(ctx, &posts); err != nil {
		return make([]storage.Post, 0), "", fmt.Errorf("can't get data from cursor: %w", err)
	}

	lastId, _ := decodeBase64PostId(posts[len(posts)-1].Id)

	nextPost, err := s.getNextPost(ctx, userId, *lastId)
	if err != nil {
		return posts, "", fmt.Errorf("something wrong with nextPostRaw: %w", err)
	}

	return posts, nextPost.Id, nil
}

func (s *mongoStorage) getNextPost(ctx context.Context, userId, postId primitive.ObjectID) (*storage.Post, error) {
	var nextPost storage.Post
	nextPostOpts := options.FindOne().SetSort(bson.M{"Time": -1})
	err := s.posts.FindOne(ctx,
		bson.M{
			"AuthorId": userId,
			"_id":      bson.M{"$gt": postId},
		}, nextPostOpts).Decode(&nextPost)

	if err != nil {
		return nil, fmt.Errorf("can't find next post - %w", err)
	}

	return &nextPost, nil
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

func encodeObjectIdToBase64(objId primitive.ObjectID) string {
	return base64.URLEncoding.EncodeToString([]byte(objId.Hex()))
}
