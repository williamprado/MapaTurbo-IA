package settings

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/pkg/response"
	"mapaturbo-ia/pkg/validator"
)

type UpdateAiActionPriceRequest struct {
	CreditsCost int32 `json:"credits_cost" validate:"required,min=0"`
	IsActive    bool  `json:"is_active"`
}

func (h *Handler) ListAiActionPrices(c *gin.Context) {
	prices, err := h.queries.ListAiActionPrices(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve AI action prices")
		return
	}

	response.Success(c, http.StatusOK, "AI action prices list", prices)
}

func (h *Handler) UpdateAiActionPrice(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	var req UpdateAiActionPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	price, err := h.queries.UpdateAiActionPrice(c.Request.Context(), database.UpdateAiActionPriceParams{
		ID:          id,
		CreditsCost: req.CreditsCost,
		IsActive:    req.IsActive,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to update AI action price")
		return
	}

	// Create Audit Log
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	meta, _ := json.Marshal(map[string]interface{}{
		"action_key":   price.ActionKey,
		"credits_cost": price.CreditsCost,
	})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: actorUserID,
		Action:      "AI_ACTION_PRICE_UPDATED",
		EntityType:  "ai_action_prices",
		EntityID:    id,
		Metadata:    meta,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "AI action price updated successfully", price)
}
