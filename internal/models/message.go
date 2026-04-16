package models

import "encoding/json"

const (
	TypeJoin      = "join"
	TypeLeave     = "leave"
	TypeMessage   = "message"
	TypeBroadcast = "broadcast"
	TypePing      = "ping"
	TypePong      = "pong"
	TypeError     = "error"
)

type IncomingMessage struct {
	Type string          `json:"type"`
	Room string          `json:"room,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type OutgoingMessage struct {
	Type     string          `json:"type"`
	Room     string          `json:"room,omitempty"`
	SenderID string          `json:"sender_id,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

type RedisEnvelope struct {
	OriginInstance string          `json:"origin_instance"`
	Room           string          `json:"room,omitempty"`
	Broadcast      bool            `json:"broadcast,omitempty"`
	Payload        json.RawMessage `json:"payload"`
}
