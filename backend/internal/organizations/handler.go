package organizations

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/internal/plans"
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

type AddUserRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"required,oneof=ORG_ADMIN USER"`
}

type UpdateUserRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=ORG_ADMIN USER"`
}

func (h *Handler) ListUsers(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	users, err := h.queries.ListOrganizationUsers(c.Request.Context(), id)
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve organization users")
		return
	}

	response.Success(c, http.StatusOK, "Organization users list", users)
}

func (h *Handler) AddUser(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(req.UserID); err != nil {
		response.BadRequest(c, "Invalid target User UUID format", nil)
		return
	}

	// Check if already member
	_, err := h.queries.GetOrganizationUser(c.Request.Context(), database.GetOrganizationUserParams{
		OrganizationID: id,
		UserID:         targetUserID,
	})
	if err == nil {
		response.BadRequest(c, "User is already a member of this organization", nil)
		return
	}

	// Validate limit of users
	limitSvc := plans.NewLimitService(h.queries)
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	canAdd, currentUsers, maxUsers, err := limitSvc.CanAddUser(c.Request.Context(), id)
	if err != nil {
		response.InternalServerError(c, "Erro ao verificar limites do plano: "+err.Error())
		return
	}
	if !canAdd {
		limitSvc.LogPlanLimitReached(c.Request.Context(), actorUserID, id, "max_users", maxUsers, currentUsers)
		response.Forbidden(c, fmt.Sprintf("A organização atingiu o limite de usuários do plano (%d/%d usuários). Faça um upgrade.", currentUsers, maxUsers))
		return
	}

	orgUser, err := h.queries.CreateOrganizationUser(c.Request.Context(), database.CreateOrganizationUserParams{
		OrganizationID: id,
		UserID:         targetUserID,
		Role:           req.Role,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to add user to organization")
		return
	}

	// Create Audit Log

	meta, _ := json.Marshal(map[string]string{
		"role":    req.Role,
		"user_id": req.UserID,
	})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    actorUserID,
		OrganizationID: id,
		Action:         "ORGANIZATION_USER_ADDED",
		EntityType:     "organization_users",
		EntityID:       orgUser.ID,
		Metadata:       meta,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusCreated, "User added to organization successfully", orgUser)
}

func (h *Handler) RemoveUser(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	targetUserIDStr := c.Param("userId")
	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(targetUserIDStr); err != nil {
		response.BadRequest(c, "Invalid target User UUID format", nil)
		return
	}

	// Verify membership
	orgUser, err := h.queries.GetOrganizationUser(c.Request.Context(), database.GetOrganizationUserParams{
		OrganizationID: id,
		UserID:         targetUserID,
	})
	if err != nil {
		response.NotFound(c, "User is not a member of this organization")
		return
	}

	err = h.queries.RemoveOrganizationUser(c.Request.Context(), database.RemoveOrganizationUserParams{
		OrganizationID: id,
		UserID:         targetUserID,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to remove user from organization")
		return
	}

	// Create Audit Log
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	meta, _ := json.Marshal(map[string]string{
		"user_id": targetUserIDStr,
	})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    actorUserID,
		OrganizationID: id,
		Action:         "ORGANIZATION_USER_REMOVED",
		EntityType:     "organization_users",
		EntityID:       orgUser.ID,
		Metadata:       meta,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "User removed from organization successfully", nil)
}

func (h *Handler) UpdateUserRole(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	targetUserIDStr := c.Param("userId")
	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(targetUserIDStr); err != nil {
		response.BadRequest(c, "Invalid target User UUID format", nil)
		return
	}

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	// Verify membership
	_, err := h.queries.GetOrganizationUser(c.Request.Context(), database.GetOrganizationUserParams{
		OrganizationID: id,
		UserID:         targetUserID,
	})
	if err != nil {
		response.NotFound(c, "User is not a member of this organization")
		return
	}

	orgUser, err := h.queries.UpdateOrganizationUserRole(c.Request.Context(), database.UpdateOrganizationUserRoleParams{
		OrganizationID: id,
		UserID:         targetUserID,
		Role:           req.Role,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to update user role")
		return
	}

	// Create Audit Log
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	meta, _ := json.Marshal(map[string]string{
		"new_role": req.Role,
		"user_id":  targetUserIDStr,
	})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    actorUserID,
		OrganizationID: id,
		Action:         "ORGANIZATION_USER_ROLE_UPDATED",
		EntityType:     "organization_users",
		EntityID:       orgUser.ID,
		Metadata:       meta,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "User role updated successfully", orgUser)
}

