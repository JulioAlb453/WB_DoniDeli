package config

import (
	"log/slog"
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
	cfg := &Config{
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
	}

	apiBase := apiBaseURLFromEnv()
	peer, err := fetchContactoChatPeer(apiBase)
	if err != nil {
		if apiBase == "" {
			slog.Warn("peer de chat no resuelto: el servidor WS es otro proceso distinto al front Angular; " +
				"debes exportar la misma base de API aquí, p. ej. API_BASE_URL=http://127.0.0.1:8000 " +
				"(o DONUTS_API_BASE_URL). Si la API sí está levantada pero ves esto, falta esa variable en el shell que ejecuta go run.")
		} else {
			slog.Warn("no se pudo obtener peer desde la API", "error", err, "api_base", apiBase)
		}
	}
	if peer == "" {
		peer = strings.TrimSpace(os.Getenv("ADMIN_CHAT_PEER_ID"))
		if peer == "" {
			slog.Warn("ADMIN_CHAT_PEER_ID vacío: copia inbox del hub desactivada; define API_BASE_URL (o DONUTS_API_BASE_URL) o ADMIN_CHAT_PEER_ID")
		}
	} else {
		slog.Info("admin chat peer resuelto desde API (usuario admin en BD)", "peer", peer)
	}

	inbox := strings.TrimSpace(os.Getenv("ADMIN_WS_INBOX_USER_ID"))
	if inbox == "" {
		inbox = peer
	}

	cfg.AdminChatPeerID = peer
	cfg.AdminWsInboxUserID = inbox
	return cfg
}

func apiBaseURLFromEnv() string {
	for _, key := range []string{"API_BASE_URL", "DONUTS_API_BASE_URL"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
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
