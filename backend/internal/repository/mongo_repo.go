package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
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
	client     *mongo.Client
	db         *mongo.Database
	users      *mongo.Collection
	polls      *mongo.Collection
	votes      *mongo.Collection
	isInMemory bool

	// In-memory storage structures for zero-config fallback
	mu       sync.RWMutex
	memUsers map[string]*models.User // keyed by ID hex and email
	memPolls map[string]*models.Poll // keyed by ID hex
	memVotes map[string]*models.Vote // keyed by "pollID:voterToken"
}

func NewMongoRepo(ctx context.Context, uri, dbName string) (*MongoRepo, error) {
	clientOpts := options.Client().ApplyURI(uri).SetTimeout(3 * time.Second)
	client, err := mongo.Connect(ctx, clientOpts)
	
	pingErr := error(nil)
	if err == nil {
		pingCtx, pingCancel := context.WithTimeout(ctx, 2*time.Second)
		defer pingCancel()
		pingErr = client.Ping(pingCtx, nil)
	}

	if err != nil || pingErr != nil {
		log.Printf("[INFO] MongoDB at '%s' is not reachable (%v). Using fast In-Memory fallback store.", uri, pingErr)
		return &MongoRepo{
			isInMemory: true,
			memUsers:   make(map[string]*models.User),
			memPolls:   make(map[string]*models.Poll),
			memVotes:   make(map[string]*models.Vote),
		}, nil
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
	if r.client != nil {
		return r.client.Disconnect(ctx)
	}
	return nil
}

func (r *MongoRepo) initIndexes(ctx context.Context) error {
	if r.isInMemory {
		return nil
	}
	_, err := r.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

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

	_, err = r.polls.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "creator_id", Value: 1}},
	})
	return err
}

// User Operations

func (r *MongoRepo) CreateUser(ctx context.Context, user *models.User) error {
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	if r.isInMemory {
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, u := range r.memUsers {
			if u.Email == user.Email {
				return ErrUserAlreadyExists
			}
		}
		user.ID = primitive.NewObjectID()
		copyUser := *user
		r.memUsers[user.ID.Hex()] = &copyUser
		return nil
	}

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
	if r.isInMemory {
		r.mu.RLock()
		defer r.mu.RUnlock()
		for _, u := range r.memUsers {
			if u.Email == email {
				copyUser := *u
				return &copyUser, nil
			}
		}
		return nil, ErrUserNotFound
	}

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
	if r.isInMemory {
		r.mu.RLock()
		defer r.mu.RUnlock()
		if u, exists := r.memUsers[id.Hex()]; exists {
			copyUser := *u
			return &copyUser, nil
		}
		return nil, ErrUserNotFound
	}

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

	if r.isInMemory {
		r.mu.Lock()
		defer r.mu.Unlock()
		poll.ID = primitive.NewObjectID()
		copyPoll := *poll
		// Deep copy options
		copyPoll.Options = make([]models.Option, len(poll.Options))
		copy(copyPoll.Options, poll.Options)
		r.memPolls[poll.ID.Hex()] = &copyPoll
		return nil
	}

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
	if r.isInMemory {
		r.mu.RLock()
		defer r.mu.RUnlock()
		p, exists := r.memPolls[id.Hex()]
		if !exists {
			return nil, ErrPollNotFound
		}
		copyPoll := *p
		copyPoll.Options = make([]models.Option, len(p.Options))
		copy(copyPoll.Options, p.Options)
		return &copyPoll, nil
	}

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
	if r.isInMemory {
		r.mu.RLock()
		defer r.mu.RUnlock()
		var polls []models.Poll
		for _, p := range r.memPolls {
			if p.CreatorID == creatorID {
				copyPoll := *p
				copyPoll.Options = make([]models.Option, len(p.Options))
				copy(copyPoll.Options, p.Options)
				polls = append(polls, copyPoll)
			}
		}
		return polls, nil
	}

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
	if r.isInMemory {
		r.mu.Lock()
		defer r.mu.Unlock()
		p, exists := r.memPolls[pollID.Hex()]
		if !exists || (creatorID != primitive.NilObjectID && p.CreatorID != creatorID) {
			return ErrPollNotFound
		}
		p.IsClosed = true
		p.UpdatedAt = time.Now().UTC()
		return nil
	}

	filter := bson.M{
		"_id": pollID,
	}
	if creatorID != primitive.NilObjectID {
		filter["creator_id"] = creatorID
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

// FindAndCloseExpiredPolls finds active polls whose expiry time has passed, marks them closed, and returns them.
func (r *MongoRepo) FindAndCloseExpiredPolls(ctx context.Context) ([]models.Poll, error) {
	now := time.Now().UTC()
	var expiredPolls []models.Poll

	if r.isInMemory {
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, p := range r.memPolls {
			if !p.IsClosed && p.ExpiryAt != nil && now.After(*p.ExpiryAt) {
				p.IsClosed = true
				p.UpdatedAt = now
				copyPoll := *p
				expiredPolls = append(expiredPolls, copyPoll)
			}
		}
		return expiredPolls, nil
	}

	filter := bson.M{
		"is_closed": false,
		"expiry_at": bson.M{"$lte": now, "$ne": nil},
	}

	cursor, err := r.polls.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &expiredPolls); err != nil {
		return nil, err
	}

	if len(expiredPolls) > 0 {
		var ids []primitive.ObjectID
		for _, p := range expiredPolls {
			ids = append(ids, p.ID)
		}
		_, _ = r.polls.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": ids}}, bson.M{
			"$set": bson.M{
				"is_closed":  true,
				"updated_at": now,
			},
		})
	}

	return expiredPolls, nil
}

func (r *MongoRepo) DeletePoll(ctx context.Context, pollID primitive.ObjectID, creatorID primitive.ObjectID) error {
	if r.isInMemory {
		r.mu.Lock()
		defer r.mu.Unlock()
		p, exists := r.memPolls[pollID.Hex()]
		if !exists || p.CreatorID != creatorID {
			return ErrPollNotFound
		}
		delete(r.memPolls, pollID.Hex())
		return nil
	}

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
	_, _ = r.votes.DeleteMany(ctx, bson.M{"poll_id": pollID})
	return nil
}

// Vote Operations

func (r *MongoRepo) RecordVote(ctx context.Context, vote *models.Vote) error {
	vote.CreatedAt = time.Now().UTC()

	if r.isInMemory {
		r.mu.Lock()
		defer r.mu.Unlock()

		voteKey := fmt.Sprintf("%s:%s", vote.PollID.Hex(), vote.VoterToken)
		if _, exists := r.memVotes[voteKey]; exists {
			return ErrVoteAlreadyExists
		}

		vote.ID = primitive.NewObjectID()
		copyVote := *vote
		r.memVotes[voteKey] = &copyVote

		// Increment poll totals in memory
		if p, exists := r.memPolls[vote.PollID.Hex()]; exists {
			p.TotalVotes++
			for i := range p.Options {
				if p.Options[i].ID == vote.OptionID {
					p.Options[i].VoteCount++
					break
				}
			}
			p.UpdatedAt = time.Now().UTC()
		}
		return nil
	}

	_, err := r.votes.InsertOne(ctx, vote)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrVoteAlreadyExists
		}
		return err
	}

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
	if r.isInMemory {
		r.mu.RLock()
		defer r.mu.RUnlock()
		voteKey := fmt.Sprintf("%s:%s", pollID.Hex(), voterToken)
		if v, exists := r.memVotes[voteKey]; exists {
			return v.OptionID, true, nil
		}
		return "", false, nil
	}

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
