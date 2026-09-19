package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/livepoll/backend/internal/models"
	"github.com/livepoll/backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInvalidPollData     = errors.New("invalid poll data")
	ErrDuplicateOptions    = errors.New("poll options must be distinct")
	ErrInsufficientOptions = errors.New("at least 2 distinct options are required")
	ErrTooManyOptions      = errors.New("maximum 10 options allowed")
	ErrPollExpired         = errors.New("poll has expired")
	ErrPollClosed          = errors.New("poll is closed for voting")
	ErrUnauthorized        = errors.New("unauthorized action on poll")
)

type PollService struct {
	mongoRepo *repository.MongoRepo
	redisRepo *repository.RedisRepo
}

func NewPollService(mongoRepo *repository.MongoRepo, redisRepo *repository.RedisRepo) *PollService {
	return &PollService{
		mongoRepo: mongoRepo,
		redisRepo: redisRepo,
	}
}

func (s *PollService) CreatePoll(ctx context.Context, creatorID primitive.ObjectID, req *models.CreatePollRequest) (*models.Poll, error) {
	title := strings.TrimSpace(req.Title)
	if len(title) < 3 || len(title) > 300 {
		return nil, fmt.Errorf("%w: title must be between 3 and 300 characters", ErrInvalidPollData)
	}

	if len(req.Options) < 2 {
		return nil, ErrInsufficientOptions
	}
	if len(req.Options) > 10 {
		return nil, ErrTooManyOptions
	}

	seenOptions := make(map[string]bool)
	var options []models.Option

	for i, rawOpt := range req.Options {
		optText := strings.TrimSpace(rawOpt)
		if optText == "" {
			continue
		}
		if len(optText) > 150 {
			return nil, fmt.Errorf("%w: option '%s' exceeds 150 characters", ErrInvalidPollData, optText)
		}

		normalized := strings.ToLower(optText)
		if seenOptions[normalized] {
			return nil, fmt.Errorf("%w: '%s' is duplicated", ErrDuplicateOptions, optText)
		}
		seenOptions[normalized] = true

		options = append(options, models.Option{
			ID:        fmt.Sprintf("opt_%s", uuid.New().String()[:8]),
			Text:      optText,
			VoteCount: 0,
			Order:     i + 1,
		})
	}

	if len(options) < 2 {
		return nil, ErrInsufficientOptions
	}

	// Validate expiry time if supplied
	if req.ExpiryAt != nil && !req.ExpiryAt.After(time.Now()) {
		return nil, fmt.Errorf("%w: expiry date must be in the future", ErrInvalidPollData)
	}

	poll := &models.Poll{
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		Options:     options,
		CreatorID:   creatorID,
		IsClosed:    false,
		ExpiryAt:    req.ExpiryAt,
		TotalVotes:  0,
	}

	if err := s.mongoRepo.CreatePoll(ctx, poll); err != nil {
		return nil, fmt.Errorf("failed to create poll in mongo: %w", err)
	}

	// Initialize live counts cache in Redis
	if err := s.redisRepo.InitPollCounts(ctx, poll.ID.Hex(), poll.Options, 0); err != nil {
		// Log warning, but Mongo is durable source of truth
		fmt.Printf("Warning: failed to initialize redis counts: %v\n", err)
	}

	return poll, nil
}

func (s *PollService) GetPoll(ctx context.Context, pollIDStr string, voterToken string) (*models.PollDetailResponse, error) {
	pollID, err := primitive.ObjectIDFromHex(pollIDStr)
	if err != nil {
		return nil, repository.ErrPollNotFound
	}

	poll, err := s.mongoRepo.GetPollByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	// Check if poll is expired
	if poll.ExpiryAt != nil && time.Now().After(*poll.ExpiryAt) && !poll.IsClosed {
		poll.IsClosed = true
		// Asynchronously mark as closed in Mongo
		go func(pid primitive.ObjectID, cid primitive.ObjectID) {
			_ = s.mongoRepo.ClosePoll(context.Background(), pid, cid)
		}(poll.ID, poll.CreatorID)
	}

	// Fetch live counts from Redis if available
	liveCounts, totalVotes, exists, err := s.redisRepo.GetPollCounts(ctx, pollIDStr)
	if err == nil && exists {
		poll.TotalVotes = totalVotes
		for i := range poll.Options {
			if count, ok := liveCounts[poll.Options[i].ID]; ok {
				poll.Options[i].VoteCount = count
			}
		}
	} else {
		// Warm Redis cache from Mongo data
		_ = s.redisRepo.InitPollCounts(ctx, pollIDStr, poll.Options, poll.TotalVotes)
	}

	// Compute percentages
	percentages := s.CalculatePercentages(poll.Options, poll.TotalVotes)

	// Check if current voter has voted
	hasVoted := false
	var votedFor string

	if voterToken != "" {
		if hasVotedInRedis, err := s.redisRepo.HasVoterVoted(ctx, pollIDStr, voterToken); err == nil && hasVotedInRedis {
			hasVoted = true
		} else {
			// Fallback check in Mongo
			if optID, voted, err := s.mongoRepo.GetVotedOption(ctx, pollID, voterToken); err == nil && voted {
				hasVoted = true
				votedFor = optID
			}
		}

		if hasVoted && votedFor == "" {
			if optID, voted, err := s.mongoRepo.GetVotedOption(ctx, pollID, voterToken); err == nil && voted {
				votedFor = optID
			}
		}
	}

	return &models.PollDetailResponse{
		Poll:        *poll,
		HasVoted:    hasVoted,
		VotedFor:    votedFor,
		Percentages: percentages,
	}, nil
}

