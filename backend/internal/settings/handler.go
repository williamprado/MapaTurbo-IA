package settings

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/pkg/response"
)

type Handler struct {
	db      *pgxpool.Pool
	queries *database.Queries
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{
		db:      db,
		queries: database.New(db),
	}
}

type UpdateSettingRequest struct {
	Value       json.RawMessage `json:"value" validate:"required"`
	Description string          `json:"description"`
	IsPublic    *bool           `json:"is_public"`
}

func (h *Handler) List(c *gin.Context) {
	settings, err := h.queries.ListSystemSettings(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve system settings")
		return
	}

	response.Success(c, http.StatusOK, "System settings list", settings)
}

func (h *Handler) Update(c *gin.Context) {
	key := c.Param("key")

	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input", err.Error())
		return
	}

	// Fetch existing setting first to merge values and check check-exist
	existing, err := h.queries.GetSystemSetting(c.Request.Context(), key)
	description := req.Description
	isPublic := true

	if err == nil {
		if req.Description == "" && existing.Description.Valid {
			description = existing.Description.String
		}
		if req.IsPublic == nil {
			isPublic = existing.IsPublic
		} else {
			isPublic = *req.IsPublic
		}
	} else {
		// If creating a new setting, default isPublic to true if not specified
		if req.IsPublic != nil {
			isPublic = *req.IsPublic
		}
	}

	setting, err := h.queries.UpsertSystemSetting(c.Request.Context(), database.UpsertSystemSettingParams{
		Key:         key,
		Value:       req.Value,
		Description: pgtype.Text{String: description, Valid: description != ""},
		IsPublic:    isPublic,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to update system setting")
		return
	}

	// Create Audit Log
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	meta, _ := json.Marshal(map[string]string{
		"key": key,
	})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: actorUserID,
		Action:      "SYSTEM_SETTING_UPDATED",
		EntityType:  "system_settings",
		Metadata:    meta,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "System setting updated successfully", setting)
}
