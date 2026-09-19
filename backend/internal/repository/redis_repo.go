package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/livepoll/backend/internal/models"
	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	client *redis.Client
}

func NewRedisRepo(redisURL string) (*RedisRepo, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisRepo{client: client}, nil
}

func (r *RedisRepo) Close() error {
	return r.client.Close()
}

func (r *RedisRepo) Client() *redis.Client {
	return r.client
}

// Key helpers
func pollCountsKey(pollID string) string {
	return fmt.Sprintf("poll:%s:counts", pollID)
}

func pollVotersKey(pollID string) string {
	return fmt.Sprintf("poll:%s:voters", pollID)
}

func pollEventChannel(pollID string) string {
	return fmt.Sprintf("poll:events:%s", pollID)
}

// RegisterVoter checks and atomically adds the voter token to the poll's voter set.
// Returns true if voter was added (first vote), false if already voted.
func (r *RedisRepo) RegisterVoter(ctx context.Context, pollID, voterToken string) (bool, error) {
	key := pollVotersKey(pollID)
	// SAdd returns number of elements added (1 if new, 0 if already existed)
	added, err := r.client.SAdd(ctx, key, voterToken).Result()
	if err != nil {
		return false, fmt.Errorf("redis SAdd failed: %w", err)
	}
	return added > 0, nil
}

// HasVoterVoted checks if the voter token exists in the poll's voter set.
func (r *RedisRepo) HasVoterVoted(ctx context.Context, pollID, voterToken string) (bool, error) {
	key := pollVotersKey(pollID)
	isMember, err := r.client.SIsMember(ctx, key, voterToken).Result()
	if err != nil {
		return false, fmt.Errorf("redis SIsMember failed: %w", err)
	}
	return isMember, nil
}

// InitPollCounts sets up the initial option vote counts in Redis if not already present.
func (r *RedisRepo) InitPollCounts(ctx context.Context, pollID string, options []models.Option, totalVotes int64) error {
	key := pollCountsKey(pollID)
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}

	pipeline := r.client.Pipeline()
	values := make(map[string]interface{})
	for _, opt := range options {
		values[opt.ID] = opt.VoteCount
	}
	values["__total__"] = totalVotes

	pipeline.HSet(ctx, key, values)
	// Set 30 days expiration on active poll counts in Redis
	pipeline.Expire(ctx, key, 30*24*time.Hour)
	_, err = pipeline.Exec(ctx)
	return err
}

// IncrementVoteCount atomically increments the option vote count and total count in Redis.
func (r *RedisRepo) IncrementVoteCount(ctx context.Context, pollID, optionID string) (map[string]int64, int64, error) {
	key := pollCountsKey(pollID)

	pipeline := r.client.TxPipeline()
	pipeline.HIncrBy(ctx, key, optionID, 1)
	pipeline.HIncrBy(ctx, key, "__total__", 1)
	pipeline.HGetAll(ctx, key)

	cmders, err := pipeline.Exec(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("redis increment pipeline failed: %w", err)
	}

	hgetAllCmd, ok := cmders[2].(*redis.MapStringStringCmd)
	if !ok {
		return nil, 0, fmt.Errorf("invalid type assertion on HGetAll command")
	}

	resultMap, err := hgetAllCmd.Result()
	if err != nil {
		return nil, 0, err
	}

	counts := make(map[string]int64)
	var total int64
	for k, v := range resultMap {
		val, _ := strconv.ParseInt(v, 10, 64)
		if k == "__total__" {
			total = val
		} else {
			counts[k] = val
		}
	}

	return counts, total, nil
}

// GetPollCounts retrieves current live counts from Redis.
func (r *RedisRepo) GetPollCounts(ctx context.Context, pollID string) (map[string]int64, int64, bool, error) {
	key := pollCountsKey(pollID)
	resultMap, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, 0, false, err
	}
	if len(resultMap) == 0 {
		return nil, 0, false, nil
	}

	counts := make(map[string]int64)
	var total int64
	for k, v := range resultMap {
		val, _ := strconv.ParseInt(v, 10, 64)
		if k == "__total__" {
			total = val
		} else {
			counts[k] = val
		}
	}

	return counts, total, true, nil
}

// PublishPollUpdate broadcasts a live update payload to the poll's pub/sub channel.
func (r *RedisRepo) PublishPollUpdate(ctx context.Context, pollID string, update *models.LivePollUpdate) error {
	channel := pollEventChannel(pollID)
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal live poll update: %w", err)
	}

	return r.client.Publish(ctx, channel, data).Err()
}

// SubscribeToPoll subscribes to updates for a specific poll.
func (r *RedisRepo) SubscribeToPoll(ctx context.Context, pollID string) *redis.PubSub {
	channel := pollEventChannel(pollID)
	return r.client.Subscribe(ctx, channel)
}
