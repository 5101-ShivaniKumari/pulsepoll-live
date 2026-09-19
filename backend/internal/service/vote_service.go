package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"github.com/livepoll/backend/internal/models"
	"github.com/livepoll/backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInvalidOptionID = errors.New("selected option does not exist in this poll")
	ErrVoterTokenEmpty = errors.New("voter token cannot be empty")
)

type VoteService struct {
	mongoRepo *repository.MongoRepo
	redisRepo *repository.RedisRepo
}

func NewVoteService(mongoRepo *repository.MongoRepo, redisRepo *repository.RedisRepo) *VoteService {
	return &VoteService{
		mongoRepo: mongoRepo,
		redisRepo: redisRepo,
	}
}

func (s *VoteService) CastVote(ctx context.Context, pollIDStr string, req *models.VoteRequest, clientIP string) (*models.LivePollUpdate, error) {
	voterToken := strings.TrimSpace(req.VoterToken)
	if voterToken == "" || len(voterToken) < 10 {
		return nil, ErrVoterTokenEmpty
	}

	pollID, err := primitive.ObjectIDFromHex(pollIDStr)
	if err != nil {
		return nil, repository.ErrPollNotFound
	}

	// 1. Fetch Poll metadata from MongoDB
	poll, err := s.mongoRepo.GetPollByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	// 2. Check if poll is closed or expired
	if poll.IsClosed {
		return nil, ErrPollClosed
	}
	if poll.ExpiryAt != nil && time.Now().After(*poll.ExpiryAt) {
		return nil, ErrPollExpired
	}

	// 3. Verify option exists in this poll
	var validOption bool
	for _, opt := range poll.Options {
		if opt.ID == req.OptionID {
			validOption = true
			break
		}
	}
	if !validOption {
		return nil, ErrInvalidOptionID
	}

	// 4. Redis Fast-Path Deduplication: Atomic SADD
	isNewVoter, err := s.redisRepo.RegisterVoter(ctx, pollIDStr, voterToken)
	if err != nil {
		log.Printf("Redis RegisterVoter error: %v, falling back to Mongo check", err)
		// Fallback check against Mongo
		_, hasVoted, mErr := s.mongoRepo.GetVotedOption(ctx, pollID, voterToken)
		if mErr == nil && hasVoted {
			return nil, repository.ErrVoteAlreadyExists
		}
	} else if !isNewVoter {
		return nil, repository.ErrVoteAlreadyExists
	}

	// 5. Redis Fast-Path Count Increment: Atomic HINCRBY
	counts, totalVotes, err := s.redisRepo.IncrementVoteCount(ctx, pollIDStr, req.OptionID)
	if err != nil {
		log.Printf("Redis IncrementVoteCount error: %v", err)
		// Fallback counts
		totalVotes = poll.TotalVotes + 1
		counts = make(map[string]int64)
		for _, opt := range poll.Options {
			if opt.ID == req.OptionID {
				counts[opt.ID] = opt.VoteCount + 1
			} else {
				counts[opt.ID] = opt.VoteCount
			}
		}
	}

	// 6. Durable MongoDB Write (Recorded in goroutine to not block high-throughput realtime path)
	ipHash := hashString(clientIP)
	voteRecord := &models.Vote{
		PollID:     pollID,
		OptionID:   req.OptionID,
		VoterToken: voterToken,
		IPHash:     ipHash,
	}

	go func(v *models.Vote) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.mongoRepo.RecordVote(bgCtx, v); err != nil {
			log.Printf("Error recording durable vote to Mongo: %v", err)
		}
	}(voteRecord)

	// 7. Calculate real-time percentages
	percentages := make(map[string]float64)
	if totalVotes > 0 {
		for optID, count := range counts {
			pct := (float64(count) / float64(totalVotes)) * 100.0
			percentages[optID] = math.Round(pct*10) / 10
		}
	}

	// 8. Construct Live Update
	liveUpdate := &models.LivePollUpdate{
		Type:        "vote_cast",
		PollID:      pollIDStr,
		TotalVotes:  totalVotes,
		OptionVotes: counts,
		Percentages: percentages,
		IsClosed:    poll.IsClosed,
		Timestamp:   time.Now().UnixMilli(),
	}

	// 9. Publish update to Redis Pub/Sub channel
	if pubErr := s.redisRepo.PublishPollUpdate(ctx, pollIDStr, liveUpdate); pubErr != nil {
		log.Printf("Failed to publish live update to Redis Pub/Sub: %v", pubErr)
	}

	return liveUpdate, nil
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
