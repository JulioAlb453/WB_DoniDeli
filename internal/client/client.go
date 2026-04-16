package client

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"WB-donideli/internal/config"
	"WB-donideli/internal/models"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	Room   string
	mu     sync.RWMutex
	cfg    *config.Config
	hub    Hub
	closed bool
}

type Hub interface {
	Register(c *Client)
	Unregister(c *Client)
	HandleMessage(sender *Client, msg models.IncomingMessage)
}

func New(id string, conn *websocket.Conn, cfg *config.Config, hub Hub) *Client {
	return &Client{
		ID:   id,
		Conn: conn,
		Send: make(chan []byte, cfg.SendBufferSize),
		cfg:  cfg,
		hub:  hub,
	}
}

func (c *Client) GetRoom() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Room
}

func (c *Client) SetRoom(room string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Room = room
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Close()
	}()

	c.Conn.SetReadLimit(c.cfg.MaxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(c.cfg.ReadWait))

	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(c.cfg.ReadWait))
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("unexpected close", "client", c.ID, "error", err)
			}
			return
		}

		var msg models.IncomingMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.sendError("invalid JSON format")
			continue
		}

		if msg.Type == models.TypePing {
			c.sendPong()
			continue
		}

		c.hub.HandleMessage(c, msg)
	}
}


func (c *Client) WritePump() {
	ticker := time.NewTicker(c.cfg.HeartbeatInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				slog.Warn("write failed", "client", c.ID, "error", err)
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.Send)
		_ = c.Conn.Close()
	}
}

func (c *Client) sendError(detail string) {
	out := models.OutgoingMessage{Type: models.TypeError, Data: json.RawMessage(`"` + detail + `"`)}
	data, _ := json.Marshal(out)
	select {
	case c.Send <- data:
	default:
		slog.Warn("send buffer full, dropping error", "client", c.ID)
	}
}

func (c *Client) sendPong() {
	out := models.OutgoingMessage{Type: models.TypePong}
	data, _ := json.Marshal(out)
	select {
	case c.Send <- data:
	default:
	}
}
