package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	MongoURI     string
	MongoDBName  string
	RedisURL     string
	JWTSecret    string
	CORSOrigin   string
	Environment  string
	ReadTimeout  int
	WriteTimeout int
}

func LoadConfig() *Config {
	// Try loading .env if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading, reading from environment")
	}

	port := getEnv("PORT", "8080")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	mongoDBName := getEnv("MONGO_DB_NAME", "livepoll_db")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-livepoll-jwt-key-2026-change-in-prod")
	corsOrigin := getEnv("CORS_ORIGIN", "*")
	env := getEnv("APP_ENV", "development")

	readTimeout, _ := strconv.Atoi(getEnv("READ_TIMEOUT_SECONDS", "15"))
	writeTimeout, _ := strconv.Atoi(getEnv("WRITE_TIMEOUT_SECONDS", "15"))

	return &Config{
		Port:         port,
		MongoURI:     mongoURI,
		MongoDBName:  mongoDBName,
		RedisURL:     redisURL,
		JWTSecret:    jwtSecret,
		CORSOrigin:   corsOrigin,
		Environment:  env,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}
