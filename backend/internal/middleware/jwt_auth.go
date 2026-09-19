package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/livepoll/backend/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CtxUserIDKey   = "userID"
	CtxUserEmailKey = "userEmail"
	CtxUserNameKey  = "userName"
)

func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid Authorization header format. Expected 'Bearer <token>'",
			})
			return
		}

		tokenString := parts[1]
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}

		objID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid user ID encoded in token",
			})
			return
		}

		c.Set(CtxUserIDKey, objID)
		c.Set(CtxUserEmailKey, claims.Email)
		c.Set(CtxUserNameKey, claims.Name)
		c.Next()
	}
}

func OptionalAuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				if claims, err := authService.ValidateToken(parts[1]); err == nil {
					if objID, err := primitive.ObjectIDFromHex(claims.UserID); err == nil {
						c.Set(CtxUserIDKey, objID)
						c.Set(CtxUserEmailKey, claims.Email)
						c.Set(CtxUserNameKey, claims.Name)
					}
				}
			}
		}
		c.Next()
	}
}
