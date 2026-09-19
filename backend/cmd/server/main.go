package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/livepoll/backend/internal/config"
	"github.com/livepoll/backend/internal/handler"
	"github.com/livepoll/backend/internal/middleware"
	"github.com/livepoll/backend/internal/repository"
	"github.com/livepoll/backend/internal/service"
	ws "github.com/livepoll/backend/internal/websocket"
)

func main() {
	cfg := config.LoadConfig()

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	log.Printf("Starting Live Polling Service on port :%s [Env: %s]", cfg.Port, cfg.Environment)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mongoRepo, err := repository.NewMongoRepo(ctx, cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to MongoDB at %s: %v", cfg.MongoURI, err)
	}
	defer func() {
		discCtx, discCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer discCancel()
		_ = mongoRepo.Close(discCtx)
	}()
	log.Println("Connected to MongoDB successfully")

	redisRepo, err := repository.NewRedisRepo(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to Redis at %s: %v", cfg.RedisURL, err)
	}
	defer func() {
		_ = redisRepo.Close()
	}()
	log.Println("Connected to Redis successfully")

	authService := service.NewAuthService(mongoRepo, cfg.JWTSecret)
	pollService := service.NewPollService(mongoRepo, redisRepo)
	voteService := service.NewVoteService(mongoRepo, redisRepo)

	hub := ws.NewHub(redisRepo)
	go hub.Run()
	log.Println("WebSocket Hub initialized and running")

	authHandler := handler.NewAuthHandler(authService)
	pollHandler := handler.NewPollHandler(pollService)
	voteHandler := handler.NewVoteHandler(voteService)
	wsHandler := handler.NewWSHandler(hub, pollService)

	router := gin.Default()

	router.Use(middleware.CORSMiddleware(cfg.CORSOrigin))
	router.Use(gin.Recovery())

	generalRateLimiter := middleware.NewRateLimiter(60, time.Minute)
	voteRateLimiter := middleware.NewRateLimiter(30, time.Minute)

	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"service":   "livepoll-backend",
			"version":   "1.0.0",
		})
	}
	router.GET("/health", healthHandler)
	router.GET("/api/v1/health", healthHandler)

	v1 := router.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", generalRateLimiter.Middleware(), authHandler.Register)
			authGroup.POST("/login", generalRateLimiter.Middleware(), authHandler.Login)
			authGroup.GET("/me", middleware.AuthMiddleware(authService), authHandler.Me)
		}

		pollGroup := v1.Group("/polls")
		{
			pollGroup.GET("/:id", pollHandler.GetPoll)
			pollGroup.POST("/:id/vote", voteRateLimiter.Middleware(), voteHandler.Vote)
			pollGroup.GET("/:id/ws", wsHandler.HandleWS)

			protected := pollGroup.Group("")
			protected.Use(middleware.AuthMiddleware(authService))
			{
				protected.POST("", pollHandler.CreatePoll)
				protected.GET("/my", pollHandler.GetMyPolls)
				protected.PATCH("/:id/close", pollHandler.ClosePoll)
				protected.DELETE("/:id", pollHandler.DeletePoll)
			}
		}
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Printf("Server ready and listening on http://0.0.0.0:%s", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly")
}
