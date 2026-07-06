package database

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"mapaturbo-ia/pkg/crypto"
	"mapaturbo-ia/pkg/logger"
)

func SeedBootstrapAdmin(ctx context.Context, db *pgxpool.Pool) error {
	queries := New(db)

	email := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	if email == "" {
		email = "admin@admin.com"
	}
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if password == "" {
		password = "@Admin2328"
	}

	_, err := queries.GetUserByEmail(ctx, email)
	if err == nil {
		logger.Log.Info("Bootstrap Admin already exists. Skipping admin seed.")
	} else if errors.Is(err, pgx.ErrNoRows) {
		logger.Log.Info("Creating Bootstrap Admin...", zap.String("email", email))

		passwordHash, err := crypto.HashPassword(password)
		if err != nil {
			return err
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		txQueries := queries.WithTx(tx)

		user, err := txQueries.CreateUser(ctx, CreateUserParams{
			Email:        email,
			PasswordHash: passwordHash,
			Name:         "Super Admin",
			GlobalRole:   "SUPER_ADMIN",
			Status:       "ACTIVE",
		})
		if err != nil {
			return err
		}

		org, err := txQueries.CreateOrganization(ctx, CreateOrganizationParams{
			Name:   "MapaTurbo",
			Slug:   "mapaturbo",
			Status: "ACTIVE",
		})
		if err != nil {
			return err
		}

		_, err = txQueries.CreateOrganizationUser(ctx, CreateOrganizationUserParams{
			OrganizationID: org.ID,
			UserID:         user.ID,
			Role:           "ORG_ADMIN",
		})
		if err != nil {
			return err
		}

		_, err = txQueries.InitializeCreditBalance(ctx, InitializeCreditBalanceParams{
			OrganizationID: org.ID,
			Balance:        1000,
		})
		if err != nil {
			return err
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"email": email,
			"org":   org.Slug,
		})
		_, err = txQueries.CreateAuditLog(ctx, CreateAuditLogParams{
			ActorUserID:    user.ID,
			OrganizationID: org.ID,
			Action:         "SUPER_ADMIN_CREATED",
			EntityType:     "users",
			EntityID:       user.ID,
			Metadata:       meta,
			Ip:             pgtype.Text{String: "127.0.0.1", Valid: true},
			UserAgent:      pgtype.Text{String: "System Bootstrap", Valid: true},
		})
		if err != nil {
			return err
		}

		err = tx.Commit(ctx)
		if err != nil {
			return err
		}

		logger.Log.Info("Bootstrap Admin created successfully!")
	} else {
		return err
	}

	actions := []struct {
		Key  string
		Name string
		Cost int32
	}{
		{"GENERATE_MAP_TOPIC", "Generate Mind Map by Topic", 10},
		{"GENERATE_MAP_TEXT", "Generate Mind Map by Text", 15},
		{"GENERATE_MAP_PDF", "Generate Mind Map by PDF", 20},
		{"GENERATE_FLASHCARDS", "Generate Flashcards", 5},
		{"EXPAND_NODE", "Expand Node with IA", 2},
		{"SUMMARIZE_NODE", "Summarize Node with IA", 2},
		{"EXPORT_PDF", "Export to PDF", 0},
		{"EXPORT_IMAGE", "Export to Image", 0},
	}

	for _, action := range actions {
		_, err := queries.GetAiActionPrice(ctx, action.Key)
		if errors.Is(err, pgx.ErrNoRows) {
			_, err = queries.CreateAiActionPrice(ctx, CreateAiActionPriceParams{
				ActionKey:   action.Key,
				Name:        action.Name,
				CreditsCost: action.Cost,
				IsActive:    true,
			})
			if err != nil {
				logger.Log.Error("Failed to seed action price", zap.String("key", action.Key), zap.Error(err))
			}
		}
	}

	settings := []struct {
		Key         string
		Value       interface{}
		Description string
	}{
		{"maintenance_mode", false, "Enable or disable global maintenance mode"},
		{"public_signup_enabled", true, "Enable or disable public signup"},
		{"default_trial_days", 7, "Default trial days for new signups"},
		{"default_currency", "BRL", "Default platform currency"},
		{"default_credits_on_signup", 100, "Default credits allocated to new signups"},
	}

	for _, setting := range settings {
		_, err := queries.GetSystemSetting(ctx, setting.Key)
		if errors.Is(err, pgx.ErrNoRows) {
			valBytes, _ := json.Marshal(setting.Value)
			_, err = queries.UpsertSystemSetting(ctx, UpsertSystemSettingParams{
				Key:         setting.Key,
				Value:       valBytes,
				Description: pgtype.Text{String: setting.Description, Valid: true},
				IsPublic:    true,
			})
			if err != nil {
				logger.Log.Error("Failed to seed system setting", zap.String("key", setting.Key), zap.Error(err))
			}
		}
	}

	return nil
}
