package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/livepoll/backend/internal/models"
	"github.com/livepoll/backend/internal/service"
	ws "github.com/livepoll/backend/internal/websocket"
)

type WSHandler struct {
	hub         *ws.Hub
	pollService *service.PollService
}

func NewWSHandler(hub *ws.Hub, pollService *service.PollService) *WSHandler {
	return &WSHandler{
		hub:         hub,
		pollService: pollService,
	}
}

func (h *WSHandler) HandleWS(c *gin.Context) {
	pollID := c.Param("id")
	if pollID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Poll ID is required"})
		return
	}

	// Fetch current state to seed initial frame
	var initialState *models.LivePollUpdate
	detail, err := h.pollService.GetPoll(c.Request.Context(), pollID, "")
	if err == nil {
		optVotes := make(map[string]int64)
		for _, opt := range detail.Options {
			optVotes[opt.ID] = opt.VoteCount
		}

		initialState = &models.LivePollUpdate{
			Type:        "poll_state",
			PollID:      pollID,
			TotalVotes:  detail.TotalVotes,
			OptionVotes: optVotes,
			Percentages: detail.Percentages,
			IsClosed:    detail.IsClosed,
			Timestamp:   time.Now().UnixMilli(),
		}
	} else {
		log.Printf("[WS] Could not get initial state for poll %s: %v", pollID, err)
	}

	err = ws.ServeWS(h.hub, c.Writer, c.Request, pollID, initialState)
	if err != nil {
		log.Printf("[WS] ServeWS failed: %v", err)
	}
}
