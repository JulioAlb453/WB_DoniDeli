package redis

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"WB-donideli/internal/models"

	"github.com/redis/go-redis/v9"
)

const channel = "ws:messages"

type Service struct {
	client *redis.Client
}

func New(addr string) (*Service, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	slog.Info("connected to Redis", "addr", addr)
	return &Service{client: rdb}, nil
}

func (s *Service) Publish(ch string, data []byte) error {
	return s.client.Publish(context.Background(), ch, data).Err()
}

type MessageReceiver func(models.RedisEnvelope)


func (s *Service) Subscribe(ctx context.Context, receiver MessageReceiver) {
	sub := s.client.Subscribe(ctx, channel)
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case redisMsg, ok := <-ch:
			if !ok {
				return
			}
			var env models.RedisEnvelope
			if err := json.Unmarshal([]byte(redisMsg.Payload), &env); err != nil {
				slog.Warn("invalid redis message", "error", err)
				continue
			}
			receiver(env)
		}
	}
}

func (s *Service) Close() error {
	return s.client.Close()
}
