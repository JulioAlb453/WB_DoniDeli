package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	RedisURL          string
	JWTSecret         string
	HeartbeatInterval time.Duration

	MaxMessageSize    int64
	WriteWait         time.Duration
	ReadWait          time.Duration
	MaxConnsPerClient int
	SendBufferSize    int
}

func Load() *Config {
	return &Config{
		Port:              envOrDefault("PORT", "8080"),
		RedisURL:          envOrDefault("REDIS_URL", "localhost:6379"),
		JWTSecret:         envOrDefault("JWT_SECRET", "change-me-in-production"),
		HeartbeatInterval: durationOrDefault("HEARTBEAT_INTERVAL", 30*time.Second),
		MaxMessageSize:    int64(intOrDefault("MAX_MESSAGE_SIZE", 4096)),
		WriteWait:         10 * time.Second,
		ReadWait:          60 * time.Second,
		MaxConnsPerClient: intOrDefault("MAX_CONNS_PER_CLIENT", 5),
		SendBufferSize:    256,
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
