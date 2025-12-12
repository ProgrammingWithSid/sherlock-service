package queue

import (
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	"github.com/sherlock/service/internal/types"
)

type ReviewQueue struct {
	client    *asynq.Client
	server    *asynq.Server
	redisAddr string
}

func NewRedisClient(redisURL string) *redis.Client {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	}
	return redis.NewClient(opt)
}

func NewReviewQueue(redisClient *redis.Client) *ReviewQueue {
	redisAddr := redisClient.Options().Addr
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr: redisAddr,
	})

	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 5,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	return &ReviewQueue{
		client:    client,
		server:    server,
		redisAddr: redisAddr,
	}
}

func (q *ReviewQueue) EnqueueReviewJob(job *types.ReviewJob, priority int) (string, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}

	queueName := "default"
	if priority >= 50 {
		queueName = "critical"
	} else if priority < 10 {
		queueName = "low"
	}

	task := asynq.NewTask("review", payload)
	info, err := q.client.Enqueue(task, asynq.Queue(queueName))
	if err != nil {
		return "", fmt.Errorf("failed to enqueue job: %w", err)
	}

	return info.ID, nil
}

func (q *ReviewQueue) EnqueueCommandJob(job *types.CommandJob) (string, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}

	task := asynq.NewTask("command", payload)
	info, err := q.client.Enqueue(task, asynq.Queue("default"))
	if err != nil {
		return "", fmt.Errorf("failed to enqueue job: %w", err)
	}

	return info.ID, nil
}

func (q *ReviewQueue) Close() error {
	return q.client.Close()
}

func (q *ReviewQueue) GetServer() *asynq.Server {
	return q.server
}

func (q *ReviewQueue) GetInspector() *asynq.Inspector {
	return asynq.NewInspector(asynq.RedisClientOpt{
		Addr: q.redisAddr,
	})
}