func (s *PollService) GetPollsByCreator(ctx context.Context, creatorID primitive.ObjectID) ([]models.Poll, error) {
	polls, err := s.mongoRepo.GetPollsByCreatorID(ctx, creatorID)
	if err != nil {
		return nil, err
	}

	// Enrich with live counts from Redis
	for i := range polls {
		pidStr := polls[i].ID.Hex()
		liveCounts, totalVotes, exists, err := s.redisRepo.GetPollCounts(ctx, pidStr)
		if err == nil && exists {
			polls[i].TotalVotes = totalVotes
			for j := range polls[i].Options {
				if count, ok := liveCounts[polls[i].Options[j].ID]; ok {
					polls[i].Options[j].VoteCount = count
				}
			}
		}
	}

	return polls, nil
}

func (s *PollService) ClosePoll(ctx context.Context, pollIDStr string, creatorID primitive.ObjectID) error {
	pollID, err := primitive.ObjectIDFromHex(pollIDStr)
	if err != nil {
		return repository.ErrPollNotFound
	}

	if err := s.mongoRepo.ClosePoll(ctx, pollID, creatorID); err != nil {
		return err
	}

	// Broadcast poll_closed event via Redis Pub/Sub
	detail, err := s.GetPoll(ctx, pollIDStr, "")
	if err == nil {
		optVotes := make(map[string]int64)
		for _, opt := range detail.Options {
			optVotes[opt.ID] = opt.VoteCount
		}

		update := &models.LivePollUpdate{
			Type:        "poll_closed",
			PollID:      pollIDStr,
			TotalVotes:  detail.TotalVotes,
			OptionVotes: optVotes,
			Percentages: detail.Percentages,
			IsClosed:    true,
			Timestamp:   time.Now().UnixMilli(),
		}
		_ = s.redisRepo.PublishPollUpdate(ctx, pollIDStr, update)
	}

	return nil
}

func (s *PollService) DeletePoll(ctx context.Context, pollIDStr string, creatorID primitive.ObjectID) error {
	pollID, err := primitive.ObjectIDFromHex(pollIDStr)
	if err != nil {
		return repository.ErrPollNotFound
	}

	return s.mongoRepo.DeletePoll(ctx, pollID, creatorID)
}

func (s *PollService) CalculatePercentages(options []models.Option, totalVotes int64) map[string]float64 {
	percentages := make(map[string]float64)
	if totalVotes <= 0 {
		for _, opt := range options {
			percentages[opt.ID] = 0.0
		}
		return percentages
	}

	for _, opt := range options {
		pct := (float64(opt.VoteCount) / float64(totalVotes)) * 100.0
		// Round to 1 decimal place
		percentages[opt.ID] = math.Round(pct*10) / 10
	}
	return percentages
}

// StartAutoCloseScheduler runs a background worker checking for expired polls and broadcasting closures in real time
func (s *PollService) StartAutoCloseScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				expiredPolls, err := s.mongoRepo.FindAndCloseExpiredPolls(ctx)
				if err != nil || len(expiredPolls) == 0 {
					continue
				}
				for _, p := range expiredPolls {
					pidStr := p.ID.Hex()
					detail, err := s.GetPoll(ctx, pidStr, "")
					if err == nil {
						optVotes := make(map[string]int64)
						for _, opt := range detail.Options {
							optVotes[opt.ID] = opt.VoteCount
						}

						viewerCount, _ := s.redisRepo.GetViewerCount(ctx, pidStr)
						update := &models.LivePollUpdate{
							Type:        "poll_closed",
							PollID:      pidStr,
							TotalVotes:  detail.TotalVotes,
							OptionVotes: optVotes,
							Percentages: detail.Percentages,
							IsClosed:    true,
							ViewerCount: viewerCount,
							Timestamp:   time.Now().UnixMilli(),
						}
						_ = s.redisRepo.PublishPollUpdate(ctx, pidStr, update)
					}
				}
			}
		}
	}()
}
