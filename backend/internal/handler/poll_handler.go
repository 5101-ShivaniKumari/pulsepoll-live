package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/livepoll/backend/internal/middleware"
	"github.com/livepoll/backend/internal/models"
	"github.com/livepoll/backend/internal/repository"
	"github.com/livepoll/backend/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PollHandler struct {
	pollService *service.PollService
}

func NewPollHandler(pollService *service.PollService) *PollHandler {
	return &PollHandler{pollService: pollService}
}

func (h *PollHandler) CreatePoll(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.CtxUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required to create a poll"})
		return
	}
	creatorID := userIDVal.(primitive.ObjectID)

	var req models.CreatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	poll, err := h.pollService.CreatePoll(c.Request.Context(), creatorID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, poll)
}

func (h *PollHandler) GetPoll(c *gin.Context) {
	pollID := c.Param("id")
	if pollID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Poll ID is required"})
		return
	}

	// Voter token can be provided in custom header or query param
	voterToken := c.GetHeader("X-Voter-Token")
	if voterToken == "" {
		voterToken = c.Query("voter_token")
	}

	pollDetail, err := h.pollService.GetPoll(c.Request.Context(), pollID, voterToken)
	if err != nil {
		if errors.Is(err, repository.ErrPollNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch poll"})
		return
	}

	c.JSON(http.StatusOK, pollDetail)
}

func (h *PollHandler) GetMyPolls(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.CtxUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	creatorID := userIDVal.(primitive.ObjectID)

	polls, err := h.pollService.GetPollsByCreator(c.Request.Context(), creatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch your polls"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"polls": polls})
}

func (h *PollHandler) ClosePoll(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.CtxUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	creatorID := userIDVal.(primitive.ObjectID)
	pollID := c.Param("id")

	if err := h.pollService.ClosePoll(c.Request.Context(), pollID, creatorID); err != nil {
		if errors.Is(err, repository.ErrPollNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found or not owned by you"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close poll"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Poll closed successfully"})
}

func (h *PollHandler) DeletePoll(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.CtxUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	creatorID := userIDVal.(primitive.ObjectID)
	pollID := c.Param("id")

	if err := h.pollService.DeletePoll(c.Request.Context(), pollID, creatorID); err != nil {
		if errors.Is(err, repository.ErrPollNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found or not owned by you"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete poll"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Poll deleted successfully"})
}
