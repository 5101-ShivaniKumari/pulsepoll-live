package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/livepoll/backend/internal/models"
	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	client     *redis.Client
	isInMemory bool

	// In-memory fallback structures
	mu             sync.RWMutex
	memCounts      map[string]map[string]int64 // pollID -> (optionID -> count)
	memVoters      map[string]map[string]bool  // pollID -> (voterToken -> bool)
	memSubscribers map[string][]chan []byte    // pollID -> list of active event channels
}

func NewRedisRepo(redisURL string) (*RedisRepo, error) {
	opts, err := redis.ParseURL(redisURL)
	var client *redis.Client
	pingErr := error(nil)

	if err == nil {
		client = redis.NewClient(opts)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		pingErr = client.Ping(ctx).Err()
	}

	if err != nil || pingErr != nil {
		log.Printf("[INFO] Redis at '%s' is not reachable (%v). Using fast In-Memory Pub/Sub & Caching Engine.", redisURL, pingErr)
		return &RedisRepo{
			isInMemory:     true,
			memCounts:      make(map[string]map[string]int64),
			memVoters:      make(map[string]map[string]bool),
			memSubscribers: make(map[string][]chan []byte),
		}, nil
	}

	return &RedisRepo{client: client}, nil
}

func (r *RedisRepo) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *RedisRepo) Client() *redis.Client {
	return r.client
}

func (r *RedisRepo) IsInMemory() bool {
	return r.isInMemory
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
func (r *RedisRepo) RegisterVoter(ctx context.Context, pollID, voterToken string) (bool, error) {
	if r.isInMemory {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.memVoters[pollID] == nil {
			r.memVoters[pollID] = make(map[string]bool)
		}
		if r.memVoters[pollID][voterToken] {
			return false, nil // Already voted
		}
		r.memVoters[pollID][voterToken] = true
		return true, nil
	}

	key := pollVotersKey(pollID)
	added, err := r.client.SAdd(ctx, key, voterToken).Result()
	if err != nil {
		return false, fmt.Errorf("redis SAdd failed: %w", err)
	}
	return added > 0, nil
}

// HasVoterVoted checks if the voter token exists in the poll's voter set.
func (r *RedisRepo) HasVoterVoted(ctx context.Context, pollID, voterToken string) (bool, error) {
	if r.isInMemory {
		r.mu.RLock()
		defer r.mu.RUnlock()
		if r.memVoters[pollID] == nil {
			return false, nil
		}
		return r.memVoters[pollID][voterToken], nil
	}

	key := pollVotersKey(pollID)
	isMember, err := r.client.SIsMember(ctx, key, voterToken).Result()
	if err != nil {
		return false, fmt.Errorf("redis SIsMember failed: %w", err)
	}
	return isMember, nil
}

// InitPollCounts sets up the initial option vote counts in Redis if not already present.
func (r *RedisRepo) InitPollCounts(ctx context.Context, pollID string, options []models.Option, totalVotes int64) error {
	if r.isInMemory {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.memCounts[pollID] == nil {
			r.memCounts[pollID] = make(map[string]int64)
			for _, opt := range options {
				r.memCounts[pollID][opt.ID] = opt.VoteCount
			}
			r.memCounts[pollID]["__total__"] = totalVotes
		}
		return nil
	}

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
	pipeline.Expire(ctx, key, 30*24*time.Hour)
	_, err = pipeline.Exec(ctx)
	return err
}

// IncrementVoteCount atomically increments the option vote count and total count in Redis.
func (r *RedisRepo) IncrementVoteCount(ctx context.Context, pollID, optionID string) (map[string]int64, int64, error) {
	if r.isInMemory {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.memCounts[pollID] == nil {
			r.memCounts[pollID] = make(map[string]int64)
		}
		r.memCounts[pollID][optionID]++
		r.memCounts[pollID]["__total__"]++

		res := make(map[string]int64)
		var total int64
		for k, v := range r.memCounts[pollID] {
			if k == "__total__" {
				total = v
			} else {
				res[k] = v
			}
		}
		return res, total, nil
	}

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
	if r.isInMemory {
		r.mu.RLock()
		defer r.mu.RUnlock()
		m, exists := r.memCounts[pollID]
		if !exists {
			return nil, 0, false, nil
		}
		counts := make(map[string]int64)
		var total int64
		for k, v := range m {
			if k == "__total__" {
				total = v
			} else {
				counts[k] = v
			}
		}
		return counts, total, true, nil
	}

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

// PublishPollUpdate broadcasts a live update payload.
func (r *RedisRepo) PublishPollUpdate(ctx context.Context, pollID string, update *models.LivePollUpdate) error {
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal live poll update: %w", err)
	}

	if r.isInMemory {
		r.mu.RLock()
		subs := r.memSubscribers[pollID]
		r.mu.RUnlock()
		for _, ch := range subs {
			select {
			case ch <- data:
			default:
			}
		}
		return nil
	}

	channel := pollEventChannel(pollID)
	return r.client.Publish(ctx, channel, data).Err()
}

// SubscribeToPoll subscribes to updates for a specific poll.
func (r *RedisRepo) SubscribeToPoll(ctx context.Context, pollID string) *redis.PubSub {
	if r.isInMemory {
		return nil
	}
	channel := pollEventChannel(pollID)
	return r.client.Subscribe(ctx, channel)
}

// SubscribeInMemory registers an in-memory channel for poll events.
func (r *RedisRepo) SubscribeInMemory(pollID string, ch chan []byte) func() {
	r.mu.Lock()
	r.memSubscribers[pollID] = append(r.memSubscribers[pollID], ch)
	r.mu.Unlock()

	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		subs := r.memSubscribers[pollID]
		for i, c := range subs {
			if c == ch {
				r.memSubscribers[pollID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
}
