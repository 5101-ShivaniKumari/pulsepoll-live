package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/livepoll/backend/internal/models"
	"github.com/livepoll/backend/internal/repository"
	"github.com/livepoll/backend/internal/service"
)

type VoteHandler struct {
	voteService *service.VoteService
}

func NewVoteHandler(voteService *service.VoteService) *VoteHandler {
	return &VoteHandler{voteService: voteService}
}

func (h *VoteHandler) Vote(c *gin.Context) {
	pollID := c.Param("id")
	if pollID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Poll ID is required"})
		return
	}

	var req models.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vote request: " + err.Error()})
		return
	}

	// Read voter token from header if not in payload
	if req.VoterToken == "" {
		req.VoterToken = c.GetHeader("X-Voter-Token")
	}

	clientIP := c.ClientIP()
	liveUpdate, err := h.voteService.CastVote(c.Request.Context(), pollID, &req, clientIP)
	if err != nil {
		if errors.Is(err, repository.ErrVoteAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "You have already voted on this poll."})
			return
		}
		if errors.Is(err, repository.ErrPollNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
			return
		}
		if errors.Is(err, service.ErrPollClosed) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "This poll is closed for voting."})
			return
		}
		if errors.Is(err, service.ErrPollExpired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "This poll has expired."})
			return
		}
		if errors.Is(err, service.ErrInvalidOptionID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Selected option is invalid."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process vote: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Vote registered successfully",
		"live_update": liveUpdate,
	})
}
