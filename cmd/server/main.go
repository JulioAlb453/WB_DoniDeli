package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"strconv"
	"time"

	"WB-donideli/internal/config"
	"WB-donideli/internal/handler"
	"WB-donideli/internal/hub"
	redissvc "WB-donideli/internal/redis"

	"github.com/google/uuid"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	instanceID := uuid.NewString()

	slog.Info("starting server", "instance", instanceID, "port", cfg.Port)

	var redisSvc *redissvc.Service
	var redisPub hub.RedisPublisher

	redisSvc, err := redissvc.New(cfg.RedisURL)
	if err != nil {
		slog.Warn("redis unavailable, running without horizontal scaling", "error", err)
	} else {
		redisPub = redisSvc
	}

	h := hub.New(instanceID, redisPub, cfg.AdminChatPeerID, cfg.AdminWsInboxUserID)
	go h.Run()

	redisCtx, redisCancel := context.WithCancel(context.Background())
	defer redisCancel()

	if redisSvc != nil {
		go redisSvc.Subscribe(redisCtx, h.DeliverFromRedis)
	}

	mux := http.NewServeMux()
	wsHandler := handler.NewWSHandler(h, cfg)
	tokenHandler := handler.NewTokenHandler(cfg)
	mux.Handle("/ws", wsHandler)
	mux.Handle("/auth/token", tokenHandler)
	mux.Handle("/", http.FileServer(http.Dir("static")))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","clients":` + itoa(h.ActiveClients()) + `}`))
	})

	withCORS := handler.CORSMiddleware(cfg.AllowedOrigins, mux)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      withCORS,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-quit
	slog.Info("shutting down", "signal", sig)

	redisCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced shutdown", "error", err)
	}

	if redisSvc != nil {
		_ = redisSvc.Close()
	}

	slog.Info("server stopped")
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
