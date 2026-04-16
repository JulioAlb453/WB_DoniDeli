package hub

import (
	"encoding/json"
	"log/slog"
	"sync"

	"WB-donideli/internal/client"
	"WB-donideli/internal/models"
)

type RedisPublisher interface {
	Publish(channel string, data []byte) error
}

type Hub struct {
	clients    map[*client.Client]bool
	rooms      map[string]map[*client.Client]bool
	register   chan *client.Client
	unregister chan *client.Client
	mu         sync.RWMutex
	instanceID string
	redisPub   RedisPublisher
}

func New(instanceID string, redisPub RedisPublisher) *Hub {
	return &Hub{
		clients:    make(map[*client.Client]bool),
		rooms:      make(map[string]map[*client.Client]bool),
		register:   make(chan *client.Client, 64),
		unregister: make(chan *client.Client, 64),
		instanceID: instanceID,
		redisPub:   redisPub,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			slog.Info("client registered", "client", c.ID)

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				h.removeFromAllRooms(c)
			}
			h.mu.Unlock()
			slog.Info("client unregistered", "client", c.ID)
		}
	}
}

func (h *Hub) Register(c *client.Client) {
	h.register <- c
}

func (h *Hub) Unregister(c *client.Client) {
	h.unregister <- c
}

func (h *Hub) HandleMessage(sender *client.Client, msg models.IncomingMessage) {
	switch msg.Type {
	case models.TypeJoin:
		h.joinRoom(sender, msg.Room)
	case models.TypeLeave:
		h.leaveRoom(sender, msg.Room)
	case models.TypeMessage:
		h.sendToRoom(sender, msg)
	case models.TypeBroadcast:
		h.broadcastAll(sender, msg)
	default:
		slog.Warn("unknown message type", "type", msg.Type, "client", sender.ID)
	}
}

func (h *Hub) joinRoom(c *client.Client, room string) {
	if room == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*client.Client]bool)
	}
	h.rooms[room][c] = true
	c.SetRoom(room)
	slog.Info("client joined room", "client", c.ID, "room", room)
}

func (h *Hub) leaveRoom(c *client.Client, room string) {
	if room == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if members, ok := h.rooms[room]; ok {
		delete(members, c)
		if len(members) == 0 {
			delete(h.rooms, room)
		}
	}
	if c.GetRoom() == room {
		c.SetRoom("")
	}
	slog.Info("client left room", "client", c.ID, "room", room)
}

func (h *Hub) sendToRoom(sender *client.Client, msg models.IncomingMessage) {
	if msg.Room == "" {
		return
	}

	out := models.OutgoingMessage{
		Type:     models.TypeMessage,
		Room:     msg.Room,
		SenderID: sender.ID,
		Data:     msg.Data,
	}
	payload, err := json.Marshal(out)
	if err != nil {
		return
	}

	h.deliverToRoom(msg.Room, payload, sender)
	h.publishToRedis(msg.Room, false, payload)
}

func (h *Hub) broadcastAll(sender *client.Client, msg models.IncomingMessage) {
	out := models.OutgoingMessage{
		Type:     models.TypeBroadcast,
		SenderID: sender.ID,
		Data:     msg.Data,
	}
	payload, err := json.Marshal(out)
	if err != nil {
		return
	}

	h.deliverToAll(payload, sender)
	h.publishToRedis("", true, payload)
}

func (h *Hub) DeliverFromRedis(env models.RedisEnvelope) {
	if env.OriginInstance == h.instanceID {
		return
	}
	if env.Broadcast {
		h.deliverToAll(env.Payload, nil)
	} else if env.Room != "" {
		h.deliverToRoom(env.Room, env.Payload, nil)
	}
}

func (h *Hub) deliverToRoom(room string, payload []byte, exclude *client.Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	members, ok := h.rooms[room]
	if !ok {
		return
	}
	for c := range members {
		if c == exclude {
			continue
		}
		h.safeSend(c, payload)
	}
}

func (h *Hub) deliverToAll(payload []byte, exclude *client.Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c == exclude {
			continue
		}
		h.safeSend(c, payload)
	}
}

func (h *Hub) safeSend(c *client.Client, payload []byte) {
	select {
	case c.Send <- payload:
	default:
		slog.Warn("slow client, dropping message", "client", c.ID)
	}
}

func (h *Hub) removeFromAllRooms(c *client.Client) {
	for room, members := range h.rooms {
		delete(members, c)
		if len(members) == 0 {
			delete(h.rooms, room)
		}
	}
}

func (h *Hub) publishToRedis(room string, broadcast bool, payload []byte) {
	if h.redisPub == nil {
		return
	}
	env := models.RedisEnvelope{
		OriginInstance: h.instanceID,
		Room:           room,
		Broadcast:      broadcast,
		Payload:        payload,
	}
	data, err := json.Marshal(env)
	if err != nil {
		slog.Error("failed to marshal redis envelope", "error", err)
		return
	}
	if err := h.redisPub.Publish("ws:messages", data); err != nil {
		slog.Error("redis publish failed", "error", err)
	}
}

func (h *Hub) ActiveClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) CountUserConnections(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for c := range h.clients {
		if c.ID == userID {
			count++
		}
	}
	return count
}
