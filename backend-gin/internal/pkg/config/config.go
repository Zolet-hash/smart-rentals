package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server struct {
		Port         string
		Host         string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
	}
	Database struct {
		Host        string
		Port        string
		User        string
		Password    string
		DBName      string
		SSLMode     string
		DatabaseURL string
	}
	JWT struct {
		Secret        string
		TokenExpiry   time.Duration
		RefreshExpiry time.Duration
	}
	Environment string
}

func Load() (*Config, error) {
	// Load .env for local development only
	_ = godotenv.Load()

	cfg := &Config{}

	// Server config
	// Railway uses PORT, local dev can use SERVER_PORT
	cfg.Server.Port = getEnv("PORT", getEnv("SERVER_PORT", "8080"))
	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")

	cfg.Server.ReadTimeout = 10 * time.Second
	cfg.Server.WriteTimeout = 10 * time.Second

	// Database config
	// Railway provides DATABASE_URL (preferred)
	cfg.Database.DatabaseURL = os.Getenv("DATABASE_URL")

	// Fallback for local development
	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnv("DB_PORT", "5432")
	cfg.Database.User = getEnv("DB_USER", "zoet")
	cfg.Database.Password = getEnv("DB_PASSWORD", "")
	cfg.Database.DBName = getEnv("DB_NAME", "auth_service")

	// Default to require SSL (Railway needs this)
	cfg.Database.SSLMode = getEnv("DB_SSLMODE", "require")

	// JWT config
	cfg.JWT.Secret = getEnv("JWT_SECRET", "")
	cfg.JWT.TokenExpiry = time.Hour * 24      // 24 hours
	cfg.JWT.RefreshExpiry = time.Hour * 168   // 7 days

	// Environment
	cfg.Environment = getEnv("ENV", "development")

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetDSN returns the correct database connection string
func (c *Config) GetDSN() string {
	// Prefer Railway / production connection
	if c.Database.DatabaseURL != "" {
		return c.Database.DatabaseURL
	}

	// Fallback for local development
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}
