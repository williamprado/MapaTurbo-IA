package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"mapaturbo-ia/internal/auth"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/internal/middleware"
	"mapaturbo-ia/internal/organizations"
	"mapaturbo-ia/internal/plans"
	"mapaturbo-ia/internal/users"
	"mapaturbo-ia/internal/settings"
	"mapaturbo-ia/internal/audit"
	"mapaturbo-ia/internal/uploads"
	"mapaturbo-ia/internal/payments"
	"mapaturbo-ia/internal/ai"
	"mapaturbo-ia/internal/mindmaps"
	"mapaturbo-ia/pkg/config"
	dbpkg "mapaturbo-ia/pkg/database"
	"mapaturbo-ia/pkg/logger"
	"mapaturbo-ia/pkg/queue"
	"mapaturbo-ia/pkg/storage"
	"mapaturbo-ia/pkg/validator"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger.InitLogger(cfg.AppEnv)
	defer logger.Log.Sync()

	logger.Log.Info("Starting MapaTurbo IA API...", zap.String("env", cfg.AppEnv))

	logger.Log.Info("Applying database migrations...")
	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Log.Fatal("Failed to open DB for migrations", zap.Error(err))
	}
	if err := goose.SetDialect("postgres"); err != nil {
		logger.Log.Fatal("Failed to set goose dialect", zap.Error(err))
	}
	migrationsDir := "db/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = "../db/migrations"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			migrationsDir = "backend/db/migrations"
		}
	}
	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		logger.Log.Fatal("Database migrations failed", zap.Error(err))
	}
	sqlDB.Close()
	logger.Log.Info("Database migrations applied successfully")

	pool, err := dbpkg.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database pool", zap.Error(err))
	}
	defer pool.Close()

	logger.Log.Info("Running database seed/bootstrap...")
	if err := database.SeedBootstrapAdmin(context.Background(), pool); err != nil {
		logger.Log.Fatal("Database seed failed", zap.Error(err))
	}

	validator.InitValidator()

	_, err = storage.InitS3(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioUseSSL)
	if err != nil {
		logger.Log.Error("MinIO storage initialization warning (will retry later)", zap.Error(err))
	}

	_, err = queue.InitQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Log.Error("Redis queue initialization warning (will retry later)", zap.Error(err))
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Log.Info("HTTP Request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	})

	r.Use(corsMiddleware())

	authHandler := auth.NewHandler(pool, cfg.JWTSecret)
	orgHandler := organizations.NewHandler(pool)
	planHandler := plans.NewHandler(pool)
	userHandler := users.NewHandler(pool)
	settingsHandler := settings.NewHandler(pool)
	auditHandler := audit.NewHandler(pool)
	uploadHandler := uploads.NewHandler(pool, storage.Client)
	payHandler := payments.NewHandler(pool, cfg.EncryptionKey)
	aiHandler := ai.NewHandler(pool, cfg.EncryptionKey)
	mindmapHandler := mindmaps.NewHandler(pool)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "OK",
			"time":   time.Now().Format(time.RFC3339),
		})
	})
	r.GET("/plans/public", planHandler.ListPublic)
	r.POST("/webhooks/asaas", payHandler.HandleAsaasWebhook)

	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/refresh", authHandler.Refresh)

	authGroup := r.Group("")
	authGroup.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		authGroup.GET("/auth/me", authHandler.Me)
		authGroup.POST("/auth/logout", authHandler.Logout)

		tenantGroup := authGroup.Group("")
		tenantGroup.Use(middleware.TenantMiddleware(pool))
		{
			tenantGroup.POST("/uploads", uploadHandler.Upload)
			tenantGroup.GET("/uploads", uploadHandler.List)
			tenantGroup.GET("/uploads/:id", uploadHandler.GetByID)
			tenantGroup.GET("/credits/balance", orgHandler.GetBalance)
			tenantGroup.POST("/billing/checkout", payHandler.CreateCheckout)
			tenantGroup.GET("/billing/invoices", payHandler.ListInvoices)

			// Mindmaps routes
			tenantGroup.POST("/mindmaps/generate", mindmapHandler.Generate)
			tenantGroup.POST("/mindmaps/generate-from-upload", mindmapHandler.GenerateFromUpload)
			tenantGroup.GET("/generation-jobs/:id", mindmapHandler.GetJob)
			tenantGroup.GET("/generation-jobs", mindmapHandler.ListJobs)
			tenantGroup.GET("/mindmaps", mindmapHandler.ListMindMaps)
			tenantGroup.GET("/mindmaps/:id", mindmapHandler.GetMindMap)
			tenantGroup.PATCH("/mindmaps/:id", mindmapHandler.UpdateMindMap)
			tenantGroup.DELETE("/mindmaps/:id", mindmapHandler.DeleteMindMap)
		}

		adminGroup := authGroup.Group("/admin")
		adminGroup.Use(middleware.RequireSuperAdmin())
		{
			adminGroup.GET("/organizations", orgHandler.List)
			adminGroup.POST("/organizations", orgHandler.Create)
			adminGroup.GET("/organizations/:id", orgHandler.GetByID)
			adminGroup.PATCH("/organizations/:id", orgHandler.Update)
			adminGroup.GET("/organizations/:id/users", orgHandler.ListUsers)
			adminGroup.POST("/organizations/:id/users", orgHandler.AddUser)
			adminGroup.DELETE("/organizations/:id/users/:userId", orgHandler.RemoveUser)
			adminGroup.PATCH("/organizations/:id/users/:userId/role", orgHandler.UpdateUserRole)

			adminGroup.GET("/plans", planHandler.List)
			adminGroup.POST("/plans", planHandler.Create)
			adminGroup.GET("/plans/:id", planHandler.GetByID)
			adminGroup.PATCH("/plans/:id", planHandler.Update)

			adminGroup.GET("/users", userHandler.List)
			adminGroup.GET("/users/:id", userHandler.GetByID)
			adminGroup.PATCH("/users/:id", userHandler.Update)

			// Settings routes
			adminGroup.GET("/settings", settingsHandler.List)
			adminGroup.PATCH("/settings/:key", settingsHandler.Update)

			// AI Action Prices routes
			adminGroup.GET("/ai-action-prices", settingsHandler.ListAiActionPrices)
			adminGroup.PATCH("/ai-action-prices/:id", settingsHandler.UpdateAiActionPrice)

			// Subscriptions manual admin routes
			adminGroup.GET("/subscriptions", planHandler.ListSubscriptions)
			adminGroup.POST("/subscriptions/manual", planHandler.CreateManual)

			// Audit Logs admin routes
			adminGroup.GET("/audit-logs", auditHandler.List)

			// AI Providers routes
			adminGroup.GET("/ai-providers", aiHandler.ListProviders)
			adminGroup.POST("/ai-providers", aiHandler.CreateProvider)
			adminGroup.PATCH("/ai-providers/:id", aiHandler.UpdateProvider)
			adminGroup.POST("/ai-providers/:id/test", aiHandler.TestProviderConnection)

			// Payment Providers routes
			adminGroup.GET("/payments/providers", payHandler.ListPaymentProvidersAdmin)
			adminGroup.PATCH("/payments/providers/:id", payHandler.UpdatePaymentProviderAdmin)
			adminGroup.GET("/payments/invoices", payHandler.ListInvoicesAdmin)
			adminGroup.GET("/payments/transactions", payHandler.ListTransactionsAdmin)
			adminGroup.GET("/payments/webhook-events", payHandler.ListWebhookEventsAdmin)
		}
	}

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	logger.Log.Info("Server running", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Log.Fatal("Failed to start server", zap.Error(err))
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Organization-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
