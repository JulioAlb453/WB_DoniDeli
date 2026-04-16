package handler

import (
	"log/slog"
	"net/http"

	"WB-donideli/internal/auth"
	"WB-donideli/internal/client"
	"WB-donideli/internal/config"
	"WB-donideli/internal/hub"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub *hub.Hub
	cfg *config.Config
}

func NewWSHandler(h *hub.Hub, cfg *config.Config) *WSHandler {
	return &WSHandler{hub: h, cfg: cfg}
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

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}

	c := client.New(userID, conn, wh.cfg, wh.hub)
	wh.hub.Register(c)

	go c.WritePump()
	go c.ReadPump()
}
