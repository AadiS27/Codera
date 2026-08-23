package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/codera/code-executor/internal/config"
	"github.com/redis/go-redis/v9"
)

type RedisDB struct {
	Client *redis.Client
	logger *slog.Logger
}

func ConnectRedis(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*RedisDB, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("Connected to Redis successfully")

	return &RedisDB{
		Client: client,
		logger: logger,
	}, nil
}

func (r *RedisDB) Close() {
	if r.Client != nil {
		r.logger.Info("Closing Redis connection")
		r.Client.Close()
	}
}

func (r *RedisDB) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}
