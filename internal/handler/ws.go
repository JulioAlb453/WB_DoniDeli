package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"WB-donideli/internal/auth"
	"WB-donideli/internal/client"
	"WB-donideli/internal/config"
	"WB-donideli/internal/hub"

	"github.com/gorilla/websocket"
)

type WSHandler struct {
	hub      *hub.Hub
	cfg      *config.Config
	upgrader websocket.Upgrader
}

func NewWSHandler(h *hub.Hub, cfg *config.Config) *WSHandler {
	wh := &WSHandler{hub: h, cfg: cfg}
	wh.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     wh.checkOrigin,
	}
	return wh
}

func (wh *WSHandler) checkOrigin(r *http.Request) bool {
	if len(wh.cfg.AllowedOrigins) == 1 && wh.cfg.AllowedOrigins[0] == "*" {
		return true
	}
	origin := r.Header.Get("Origin")
	for _, allowed := range wh.cfg.AllowedOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	slog.Warn("origin rejected", "origin", origin)
	return false
}

func (wh *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateToken(token, wh.cfg.JWTSecret)
	if err != nil {
		slog.Warn("auth failed", "error", err, "remote", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if wh.hub.CountUserConnections(userID) >= wh.cfg.MaxConnsPerClient {
		http.Error(w, "too many connections", http.StatusTooManyRequests)
		return
	}

	conn, err := wh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}

	c := client.New(userID, conn, wh.cfg, wh.hub)
	wh.hub.Register(c)

	go c.WritePump()
	go c.ReadPump()
}
