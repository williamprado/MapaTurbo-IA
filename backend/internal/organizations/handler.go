package organizations

import (
	"net/http"
	"strconv"
	"strings"

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

type CreateOrgRequest struct {
	Name string `json:"name" validate:"required"`
	Slug string `json:"slug" validate:"required"`
}

type UpdateOrgRequest struct {
	Name   *string `json:"name"`
	Slug   *string `json:"slug"`
	Status *string `json:"status"`
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

	orgs, err := h.queries.ListOrganizations(c.Request.Context(), database.ListOrganizationsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve organizations")
		return
	}

	count, err := h.queries.CountOrganizations(c.Request.Context())
	if err != nil {
		count = 0
	}

	response.Success(c, http.StatusOK, "Organizations list", gin.H{
		"organizations": orgs,
		"total":         count,
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	req.Slug = strings.ToLower(req.Slug)

	_, err := h.queries.GetOrganizationBySlug(c.Request.Context(), req.Slug)
	if err == nil {
		response.BadRequest(c, "Slug already in use", nil)
		return
	}

	org, err := h.queries.CreateOrganization(c.Request.Context(), database.CreateOrganizationParams{
		Name:   req.Name,
		Slug:   req.Slug,
		Status: "ACTIVE",
	})
	if err != nil {
		response.InternalServerError(c, "Failed to create organization")
		return
	}

	_, _ = h.queries.InitializeCreditBalance(c.Request.Context(), database.InitializeCreditBalanceParams{
		OrganizationID: org.ID,
		Balance:        1000,
	})

	// Create Audit Log
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    actorUserID,
		OrganizationID: org.ID,
		Action:         "ORGANIZATION_CREATED",
		EntityType:     "organizations",
		EntityID:       org.ID,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusCreated, "Organization created successfully", org)
}

func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	org, err := h.queries.GetOrganizationByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Organization not found")
		return
	}

	response.Success(c, http.StatusOK, "Organization details", org)
}

func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	var req UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input", err.Error())
		return
	}

	existing, err := h.queries.GetOrganizationByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Organization not found")
		return
	}

	params := database.UpdateOrganizationParams{
		ID:     id,
		Name:   existing.Name,
		Slug:   existing.Slug,
		Status: existing.Status,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.Slug != nil {
		params.Slug = strings.ToLower(*req.Slug)
	}
	if req.Status != nil {
		params.Status = *req.Status
	}

	org, err := h.queries.UpdateOrganization(c.Request.Context(), params)
	if err != nil {
		response.InternalServerError(c, "Failed to update organization")
		return
	}

	// Create Audit Log
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    actorUserID,
		OrganizationID: org.ID,
		Action:         "ORGANIZATION_UPDATED",
		EntityType:     "organizations",
		EntityID:       org.ID,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "Organization updated successfully", org)
}