func (h *Handler) GetBalance(c *gin.Context) {
	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	balance, err := h.queries.GetCreditBalance(c.Request.Context(), orgID)
	if err != nil {
		response.Success(c, http.StatusOK, "Balance details", gin.H{
			"balance": 0,
		})
		return
	}

	response.Success(c, http.StatusOK, "Balance details", gin.H{
		"balance":    balance.Balance,
		"updated_at": balance.UpdatedAt,
	})
}

func (h *Handler) GetCreditsHistory(c *gin.Context) {
	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	typeQuery := c.Query("type")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var typeNull pgtype.Text
	if typeQuery != "" {
		if typeQuery == "CREDIT" {
			typeNull = pgtype.Text{String: "ADD", Valid: true}
		} else if typeQuery == "DEBIT" {
			typeNull = pgtype.Text{String: "SUB", Valid: true}
		} else {
			typeNull = pgtype.Text{String: typeQuery, Valid: true}
		}
	}

	balanceVal := int32(0)
	bal, err := h.queries.GetCreditBalance(c.Request.Context(), orgID)
	if err == nil {
		balanceVal = bal.Balance
	}

	txs, err := h.queries.ListCreditTransactionsByOrganization(c.Request.Context(), database.ListCreditTransactionsByOrganizationParams{
		OrganizationID: orgID,
		Type:           typeNull,
		Limit:          int32(limit),
		Offset:         int32(offset),
	})
	if err != nil {
		txs = []database.AiCreditTransaction{}
	}

	total, err := h.queries.CountCreditTransactionsByOrganization(c.Request.Context(), database.CountCreditTransactionsByOrganizationParams{
		OrganizationID: orgID,
		Type:           typeNull,
	})
	if err != nil {
		total = 0
	}

	if page == 1 {
		userIDStr, _ := c.Get("user_id")
		var userID pgtype.UUID
		_ = userID.Scan(userIDStr)

		meta, _ := json.Marshal(map[string]interface{}{
			"organizationId": uuidToString(orgID),
		})
		_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
			ActorUserID:    userID,
			OrganizationID: orgID,
			Action:         "CREDITS_VIEWED",
			EntityType:     "ai_credit_balances",
			EntityID:       orgID,
			Metadata:       meta,
			Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
			UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
		})
	}

	response.Success(c, http.StatusOK, "Credits history", gin.H{
		"balance": balanceVal,
		"items":   txs,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func (h *Handler) GetDashboard(c *gin.Context) {
	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	limitSvc := plans.NewLimitService(h.queries)
	limits, err := limitSvc.GetLimits(c.Request.Context(), orgID)
	if err != nil {
		limits = plans.DefaultFreeLimits
	}

	mapsCount, err := h.queries.CountMindMapsByOrganization(c.Request.Context(), orgID)
	if err != nil {
		mapsCount = 0
	}

	uploadsCount, err := h.queries.CountUploadsByOrganization(c.Request.Context(), orgID)
	if err != nil {
		uploadsCount = 0
	}

	storageBytes, err := h.queries.SumUploadSizeByOrganization(c.Request.Context(), orgID)
	if err != nil {
		storageBytes = 0
	}

	usersCount, err := h.queries.CountOrganizationUsers(c.Request.Context(), orgID)
	if err != nil {
		usersCount = 0
	}

	balanceVal := int32(0)
	bal, err := h.queries.GetCreditBalance(c.Request.Context(), orgID)
	if err == nil {
		balanceVal = bal.Balance
	}

	var activePlanName = "Gratuito (Trial)"
	var activePlanID = ""
	var subscriptionStatus = "INACTIVE"
	sub, err := h.queries.GetSubscriptionByOrg(c.Request.Context(), orgID)
	if err == nil {
		subscriptionStatus = sub.Status
		plan, planErr := h.queries.GetPlanByID(c.Request.Context(), sub.PlanID)
		if planErr == nil {
			activePlanName = plan.Name
			activePlanID = uuidToString(plan.ID)
		}
	}

	recentMaps, err := h.queries.ListRecentMindMapsByOrganization(c.Request.Context(), database.ListRecentMindMapsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          5,
	})
	if err != nil {
		recentMaps = []database.ListRecentMindMapsByOrganizationRow{}
	}

	recentUploads, err := h.queries.ListUploadsByOrganization(c.Request.Context(), database.ListUploadsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          5,
		Offset:         0,
	})
	if err != nil {
		recentUploads = []database.ListUploadsByOrganizationRow{}
	}

	recentJobs, err := h.queries.ListGenerationJobsByOrganizationPaginated(c.Request.Context(), database.ListGenerationJobsByOrganizationPaginatedParams{
		OrganizationID: orgID,
		Limit:          5,
		Offset:         0,
	})
	if err != nil {
		recentJobs = []database.GenerationJob{}
	}

	response.Success(c, http.StatusOK, "User dashboard summary", gin.H{
		"plan": gin.H{
			"name":     activePlanName,
			"id":       activePlanID,
			"status":   subscriptionStatus,
			"features": limits.Features,
			"limits": gin.H{
				"max_maps":          limits.MaxMaps,
				"max_files":         limits.MaxFiles,
				"max_users":         limits.MaxUsers,
				"max_storage_bytes": limits.MaxStorageBytes,
			},
		},
		"credits": gin.H{
			"balance": balanceVal,
		},
		"usage": gin.H{
			"maps_count":    mapsCount,
			"uploads_count": uploadsCount,
			"storage_bytes": storageBytes,
			"users_count":   usersCount,
		},
		"recent_maps":    recentMaps,
		"recent_uploads": recentUploads,
		"recent_jobs":    recentJobs,
	})
}

