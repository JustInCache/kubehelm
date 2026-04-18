package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env                string
	Port               string
	LogLevel           string
	JWTSecret          string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	ReadTimeoutMS      int
	ExpensiveRateRPS   int
	ExpensiveRateBurst int
	DBHost             string
	DBPort             int
	DBName             string
	DBUser             string
	DBPassword         string
	DBSSLMode          string
	UseMemoryStore     bool
}

func Load() Config {
	return Config{
		Env:                getEnv("APP_ENV", "development"),
		Port:               getEnv("PORT", "8080"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		JWTSecret:          getEnv("JWT_SECRET", "dev-secret-change-me"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("REDIS_DB", 0),
		ReadTimeoutMS:      getEnvInt("READ_TIMEOUT_MS", 8000),
		ExpensiveRateRPS:   getEnvInt("EXPENSIVE_RATE_RPS", 3),
		ExpensiveRateBurst: getEnvInt("EXPENSIVE_RATE_BURST", 8),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnvInt("DB_PORT", 5432),
		DBName:             getEnv("DB_NAME", "kubeaudit"),
		DBUser:             getEnv("DB_USER", "kubeaudit"),
		DBPassword:         getEnv("DB_PASSWORD", "kubeaudit_secret"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		UseMemoryStore:     getEnvBool("USE_MEMORY_STORE", false),
	}
}

func NewLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(handler)
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
