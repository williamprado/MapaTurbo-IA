package ai

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/ai/providers/anthropic"
	"mapaturbo-ia/internal/ai/providers/gemini"
	"mapaturbo-ia/internal/ai/providers/grok"
	"mapaturbo-ia/internal/ai/providers/openai"
	"mapaturbo-ia/internal/ai/domain"
	"mapaturbo-ia/internal/database"
	cryptoPkg "mapaturbo-ia/pkg/crypto"
	"mapaturbo-ia/pkg/response"
	"mapaturbo-ia/pkg/validator"
)

type Handler struct {
	db            *pgxpool.Pool
	queries       *database.Queries
	encryptionKey string
}

func NewHandler(db *pgxpool.Pool, encryptionKey string) *Handler {
	return &Handler{
		db:            db,
		queries:       database.New(db),
		encryptionKey: encryptionKey,
	}
}

type AIProviderRequest struct {
	Name                string  `json:"name" validate:"required"`
	Slug                string  `json:"slug" validate:"required"`
	APIKey              string  `json:"apiKey"`
	BaseURL             string  `json:"baseUrl"`
	DefaultModel        string  `json:"defaultModel" validate:"required"`
	TextModel           string  `json:"textModel"`
	VisionModel         string  `json:"visionModel"`
	AudioModel          string  `json:"audioModel"`
	EmbeddingModel      string  `json:"embeddingModel"`
	EmbeddingDimensions int32   `json:"embeddingDimensions"`
	IsActive            bool    `json:"isActive"`
	Priority            int32   `json:"priority"`
	IsDefault           bool    `json:"isDefault"`
	LimitPerMinute      int32   `json:"limitPerMinute"`
	LimitPerDay         int32   `json:"limitPerDay"`
	CostPerCredit       float64 `json:"costPerCredit"`
}

func getAESKey(rawKey string) []byte {
	h := sha256.Sum256([]byte(rawKey))
	return h[:]
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

func (h *Handler) ListProviders(c *gin.Context) {
	providers, err := h.queries.ListAiProviders(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve AI providers")
		return
	}

	formatted := make([]map[string]interface{}, len(providers))
	for i, p := range providers {
		key := getAESKey(h.encryptionKey)
		var decryptedKey string
		if p.ApiKeySecure != "" {
			decKey, err := cryptoPkg.Decrypt(p.ApiKeySecure, key)
			if err == nil {
				decryptedKey = decKey
			}
		}

		formatted[i] = map[string]interface{}{
			"id":                  uuidToString(p.ID),
			"name":                p.Name,
			"slug":                p.Slug,
			"apiKey":              maskAPIKey(decryptedKey),
			"baseUrl":             p.BaseUrl,
			"defaultModel":        p.DefaultModel,
			"textModel":           p.TextModel,
			"visionModel":         p.VisionModel,
			"audioModel":          p.AudioModel,
			"embeddingModel":      p.EmbeddingModel,
			"embeddingDimensions": p.EmbeddingDimensions,
			"isActive":            p.IsActive,
			"priority":            p.Priority,
			"isDefault":           p.IsDefault,
			"limitPerMinute":      p.LimitPerMinute,
			"limitPerDay":         p.LimitPerDay,
			"costPerCredit":       p.CostPerCredit,
			"createdAt":           p.CreatedAt,
			"updatedAt":           p.UpdatedAt,
		}
	}

	response.Success(c, http.StatusOK, "AI providers list", formatted)
}

func (h *Handler) CreateProvider(c *gin.Context) {
	var req AIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	// Check unique slug
	_, err := h.queries.GetAiProviderBySlug(c.Request.Context(), req.Slug)
	if err == nil {
		response.BadRequest(c, "AI Provider with this slug already exists", nil)
		return
	}

	// Encrypt API key
	var encryptedKey string
	if req.APIKey != "" {
		enc, err := cryptoPkg.Encrypt(req.APIKey, getAESKey(h.encryptionKey))
		if err != nil {
			response.InternalServerError(c, "Failed to encrypt API key")
			return
		}
		encryptedKey = enc
	}

	tx, err := h.db.Begin(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to start database transaction")
		return
	}
	defer tx.Rollback(c.Request.Context())

	txQueries := h.queries.WithTx(tx)

	if req.IsDefault {
		err = txQueries.SetAllAiProvidersNotDefault(c.Request.Context())
		if err != nil {
			response.InternalServerError(c, "Failed to clear default flag on AI providers")
			return
		}
	}

	var costNumeric pgtype.Numeric
	costNumeric.Scan(fmt.Sprintf("%f", req.CostPerCredit))

	// Create Provider (we insert via manual SQL query since CreateAiProvider is missing from queries)
	var newID pgtype.UUID
	err = tx.QueryRow(c.Request.Context(),
		`INSERT INTO ai_providers (
			name, slug, api_key_secure, base_url, default_model, text_model, vision_model, audio_model, embedding_model, embedding_dimensions, is_active, priority, is_default, limit_per_minute, limit_per_day, cost_per_credit
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id`,
		req.Name, req.Slug, encryptedKey, req.BaseURL, req.DefaultModel,
		req.TextModel, req.VisionModel, req.AudioModel, req.EmbeddingModel,
		req.EmbeddingDimensions, req.IsActive, req.Priority, req.IsDefault,
		req.LimitPerMinute, req.LimitPerDay, costNumeric,
	).Scan(&newID)

	if err != nil {
		response.InternalServerError(c, "Failed to insert AI Provider: "+err.Error())
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		response.InternalServerError(c, "Failed to commit database changes")
		return
	}

	// Audit Log
	var userID pgtype.UUID
	userIDStr, exists := c.Get("user_id")
	if exists {
		_ = userID.Scan(userIDStr)
	}
	meta, _ := json.Marshal(map[string]string{"slug": req.Slug})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: userID,
		Action:      "AI_PROVIDER_CREATED",
		EntityType:  "ai_providers",
		EntityID:    newID,
		Metadata:    meta,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusCreated, "AI Provider registered successfully", gin.H{"id": uuidToString(newID)})
}

