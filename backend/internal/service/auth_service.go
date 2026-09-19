package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/livepoll/backend/internal/models"
	"github.com/livepoll/backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired authentication token")
)

type AuthService struct {
	mongoRepo *repository.MongoRepo
	jwtSecret []byte
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

func NewAuthService(mongoRepo *repository.MongoRepo, jwtSecret string) *AuthService {
	return &AuthService{
		mongoRepo: mongoRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)

	if len(req.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters long")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        email,
		Name:         name,
		PasswordHash: string(hashedPassword),
	}

	if err := s.mongoRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User: models.UserInfo{
			ID:    user.ID.Hex(),
			Name:  user.Name,
			Email: user.Email,
		},
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.mongoRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User: models.UserInfo{
			ID:    user.ID.Hex(),
			Name:  user.Name,
			Email: user.Email,
		},
	}, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	claims := JWTClaims{
		UserID: user.ID.Hex(),
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.Hex(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) GetUserByID(ctx context.Context, userID primitive.ObjectID) (*models.UserInfo, error) {
	user, err := s.mongoRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &models.UserInfo{
		ID:    user.ID.Hex(),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}
