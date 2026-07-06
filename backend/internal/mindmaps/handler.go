package mindmaps

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/pkg/queue"
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

type GenerateOptions struct {
	Depth    int    `json:"depth" validate:"required,min=2,max=5"`
	Language string `json:"language" validate:"required"`
	Style    string `json:"style"`
}

type GenerateRequest struct {
	Type    string          `json:"type" validate:"required,oneof=TOPIC TEXT"`
	Title   string          `json:"title" validate:"required"`
	Content string          `json:"content" validate:"required"`
	Options GenerateOptions `json:"options"`
}

type UpdateMindMapRequest struct {
	Title    string `json:"title"`
	IsPublic *bool  `json:"isPublic"`
}

func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	// Limits checking
	if req.Type == "TOPIC" && len(req.Content) > 300 {
		response.BadRequest(c, "O conteúdo do tema deve ter no máximo 300 caracteres.", nil)
		return
	}
	if req.Type == "TEXT" && len(req.Content) > 20000 {
		response.BadRequest(c, "O texto colado deve ter no máximo 20.000 caracteres.", nil)
		return
	}

	// 1. Resolve organization context
	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	userIDStr, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var userID pgtype.UUID
	_ = userID.Scan(userIDStr)

	// 2. Retrieve action price & balance check
	actionKey := "GENERATE_MAP_" + req.Type
	var creditsCost int32 = 10 // fallback default
	err := h.db.QueryRow(c.Request.Context(),
		"SELECT credits_cost FROM ai_action_prices WHERE action_key = $1 AND is_active = true LIMIT 1",
		actionKey,
	).Scan(&creditsCost)
	if err != nil {
		// Not active price, default fallback
		creditsCost = 10
	}

	// Check credit balance
	var currentBalance int32 = 0
	err = h.db.QueryRow(c.Request.Context(),
		"SELECT balance FROM ai_credit_balances WHERE organization_id = $1 LIMIT 1",
		orgID,
	).Scan(&currentBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.BadRequest(c, "Organização não possui saldo de créditos inicializado.", nil)
			return
		}
		response.InternalServerError(c, "Erro ao consultar saldo de créditos")
		return
	}

	if currentBalance < creditsCost {
		response.BadRequest(c, "Saldo de créditos insuficiente para realizar a geração.", nil)
		return
	}

	// 3. Create generation job
	inputJSON, _ := json.Marshal(req)
	job, err := h.queries.CreateGenerationJob(c.Request.Context(), database.CreateGenerationJobParams{
		OrganizationID: orgID,
		UserID:         userID,
		Type:           actionKey,
		Status:         "PENDING",
		Input:          inputJSON,
		CreditsCost:    creditsCost,
		StartedAt:      pgtype.Timestamptz{Valid: false}, // set on processing
	})
	if err != nil {
		response.InternalServerError(c, "Erro ao registrar job de geração no banco.")
		return
	}

	// 4. Enqueue task in Asynq background queue
	taskPayload, _ := json.Marshal(gin.H{
		"id": uuidToString(job.ID),
	})
	_, err = queue.EnqueueTask("generate_mindmap", taskPayload)
	if err != nil {
		// Log but don't fail, worker fallback or retry will pick it up
		fmt.Printf("Warning: failed to queue generate_mindmap: %v\n", err)
	}

	// Audit Log
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    userID,
		OrganizationID: orgID,
		Action:         "AI_GENERATION_REQUESTED",
		EntityType:     "generation_jobs",
		EntityID:       job.ID,
		Metadata:       inputJSON,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusCreated, "Geração de mapa mental iniciada", gin.H{
		"jobId":  uuidToString(job.ID),
		"status": job.Status,
	})
}

func (h *Handler) GetJob(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid Job ID format", nil)
		return
	}

	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	job, err := h.queries.GetGenerationJob(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Job de geração não encontrado")
		return
	}

	// Tenant check
	if uuidToString(job.OrganizationID) != uuidToString(orgID) {
		response.Forbidden(c, "Você não tem permissão para acessar este job")
		return
	}

	response.Success(c, http.StatusOK, "Job status", job)
}

