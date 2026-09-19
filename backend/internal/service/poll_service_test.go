package service_test

import (
	"context"
	"testing"

	"github.com/livepoll/backend/internal/models"
	"github.com/livepoll/backend/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCalculatePercentages(t *testing.T) {
	ps := service.NewPollService(nil, nil)

	options := []models.Option{
		{ID: "opt_1", Text: "Go", VoteCount: 3},
		{ID: "opt_2", Text: "Rust", VoteCount: 1},
	}

	percentages := ps.CalculatePercentages(options, 4)

	if percentages["opt_1"] != 75.0 {
		t.Errorf("Expected opt_1 to have 75.0%%, got %.1f%%", percentages["opt_1"])
	}
	if percentages["opt_2"] != 25.0 {
		t.Errorf("Expected opt_2 to have 25.0%%, got %.1f%%", percentages["opt_2"])
	}
}

func TestCalculatePercentagesZeroVotes(t *testing.T) {
	ps := service.NewPollService(nil, nil)

	options := []models.Option{
		{ID: "opt_1", Text: "Go", VoteCount: 0},
		{ID: "opt_2", Text: "Rust", VoteCount: 0},
	}

	percentages := ps.CalculatePercentages(options, 0)

	if percentages["opt_1"] != 0.0 || percentages["opt_2"] != 0.0 {
		t.Errorf("Expected all 0.0%% when totalVotes is 0, got %v", percentages)
	}
}

func TestCreatePollValidation(t *testing.T) {
	ps := service.NewPollService(nil, nil)
	ctx := context.Background()
	dummyCreatorID := primitive.NewObjectID()

	// Test 1: Title too short
	_, err := ps.CreatePoll(ctx, dummyCreatorID, &models.CreatePollRequest{
		Title:   "hi",
		Options: []string{"A", "B"},
	})
	if err == nil {
		t.Error("Expected error for title under 3 characters, got nil")
	}

	// Test 2: Less than 2 options
	_, err = ps.CreatePoll(ctx, dummyCreatorID, &models.CreatePollRequest{
		Title:   "Valid Question Title",
		Options: []string{"Option 1"},
	})
	if err == nil {
		t.Error("Expected error for single option poll, got nil")
	}

	// Test 3: Duplicate options
	_, err = ps.CreatePoll(ctx, dummyCreatorID, &models.CreatePollRequest{
		Title:   "Valid Question Title",
		Options: []string{"Same Option", "same option"},
	})
	if err == nil {
		t.Error("Expected error for duplicate options, got nil")
	}
}
