package main

import (
	"encoding/json"
	"errors"

	"github.com/jrallison/go-workers"
)

// WorkerClient is an interface for enqueueing workers
type WorkerClient interface {
	// Push pushes a worker onto the queue
	Push(class, args string) (string, error)
}

type redisWorkerClient struct {
	queue string
}

// NewRedisWorkerClient creates a worker client that pushes the worker to redis
func NewRedisWorkerClient(redis RedisConfig) (WorkerClient, error) {
	if redis.Host == "" {
		return nil, errors.New("Redis host required")
	}

	if redis.Queue == "" {
		return nil, errors.New("Sidekiq queue required")
	}

	workerConfig := map[string]string{
		"server":    redis.Host,
		"database":  "0",
		"pool":      "20",
		"process":   "1",
		"namespace": redis.Namespace,
	}

	if redis.Password != "" {
		workerConfig["password"] = redis.Password
	}

	workers.Configure(workerConfig)

	return &redisWorkerClient{queue: redis.Queue}, nil
}

func (r *redisWorkerClient) Push(class, args string) (string, error) {
	// This will hopefully deserialize on the ruby end as a hash
	jsonArgs := json.RawMessage([]byte(args))
	return workers.EnqueueWithOptions(
		r.queue,
		class,
		[]*json.RawMessage{&jsonArgs},
		workers.EnqueueOptions{
			Retry: true,
		},
	)
}
