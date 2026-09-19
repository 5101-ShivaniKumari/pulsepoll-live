package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/livepoll/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrUserAlreadyExists = errors.New("a user with this email already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrPollNotFound      = errors.New("poll not found")
	ErrVoteAlreadyExists = errors.New("voter has already voted on this poll")
)

type MongoRepo struct {
	client *mongo.Client
	db     *mongo.Database
	users  *mongo.Collection
	polls  *mongo.Collection
	votes  *mongo.Collection
}

func NewMongoRepo(ctx context.Context, uri, dbName string) (*MongoRepo, error) {
	clientOpts := options.Client().ApplyURI(uri).SetTimeout(10 * time.Second)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	// Ping Mongo to ensure connection is alive
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping failed: %w", err)
	}

	db := client.Database(dbName)
	repo := &MongoRepo{
		client: client,
		db:     db,
		users:  db.Collection("users"),
		polls:  db.Collection("polls"),
		votes:  db.Collection("votes"),
	}

	// Ensure Indexes
	if err := repo.initIndexes(ctx); err != nil {
		log.Printf("Warning: failed to create mongo indexes: %v", err)
	}

	return repo, nil
}

func (r *MongoRepo) Close(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}

func (r *MongoRepo) initIndexes(ctx context.Context) error {
	// User: email unique index
	_, err := r.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// Vote: unique compound index on (poll_id, voter_token)
	_, err = r.votes.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "poll_id", Value: 1},
			{Key: "voter_token", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// Poll: index on creator_id
	_, err = r.polls.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "creator_id", Value: 1}},
	})
	if err != nil {
		return err
	}

	return nil
}

// User Operations

func (r *MongoRepo) CreateUser(ctx context.Context, user *models.User) error {
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	res, err := r.users.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrUserAlreadyExists
		}
		return err
	}

	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		user.ID = oid
	}
	return nil
}

func (r *MongoRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *MongoRepo) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.users.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Poll Operations

func (r *MongoRepo) CreatePoll(ctx context.Context, poll *models.Poll) error {
	poll.CreatedAt = time.Now().UTC()
	poll.UpdatedAt = time.Now().UTC()
	poll.TotalVotes = 0
	poll.IsClosed = false

	res, err := r.polls.InsertOne(ctx, poll)
	if err != nil {
		return err
	}

	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		poll.ID = oid
	}
	return nil
}

func (r *MongoRepo) GetPollByID(ctx context.Context, id primitive.ObjectID) (*models.Poll, error) {
	var poll models.Poll
	err := r.polls.FindOne(ctx, bson.M{"_id": id}).Decode(&poll)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrPollNotFound
		}
		return nil, err
	}
	return &poll, nil
}

func (r *MongoRepo) GetPollsByCreatorID(ctx context.Context, creatorID primitive.ObjectID) ([]models.Poll, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.polls.Find(ctx, bson.M{"creator_id": creatorID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		return nil, err
	}
	if polls == nil {
		polls = []models.Poll{}
	}
	return polls, nil
}

func (r *MongoRepo) ClosePoll(ctx context.Context, pollID primitive.ObjectID, creatorID primitive.ObjectID) error {
	filter := bson.M{
		"_id":        pollID,
		"creator_id": creatorID,
	}
	update := bson.M{
		"$set": bson.M{
			"is_closed":  true,
			"updated_at": time.Now().UTC(),
		},
	}

	res, err := r.polls.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrPollNotFound
	}
	return nil
}

func (r *MongoRepo) DeletePoll(ctx context.Context, pollID primitive.ObjectID, creatorID primitive.ObjectID) error {
	filter := bson.M{
		"_id":        pollID,
		"creator_id": creatorID,
	}
	res, err := r.polls.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrPollNotFound
	}
	// Also delete votes for this poll
	_, _ = r.votes.DeleteMany(ctx, bson.M{"poll_id": pollID})
	return nil
}

// Vote Operations

func (r *MongoRepo) RecordVote(ctx context.Context, vote *models.Vote) error {
	vote.CreatedAt = time.Now().UTC()
	_, err := r.votes.InsertOne(ctx, vote)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrVoteAlreadyExists
		}
		return err
	}

	// Atomically increment the option vote count and total votes in MongoDB poll document
	filter := bson.M{
		"_id":        vote.PollID,
		"options.id": vote.OptionID,
	}
	update := bson.M{
		"$inc": bson.M{
			"total_votes":       1,
			"options.$.vote_count": 1,
		},
		"$set": bson.M{
			"updated_at": time.Now().UTC(),
		},
	}

	_, err = r.polls.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoRepo) GetVotedOption(ctx context.Context, pollID primitive.ObjectID, voterToken string) (string, bool, error) {
	var vote models.Vote
	err := r.votes.FindOne(ctx, bson.M{
		"poll_id":     pollID,
		"voter_token": voterToken,
	}).Decode(&vote)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", false, nil
		}
		return "", false, err
	}
	return vote.OptionID, true, nil
}