func (h *Handler) GetAdminDashboard(c *gin.Context) {
	stats, err := h.queries.GetAdminDashboardStats(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Erro ao carregar estatísticas do admin: "+err.Error())
		return
	}

	var recentIAErrors []database.GenerationJob
	rows, err := h.db.Query(c.Request.Context(),
		"SELECT id, organization_id, user_id, type, status, input, result, error, credits_cost, started_at, finished_at, created_at, mind_map_id FROM generation_jobs WHERE status = 'FAILED' ORDER BY finished_at DESC LIMIT 5",
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var job database.GenerationJob
			if err := rows.Scan(
				&job.ID, &job.OrganizationID, &job.UserID, &job.Type, &job.Status, &job.Input, &job.Result, &job.Error, &job.CreditsCost, &job.StartedAt, &job.FinishedAt, &job.CreatedAt, &job.MindMapID,
			); err == nil {
				recentIAErrors = append(recentIAErrors, job)
			}
		}
	}

	var recentWebhookErrors []database.WebhookEvent
	rowsWeb, err := h.db.Query(c.Request.Context(),
		"SELECT id, provider, event_type, external_id, payload, status, error, processed_at, created_at FROM webhook_events WHERE status = 'FAILED' ORDER BY created_at DESC LIMIT 5",
	)
	if err == nil {
		defer rowsWeb.Close()
		for rowsWeb.Next() {
			var event database.WebhookEvent
			if err := rowsWeb.Scan(
				&event.ID, &event.Provider, &event.EventType, &event.ExternalID, &event.Payload, &event.Status, &event.Error, &event.ProcessedAt, &event.CreatedAt,
			); err == nil {
				recentWebhookErrors = append(recentWebhookErrors, event)
			}
		}
	}

	var paidNum float64 = 0
	if stats.PaidInvoicesAmount.Valid {
		_ = stats.PaidInvoicesAmount.Scan(&paidNum)
	}

	response.Success(c, http.StatusOK, "Admin dashboard stats", gin.H{
		"total_organizations":  stats.TotalOrganizations,
		"active_organizations": stats.ActiveOrganizations,
		"total_users":          stats.TotalUsers,
		"active_users":         stats.ActiveUsers,
		"total_mind_maps":      stats.TotalMindMaps,
		"total_uploads":        stats.TotalUploads,
		"credits_consumed":     stats.CreditsConsumed,
		"active_subscriptions": stats.ActiveSubscriptions,
		"revenue_estimated":    paidNum,
		"recent_ia_errors":     recentIAErrors,
		"recent_webhook_errors": recentWebhookErrors,
	})
}

