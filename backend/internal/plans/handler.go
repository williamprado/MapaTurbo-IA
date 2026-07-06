package plans

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/pkg/response"
	"mapaturbo-ia/pkg/validator"
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

type CreatePlanRequest struct {
	Name            string                 `json:"name" validate:"required"`
	Description     string                 `json:"description"`
	PriceMonthly    float64                `json:"price_monthly"`
	PriceYearly     float64                `json:"price_yearly"`
	Currency        string                 `json:"currency" validate:"required"`
	CreditsMonthly  int32                  `json:"credits_monthly"`
	MaxMaps         int32                  `json:"max_maps"`
	MaxFiles        int32                  `json:"max_files"`
	MaxUsers        int32                  `json:"max_users"`
	MaxStorageBytes int64                  `json:"max_storage_bytes"`
	Features        map[string]interface{} `json:"features"`
	IsPublic        bool                   `json:"is_public"`
	IsActive        bool                   `json:"is_active"`
}

type UpdatePlanRequest struct {
	Name            *string                 `json:"name"`
	Description     *string                 `json:"description"`
	PriceMonthly    *float64                `json:"price_monthly"`
	PriceYearly     *float64                `json:"price_yearly"`
	Currency        *string                 `json:"currency"`
	CreditsMonthly  *int32                  `json:"credits_monthly"`
	MaxMaps         *int32                  `json:"max_maps"`
	MaxFiles        *int32                  `json:"max_files"`
	MaxUsers        *int32                  `json:"max_users"`
	MaxStorageBytes *int64                  `json:"max_storage_bytes"`
	Features        map[string]interface{}  `json:"features"`
	IsPublic        *bool                   `json:"is_public"`
	IsActive        *bool                   `json:"is_active"`
}

func (h *Handler) ListPublic(c *gin.Context) {
	plans, err := h.queries.ListPublicPlans(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve public plans")
		return
	}
	response.Success(c, http.StatusOK, "Public plans list", plans)
}

func (h *Handler) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	plans, err := h.queries.ListPlans(c.Request.Context(), database.ListPlansParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve plans")
		return
	}

	count, err := h.queries.CountPlans(c.Request.Context())
	if err != nil {
		count = 0
	}

	response.Success(c, http.StatusOK, "Plans list", gin.H{
		"plans": plans,
		"total": count,
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	// Features to raw json
	featuresBytes, _ := jsonMarshal(req.Features)

	// Price Monthly Numeric conversion
	priceMonthlyNumeric := floatToNumeric(req.PriceMonthly)
	priceYearlyNumeric := floatToNumeric(req.PriceYearly)

	plan, err := h.queries.CreatePlan(c.Request.Context(), database.CreatePlanParams{
		Name:            req.Name,
		Description:     pgtype.Text{String: req.Description, Valid: req.Description != ""},
		PriceMonthly:    priceMonthlyNumeric,
		PriceYearly:     priceYearlyNumeric,
		Currency:        req.Currency,
		CreditsMonthly:  req.CreditsMonthly,
		MaxMaps:         req.MaxMaps,
		MaxFiles:        req.MaxFiles,
		MaxUsers:        req.MaxUsers,
		MaxStorageBytes: req.MaxStorageBytes,
		Features:        featuresBytes,
		IsPublic:        req.IsPublic,
		IsActive:        req.IsActive,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to create plan")
		return
	}

	// Create Audit Log
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: actorUserID,
		Action:      "PLAN_CREATED",
		EntityType:  "plans",
		EntityID:    plan.ID,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusCreated, "Plan created successfully", plan)
}

func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	plan, err := h.queries.GetPlanByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Plan not found")
		return
	}

	response.Success(c, http.StatusOK, "Plan details", plan)
}

func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input", err.Error())
		return
	}

	existing, err := h.queries.GetPlanByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Plan not found")
		return
	}

	params := database.UpdatePlanParams{
		ID:              id,
		Name:            existing.Name,
		Description:     existing.Description,
		PriceMonthly:    existing.PriceMonthly,
		PriceYearly:     existing.PriceYearly,
		Currency:        existing.Currency,
		CreditsMonthly:  existing.CreditsMonthly,
		MaxMaps:         existing.MaxMaps,
		MaxFiles:        existing.MaxFiles,
		MaxUsers:        existing.MaxUsers,
		MaxStorageBytes: existing.MaxStorageBytes,
		Features:        existing.Features,
		IsPublic:        existing.IsPublic,
		IsActive:        existing.IsActive,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.PriceMonthly != nil {
		params.PriceMonthly = floatToNumeric(*req.PriceMonthly)
	}
	if req.PriceYearly != nil {
		params.PriceYearly = floatToNumeric(*req.PriceYearly)
	}
	if req.Currency != nil {
		params.Currency = *req.Currency
	}
	if req.CreditsMonthly != nil {
		params.CreditsMonthly = *req.CreditsMonthly
	}
	if req.MaxMaps != nil {
		params.MaxMaps = *req.MaxMaps
	}
	if req.MaxFiles != nil {
		params.MaxFiles = *req.MaxFiles
	}
	if req.MaxUsers != nil {
		params.MaxUsers = *req.MaxUsers
	}
	if req.MaxStorageBytes != nil {
		params.MaxStorageBytes = *req.MaxStorageBytes
	}
	if req.Features != nil {
		featuresBytes, _ := jsonMarshal(req.Features)
		params.Features = featuresBytes
	}
	if req.IsPublic != nil {
		params.IsPublic = *req.IsPublic
	}
	if req.IsActive != nil {
		params.IsActive = *req.IsActive
	}

	plan, err := h.queries.UpdatePlan(c.Request.Context(), params)
	if err != nil {
		response.InternalServerError(c, "Failed to update plan")
		return
	}

	// Create Audit Log
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: actorUserID,
		Action:      "PLAN_UPDATED",
		EntityType:  "plans",
		EntityID:    plan.ID,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "Plan updated successfully", plan)
}

// Helpers
func floatToNumeric(f float64) pgtype.Numeric {
	var num pgtype.Numeric
	_ = num.Scan(strconv.FormatFloat(f, 'f', 2, 64))
	return num
}

func jsonMarshal(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}
