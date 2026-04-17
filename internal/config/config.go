package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port              string
	RedisURL          string
	JWTSecret         string
	HeartbeatInterval time.Duration
	AllowedOrigins    []string

	MaxMessageSize    int64
	WriteWait         time.Duration
	ReadWait          time.Duration
	MaxConnsPerClient int
	SendBufferSize    int

	AdminChatPeerID string
	AdminWsInboxUserID string
}

func Load() *Config {
	peer := strings.TrimSpace(envOrDefault("ADMIN_CHAT_PEER_ID", "admin@gmail.com"))
	inbox := strings.TrimSpace(envOrDefault("ADMIN_WS_INBOX_USER_ID", ""))
	if inbox == "" {
		inbox = peer
	}
	return &Config{
		Port:              envOrDefault("PORT", "8080"),
		RedisURL:          envOrDefault("REDIS_URL", "localhost:6379"),
		JWTSecret:         envOrDefault("JWT_SECRET", "change-me-in-production"),
		HeartbeatInterval: durationOrDefault("HEARTBEAT_INTERVAL", 30*time.Second),
		AllowedOrigins:    parseOrigins(envOrDefault("ALLOWED_ORIGINS", "*")),
		MaxMessageSize:    int64(intOrDefault("MAX_MESSAGE_SIZE", 4096)),
		WriteWait:         10 * time.Second,
		ReadWait:          60 * time.Second,
		MaxConnsPerClient: intOrDefault("MAX_CONNS_PER_CLIENT", 5),
		SendBufferSize:    256,
		AdminChatPeerID:   peer,
		AdminWsInboxUserID: inbox,
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

func parseOrigins(raw string) []string {
	if raw == "*" {
		return []string{"*"}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			origins = append(origins, s)
		}
	}
	return origins
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
