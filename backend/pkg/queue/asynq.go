package queue

import (
	"fmt"

	"github.com/hibiken/asynq"
	"mapaturbo-ia/pkg/logger"
)

var Client *asynq.Client

func InitQueue(redisAddr string, redisPassword string, redisDB int) (*asynq.Client, error) {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	Client = client
	logger.Log.Info("Initialized Asynq Client successfully")
	return client, nil
}

func EnqueueTask(taskType string, payload []byte, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if Client == nil {
		return nil, fmt.Errorf("asynq client is not initialized")
	}

	task := asynq.NewTask(taskType, payload)
	info, err := Client.Enqueue(task, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue task: %w", err)
	}

	return info, nil
}