func (h *Handler) GetOrganizationSummary(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid Organization ID format", nil)
		return
	}

	summary, err := h.queries.GetOrganizationSummary(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Organização não encontrada: "+err.Error())
		return
	}

	var activePlanName = "Gratuito (Trial)"
	var activePlanID = ""
	var subscriptionStatus = "INACTIVE"
	var currentPeriodEnd pgtype.Timestamptz
	sub, err := h.queries.GetSubscriptionByOrg(c.Request.Context(), id)
	if err == nil {
		subscriptionStatus = sub.Status
		currentPeriodEnd = sub.CurrentPeriodEnd
		plan, planErr := h.queries.GetPlanByID(c.Request.Context(), sub.PlanID)
		if planErr == nil {
			activePlanName = plan.Name
			activePlanID = uuidToString(plan.ID)
		}
	}

	users, err := h.queries.ListOrganizationUsers(c.Request.Context(), id)
	if err != nil {
		users = []database.ListOrganizationUsersRow{}
	}

	maps, err := h.queries.ListMindMapsByOrganization(c.Request.Context(), database.ListMindMapsByOrganizationParams{
		OrganizationID: id,
		Limit:          20,
		Offset:         0,
	})
	if err != nil {
		maps = []database.MindMap{}
	}

	uploads, err := h.queries.ListUploadsByOrganization(c.Request.Context(), database.ListUploadsByOrganizationParams{
		OrganizationID: id,
		Limit:          20,
		Offset:         0,
	})
	if err != nil {
		uploads = []database.ListUploadsByOrganizationRow{}
	}

	invoices, err := h.queries.ListInvoicesByOrganization(c.Request.Context(), database.ListInvoicesByOrganizationParams{
		OrganizationID: id,
		Limit:          20,
		Offset:         0,
	})
	if err != nil {
		invoices = []database.ListInvoicesByOrganizationRow{}
	}

	var auditLogs []database.AuditLog
	rows, err := h.db.Query(c.Request.Context(),
		"SELECT id, actor_user_id, organization_id, action, entity_type, entity_id, metadata, ip, user_agent, created_at FROM audit_logs WHERE organization_id = $1 ORDER BY created_at DESC LIMIT 20",
		id,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var log database.AuditLog
			if err := rows.Scan(
				&log.ID, &log.ActorUserID, &log.OrganizationID, &log.Action, &log.EntityType, &log.EntityID, &log.Metadata, &log.Ip, &log.UserAgent, &log.CreatedAt,
			); err == nil {
				auditLogs = append(auditLogs, log)
			}
		}
	}

	response.Success(c, http.StatusOK, "Organization summary details", gin.H{
		"organization": summary,
		"plan": gin.H{
			"name":               activePlanName,
			"id":                 activePlanID,
			"status":             subscriptionStatus,
			"current_period_end": currentPeriodEnd,
		},
		"users":      users,
		"maps":       maps,
		"uploads":    uploads,
		"invoices":   invoices,
		"audit_logs": auditLogs,
	})
}

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		id.Bytes[0], id.Bytes[1], id.Bytes[2], id.Bytes[3],
		id.Bytes[4], id.Bytes[5],
		id.Bytes[6], id.Bytes[7],
		id.Bytes[8], id.Bytes[9],
		id.Bytes[10], id.Bytes[11], id.Bytes[12], id.Bytes[13], id.Bytes[14], id.Bytes[15])
}