func (h *Handler) ListJobs(c *gin.Context) {
	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	jobs, err := h.queries.ListGenerationJobsByOrganization(c.Request.Context(), database.ListGenerationJobsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          50,
		Offset:         0,
	})
	if err != nil {
		response.InternalServerError(c, "Erro ao listar jobs da organização")
		return
	}

	response.Success(c, http.StatusOK, "Jobs list", jobs)
}

func (h *Handler) ListMindMaps(c *gin.Context) {
	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	maps, err := h.queries.ListMindMapsByOrganization(c.Request.Context(), database.ListMindMapsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          50,
		Offset:         0,
	})
	if err != nil {
		response.InternalServerError(c, "Erro ao buscar mapas mentais")
		return
	}

	response.Success(c, http.StatusOK, "Mindmaps list", maps)
}

func (h *Handler) GetMindMap(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid mind map ID format", nil)
		return
	}

	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	m, err := h.queries.GetMindMap(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Mapa mental não encontrado")
		return
	}

	// Tenant check
	if uuidToString(m.OrganizationID) != uuidToString(orgID) {
		response.Forbidden(c, "Você não tem permissão para acessar este mapa")
		return
	}

	response.Success(c, http.StatusOK, "Mindmap details", m)
}

func (h *Handler) UpdateMindMap(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid mind map ID format", nil)
		return
	}

	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	userIDStr, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var userID pgtype.UUID
	_ = userID.Scan(userIDStr)

	var req UpdateMindMapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input JSON", err.Error())
		return
	}

	m, err := h.queries.GetMindMap(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Mapa mental não encontrado")
		return
	}

	// Tenant check
	if uuidToString(m.OrganizationID) != uuidToString(orgID) {
		response.Forbidden(c, "Você não tem permissão para modificar este mapa")
		return
	}

	titleParam := m.Title
	if req.Title != "" {
		titleParam = req.Title
	}

	isPublicParam := m.IsPublic
	if req.IsPublic != nil {
		isPublicParam = *req.IsPublic
	}

	updated, err := h.queries.UpdateMindMapData(c.Request.Context(), database.UpdateMindMapDataParams{
		ID:       id,
		Title:    titleParam,
		JsonData: m.JsonData,
		IsPublic: isPublicParam,
		Status:   m.Status,
	})
	if err != nil {
		response.InternalServerError(c, "Erro ao atualizar dados do mapa.")
		return
	}

	// Audit log
	meta, _ := json.Marshal(map[string]interface{}{"title": updated.Title, "is_public": updated.IsPublic})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    userID,
		OrganizationID: orgID,
		Action:         "MIND_MAP_UPDATED",
		EntityType:     "mind_maps",
		EntityID:       id,
		Metadata:       meta,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "Mapa mental atualizado", updated)
}

func (h *Handler) DeleteMindMap(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid mind map ID format", nil)
		return
	}

	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

	userIDStr, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var userID pgtype.UUID
	_ = userID.Scan(userIDStr)

	m, err := h.queries.GetMindMap(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Mapa mental não encontrado")
		return
	}

	// Tenant check
	if uuidToString(m.OrganizationID) != uuidToString(orgID) {
		response.Forbidden(c, "Você não tem permissão para deletar este mapa")
		return
	}

	err = h.queries.DeleteMindMap(c.Request.Context(), id)
	if err != nil {
		response.InternalServerError(c, "Erro ao deletar mapa do banco.")
		return
	}

	// Audit log
	meta, _ := json.Marshal(map[string]string{"title": m.Title})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    userID,
		OrganizationID: orgID,
		Action:         "MIND_MAP_DELETED",
		EntityType:     "mind_maps",
		EntityID:       id,
		Metadata:       meta,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "Mapa mental deletado com sucesso", nil)
}

// Helpers
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	var str string
	u.Scan(&str)
	return str
}
