package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	APIKey            string
	JWTSecret         string
	JWTIssuer         string
	JWTTTL            time.Duration
	BootstrapEmail    string
	BootstrapName     string
	BootstrapPassword string
	PublicBaseURL     string
	AutoMigrate       bool
	StorageDriver     string
	LocalStoragePath  string
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKey       string
	S3SecretKey       string
	S3PublicBaseURL   string
	S3UseSSL          bool
}

func Load() Config {
	return Config{
		HTTPAddr:          env("HTTP_ADDR", ":8080"),
		DatabaseURL:       env("DATABASE_URL", ""),
		APIKey:            env("API_KEY", ""),
		JWTSecret:         env("JWT_SECRET", "dev-secret-change-me"),
		JWTIssuer:         env("JWT_ISSUER", "multi-platform-distribution"),
		JWTTTL:            envDuration("JWT_TTL", 24*time.Hour),
		BootstrapEmail:    env("BOOTSTRAP_ADMIN_EMAIL", ""),
		BootstrapName:     env("BOOTSTRAP_ADMIN_NAME", "Administrator"),
		BootstrapPassword: env("BOOTSTRAP_ADMIN_PASSWORD", ""),
		PublicBaseURL:     strings.TrimRight(env("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		AutoMigrate:       envBool("AUTO_MIGRATE", true),
		StorageDriver:     env("STORAGE_DRIVER", "local"),
		LocalStoragePath:  env("LOCAL_STORAGE_PATH", "data/uploads"),
		S3Endpoint:        env("S3_ENDPOINT", ""),
		S3Region:          env("S3_REGION", "us-east-1"),
		S3Bucket:          env("S3_BUCKET", ""),
		S3AccessKey:       env("S3_ACCESS_KEY", ""),
		S3SecretKey:       env("S3_SECRET_KEY", ""),
		S3PublicBaseURL:   strings.TrimRight(env("S3_PUBLIC_BASE_URL", ""), "/"),
		S3UseSSL:          envBool("S3_USE_SSL", false),
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "":
		return fallback
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}
