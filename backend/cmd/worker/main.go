package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"mapaturbo-ia/internal/payments"
	"mapaturbo-ia/internal/mindmaps"
	"mapaturbo-ia/internal/uploads"
	"mapaturbo-ia/pkg/config"
	dbpkg "mapaturbo-ia/pkg/database"
	"mapaturbo-ia/pkg/logger"
	"mapaturbo-ia/pkg/storage"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger.InitLogger(cfg.AppEnv)
	defer logger.Log.Sync()

	logger.Log.Info("Starting MapaTurbo IA Worker...", zap.String("env", cfg.AppEnv))

	pool, err := dbpkg.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database pool", zap.Error(err))
	}
	defer pool.Close()

	_, err = storage.InitS3(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioUseSSL)
	if err != nil {
		logger.Log.Error("MinIO storage initialization warning in worker", zap.Error(err))
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		},
		asynq.Config{
			Concurrency: 5,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	payWorker := payments.NewWorker(pool)
	mindmapWorker := mindmaps.NewWorker(pool, cfg.EncryptionKey)
	uploadWorker := uploads.NewWorker(pool, cfg.EncryptionKey)

	mux := asynq.NewServeMux()

	mux.HandleFunc("health_check", handleHealthCheckTask)
	mux.HandleFunc("process_upload_placeholder", handleProcessUploadPlaceholderTask)
	mux.HandleFunc("process_pdf_upload", uploadWorker.ProcessPdfUploadTask)
	mux.HandleFunc("process_payment_webhook_placeholder", handleProcessPaymentWebhookPlaceholderTask)
	mux.HandleFunc("process_payment_webhook", payWorker.ProcessPaymentWebhookTask)
	mux.HandleFunc("generate_mindmap", mindmapWorker.ProcessGenerationTask)

	logger.Log.Info("Worker listening for jobs...")
	if err := srv.Run(mux); err != nil {
		logger.Log.Fatal("Asynq server error", zap.Error(err))
	}
}

func handleHealthCheckTask(ctx context.Context, t *asynq.Task) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.Log.Info("Processing health_check job in worker", zap.Any("payload", payload))
	return nil
}

func handleProcessUploadPlaceholderTask(ctx context.Context, t *asynq.Task) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Log.Error("Failed to parse process_upload_placeholder payload", zap.Error(err))
		return nil
	}

	logger.Log.Info("Processing upload job (placeholder)", zap.Any("payload", payload))
	return nil
}

func handleProcessPaymentWebhookPlaceholderTask(ctx context.Context, t *asynq.Task) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Log.Error("Failed to parse process_payment_webhook_placeholder payload", zap.Error(err))
		return nil
	}

	logger.Log.Info("Processing payment webhook job (placeholder)", zap.Any("payload", payload))
	return nil
}