func (h *Handler) UpdateProvider(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid AI Provider ID format", nil)
		return
	}

	var req AIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	provider, err := h.queries.GetAiProviderByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "AI Provider not found")
		return
	}

	var encryptedKey string = provider.ApiKeySecure
	if req.APIKey != "" {
		if !strings.HasPrefix(req.APIKey, "****") {
			enc, err := cryptoPkg.Encrypt(req.APIKey, getAESKey(h.encryptionKey))
			if err != nil {
				response.InternalServerError(c, "Failed to encrypt new API key")
				return
			}
			encryptedKey = enc
		}
	}

	tx, err := h.db.Begin(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to start database transaction")
		return
	}
	defer tx.Rollback(c.Request.Context())

	txQueries := h.queries.WithTx(tx)

	if req.IsDefault && !provider.IsDefault {
		err = txQueries.SetAllAiProvidersNotDefault(c.Request.Context())
		if err != nil {
			response.InternalServerError(c, "Failed to clear default flag on AI providers")
			return
		}
	}

	var costNumeric pgtype.Numeric
	costNumeric.Scan(fmt.Sprintf("%f", req.CostPerCredit))

	_, err = txQueries.UpdateAiProvider(c.Request.Context(), database.UpdateAiProviderParams{
		ID:                  id,
		Name:                req.Name,
		ApiKeySecure:        encryptedKey,
		BaseUrl:             pgtype.Text{String: req.BaseURL, Valid: req.BaseURL != ""},
		DefaultModel:        req.DefaultModel,
		TextModel:           req.TextModel,
		VisionModel:         req.VisionModel,
		AudioModel:          req.AudioModel,
		EmbeddingModel:      req.EmbeddingModel,
		EmbeddingDimensions: req.EmbeddingDimensions,
		IsActive:            req.IsActive,
		Priority:            req.Priority,
		IsDefault:           req.IsDefault,
		LimitPerMinute:      req.LimitPerMinute,
		LimitPerDay:         req.LimitPerDay,
		CostPerCredit:       costNumeric,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to update AI Provider: "+err.Error())
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		response.InternalServerError(c, "Failed to commit database changes")
		return
	}

	// Audit Log
	var userID pgtype.UUID
	userIDStr, exists := c.Get("user_id")
	if exists {
		_ = userID.Scan(userIDStr)
	}
	meta, _ := json.Marshal(map[string]string{"slug": provider.Slug})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: userID,
		Action:      "AI_PROVIDER_UPDATED",
		EntityType:  "ai_providers",
		EntityID:    id,
		Metadata:    meta,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "AI Provider updated successfully", nil)
}

func (h *Handler) TestProviderConnection(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid AI Provider ID format", nil)
		return
	}

	provider, err := h.queries.GetAiProviderByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "AI Provider not found")
		return
	}

	key := getAESKey(h.encryptionKey)
	var decryptedApiKey string
	if provider.ApiKeySecure != "" {
		dec, err := cryptoPkg.Decrypt(provider.ApiKeySecure, key)
		if err != nil {
			response.InternalServerError(c, "Failed to decrypt API key: "+err.Error())
			return
		}
		decryptedApiKey = dec
	}

	var prov domain.AIProvider
	switch provider.Slug {
	case "openai":
		prov = openai.NewProvider(decryptedApiKey, provider.BaseUrl.String)
	case "gemini":
		prov = gemini.NewProvider(decryptedApiKey, provider.BaseUrl.String)
	case "grok":
		prov = grok.NewProvider(decryptedApiKey, provider.BaseUrl.String)
	case "anthropic":
		prov = anthropic.NewProvider(decryptedApiKey, provider.BaseUrl.String)
	default:
		response.BadRequest(c, "Unsupported connection test for provider slug: "+provider.Slug, nil)
		return
	}

	ok, msg, err := prov.TestConnection(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Technical failure testing connection: "+err.Error())
		return
	}

	// Audit Log
	var userID pgtype.UUID
	userIDStr, exists := c.Get("user_id")
	if exists {
		_ = userID.Scan(userIDStr)
	}
	meta, _ := json.Marshal(map[string]interface{}{
		"slug":    provider.Slug,
		"success": ok,
		"message": msg,
	})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: userID,
		Action:      "AI_PROVIDER_TESTED",
		EntityType:  "ai_providers",
		EntityID:    id,
		Metadata:    meta,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "Connection test results", gin.H{
		"ok":      ok,
		"message": msg,
	})
}

// Helper
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	var str string
	u.Scan(&str)
	return str
}
