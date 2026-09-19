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

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process login request"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.CtxUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
		return
	}

	userInfo, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, userInfo)
}
