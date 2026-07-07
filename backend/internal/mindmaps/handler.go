package mindmaps

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/internal/plans"
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

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Viewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

type AINode struct {
	ID       string    `json:"id"`
	ParentID *string   `json:"parentId"`
	Title    string    `json:"title"`
	Content  string    `json:"content"`
	Level    int       `json:"level"`
	Order    int       `json:"order"`
	Position *Position `json:"position,omitempty"`
}

type AIEdge struct {
	ID     string `json:"id,omitempty"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type MindMapData struct {
	Title        string    `json:"title"`
	CentralTopic string    `json:"centralTopic"`
	Summary      string    `json:"summary"`
	Nodes        []AINode  `json:"nodes"`
	Edges        []AIEdge  `json:"edges"`
	Viewport     *Viewport `json:"viewport,omitempty"`
}

type UpdateMindMapRequest struct {
	Title    string           `json:"title"`
	IsPublic *bool            `json:"isPublic"`
	JsonData *json.RawMessage `json:"jsonData"`
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

	// Validate Plan limits and Feature gates
	limitSvc := plans.NewLimitService(h.queries)
	featureKey := "generateTopic"
	if req.Type == "TEXT" {
		featureKey = "generateText"
	}
	allowedFeature, err := limitSvc.CanUseFeature(c.Request.Context(), orgID, featureKey)
	if err != nil {
		response.InternalServerError(c, "Erro ao verificar limites do plano: "+err.Error())
		return
	}
	if !allowedFeature {
		limitSvc.LogFeatureBlocked(c.Request.Context(), userID, orgID, featureKey)
		response.Forbidden(c, "Seu plano atual não permite geração por "+req.Type+". Faça um upgrade.")
		return
	}

	canCreate, current, maxMaps, err := limitSvc.CanCreateMindMap(c.Request.Context(), orgID)
	if err != nil {
		response.InternalServerError(c, "Erro ao verificar limites de mapas: "+err.Error())
		return
	}
	if !canCreate {
		limitSvc.LogPlanLimitReached(c.Request.Context(), userID, orgID, "max_maps", maxMaps, current)
		response.Forbidden(c, fmt.Sprintf("Você atingiu o limite de mapas do seu plano (%d/%d mapas). Faça um upgrade.", current, maxMaps))
		return
	}

	// 2. Retrieve action price & balance check
	actionKey := "GENERATE_MAP_" + req.Type
	var creditsCost int32 = 10 // fallback default
	err = h.db.QueryRow(c.Request.Context(),
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

type GenerateFromUploadRequest struct {
	UploadID string          `json:"uploadId" validate:"required,uuid"`
	Query    string          `json:"query"`
	Options  GenerateOptions `json:"options"`
}

func (h *Handler) GenerateFromUpload(c *gin.Context) {
	var req GenerateFromUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
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

	// Validate Plan limits and Feature gates
	limitSvc := plans.NewLimitService(h.queries)
	allowedFeature, err := limitSvc.CanUseFeature(c.Request.Context(), orgID, "generatePdf")
	if err != nil {
		response.InternalServerError(c, "Erro ao verificar limites do plano: "+err.Error())
		return
	}
	if !allowedFeature {
		limitSvc.LogFeatureBlocked(c.Request.Context(), userID, orgID, "generatePdf")
		response.Forbidden(c, "Seu plano atual não permite geração a partir de PDF (RAG). Faça um upgrade.")
		return
	}

	canCreate, current, maxMaps, err := limitSvc.CanCreateMindMap(c.Request.Context(), orgID)
	if err != nil {
		response.InternalServerError(c, "Erro ao verificar limites de mapas: "+err.Error())
		return
	}
	if !canCreate {
		limitSvc.LogPlanLimitReached(c.Request.Context(), userID, orgID, "max_maps", maxMaps, current)
		response.Forbidden(c, fmt.Sprintf("Você atingiu o limite de mapas do seu plano (%d/%d mapas). Faça um upgrade.", current, maxMaps))
		return
	}

	// 1. Fetch upload details & check status and tenant isolation
	var upUUID pgtype.UUID
	_ = upUUID.Scan(req.UploadID)

	upload, err := h.queries.GetUploadByID(c.Request.Context(), upUUID)
	if err != nil {
		response.NotFound(c, "Arquivo de upload não encontrado.")
		return
	}

	if uuidToString(upload.OrganizationID) != uuidToString(orgID) {
		response.Forbidden(c, "Você não possui permissão para usar este arquivo.")
		return
	}

	if upload.Status != "PROCESSED" {
		response.BadRequest(c, "O arquivo PDF ainda está sendo processado. Aguarde a conclusão.", nil)
		return
	}

	// 2. Retrieve action price & balance check
	actionKey := "GENERATE_MAP_PDF"
	var creditsCost int32 = 15 // PDF cost fallback
	err = h.db.QueryRow(c.Request.Context(),
		"SELECT credits_cost FROM ai_action_prices WHERE action_key = $1 AND is_active = true LIMIT 1",
		actionKey,
	).Scan(&creditsCost)
	if err != nil {
		creditsCost = 15
	}

	var currentBalance int32 = 0
	err = h.db.QueryRow(c.Request.Context(),
		"SELECT balance FROM ai_credit_balances WHERE organization_id = $1 LIMIT 1",
		orgID,
	).Scan(&currentBalance)
	if err != nil {
		response.BadRequest(c, "Organização não possui saldo de créditos inicializado.", nil)
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
		StartedAt:      pgtype.Timestamptz{Valid: false},
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

	response.Success(c, http.StatusCreated, "Geração de mapa mental por PDF iniciada", gin.H{
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

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	statusQuery := c.Query("status")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var statusNull pgtype.Text
	if statusQuery != "" {
		statusNull = pgtype.Text{String: statusQuery, Valid: true}
	}

	jobs, err := h.queries.ListGenerationJobsByOrganizationPaginated(c.Request.Context(), database.ListGenerationJobsByOrganizationPaginatedParams{
		OrganizationID: orgID,
		Status:         statusNull,
		Limit:          int32(limit),
		Offset:         int32(offset),
	})
	if err != nil {
		response.InternalServerError(c, "Erro ao carregar jobs do banco: "+err.Error())
		return
	}

	total, err := h.queries.CountGenerationJobsByOrganization(c.Request.Context(), database.CountGenerationJobsByOrganizationParams{
		OrganizationID: orgID,
		Status:         statusNull,
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
			Action:         "GENERATION_HISTORY_VIEWED",
			EntityType:     "generation_jobs",
			EntityID:       orgID,
			Metadata:       meta,
			Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
			UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
		})
	}

	response.Success(c, http.StatusOK, "Jobs de geração carregados", gin.H{
		"items": jobs,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
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

	jsonDataParam := m.JsonData
	var auditMeta []byte

	if req.JsonData != nil {
		// 1. Limit raw size (500 KB)
		if len(*req.JsonData) > 500*1024 {
			response.BadRequest(c, "Tamanho do JSON excede o limite máximo permitido de 500KB.", nil)
			return
		}

		// 2. Parse JSON Data
		var data MindMapData
		if err := json.Unmarshal(*req.JsonData, &data); err != nil {
			response.BadRequest(c, "Formato do jsonData inválido: "+err.Error(), nil)
			return
		}

		// 3. Structural checks
		if len(data.Nodes) == 0 {
			response.BadRequest(c, "O mapa mental deve conter pelo menos um nó.", nil)
			return
		}
		if len(data.Nodes) > 150 {
			response.BadRequest(c, "O mapa mental não pode conter mais de 150 nós.", nil)
			return
		}

		// Validate root
		var rootNode *AINode
		rootCount := 0
		for idx, n := range data.Nodes {
			if n.ID == "root" {
				rootNode = &data.Nodes[idx]
				rootCount++
			}
		}
		if rootCount != 1 {
			response.BadRequest(c, fmt.Sprintf("Deve existir exatamente um nó principal (root), foram encontrados %d.", rootCount), nil)
			return
		}
		if rootNode.ParentID != nil && *rootNode.ParentID != "" {
			response.BadRequest(c, "O nó raiz principal (root) não pode possuir parentId.", nil)
			return
		}

		// Validate other nodes
		idMap := make(map[string]bool)
		for _, n := range data.Nodes {
			if n.ID == "" {
				response.BadRequest(c, "Todos os nós devem conter um identificador (id) válido.", nil)
				return
			}
			if idMap[n.ID] {
				response.BadRequest(c, "Identificador de nó duplicado encontrado: "+n.ID, nil)
				return
			}
			idMap[n.ID] = true

			if n.Title == "" || len(strings.TrimSpace(n.Title)) == 0 {
				response.BadRequest(c, "O título do nó '"+n.ID+"' não pode estar vazio.", nil)
				return
			}
			if len(n.Title) > 150 {
				response.BadRequest(c, "O título do nó '"+n.Title+"' excede o limite de 150 caracteres.", nil)
				return
			}
			if len(n.Content) > 2000 {
				response.BadRequest(c, "O conteúdo do nó '"+n.Title+"' excede o limite de 2000 caracteres.", nil)
				return
			}

			// Non-root parent validation
			if n.ID != "root" {
				if n.ParentID == nil || *n.ParentID == "" {
					response.BadRequest(c, "O nó '"+n.Title+"' deve possuir um parentId.", nil)
					return
				}
				parentExists := false
				for _, parent := range data.Nodes {
					if parent.ID == *n.ParentID {
						parentExists = true
						break
					}
				}
				if !parentExists {
					response.BadRequest(c, "O nó '"+n.Title+"' aponta para um nó pai inexistente: '"+*n.ParentID+"'.", nil)
					return
				}
			}
		}

		// 4. Edges checks
		for _, e := range data.Edges {
			if !idMap[e.Source] {
				response.BadRequest(c, "A conexão aponta para uma origem inexistente: '"+e.Source+"'.", nil)
				return
			}
			if !idMap[e.Target] {
				response.BadRequest(c, "A conexão aponta para um destino inexistente: '"+e.Target+"'.", nil)
				return
			}
		}

		// 5. Cycle Detection
		for _, n := range data.Nodes {
			visited := make(map[string]bool)
			curr := n.ID
			for curr != "" && curr != "root" {
				if visited[curr] {
					response.BadRequest(c, "Erro estrutural: o mapa mental contém ciclos direcionados.", nil)
					return
				}
				visited[curr] = true

				found := false
				for _, parentNode := range data.Nodes {
					if parentNode.ID == curr {
						if parentNode.ParentID != nil {
							curr = *parentNode.ParentID
						} else {
							curr = ""
						}
						found = true
						break
					}
				}
				if !found {
					break
				}
			}
		}

		jsonDataParam = *req.JsonData
		auditMeta, _ = json.Marshal(map[string]interface{}{
			"nodesCount":      len(data.Nodes),
			"edgesCount":      len(data.Edges),
			"changedByEditor": true,
		})
	} else {
		auditMeta, _ = json.Marshal(map[string]interface{}{
			"title":     titleParam,
			"is_public": isPublicParam,
		})
	}

	updated, err := h.queries.UpdateMindMapData(c.Request.Context(), database.UpdateMindMapDataParams{
		ID:       id,
		Title:    titleParam,
		JsonData: jsonDataParam,
		IsPublic: isPublicParam,
		Status:   m.Status,
	})
	if err != nil {
		response.InternalServerError(c, "Erro ao atualizar dados do mapa.")
		return
	}

	// Audit log
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    userID,
		OrganizationID: orgID,
		Action:         "MIND_MAP_UPDATED",
		EntityType:     "mind_maps",
		EntityID:       id,
		Metadata:       auditMeta,
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

type ExportCheckRequest struct {
	Format string `json:"format" validate:"required,oneof=PNG PDF"`
}

func (h *Handler) ExportCheck(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid Mind Map ID format", nil)
		return
	}

	var req ExportCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
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

	// Fetch mind map
	mindMap, err := h.queries.GetMindMap(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Mapa mental não encontrado.")
		return
	}

	// Multi-tenant check
	if uuidToString(mindMap.OrganizationID) != uuidToString(orgID) {
		response.Forbidden(c, "Você não possui permissão para acessar este mapa.")
		return
	}

	// Verify limit features
	limitSvc := plans.NewLimitService(h.queries)
	featureKey := "exportPng"
	if req.Format == "PDF" {
		featureKey = "exportPdf"
	}

	allowed, err := limitSvc.CanUseFeature(c.Request.Context(), orgID, featureKey)
	if err != nil {
		response.InternalServerError(c, "Erro ao verificar limites do plano: "+err.Error())
		return
	}

	if !allowed {
		limitSvc.LogFeatureBlocked(c.Request.Context(), userID, orgID, featureKey)
		response.Forbidden(c, fmt.Sprintf("Seu plano atual não permite a exportação no formato %s. Faça um upgrade.", req.Format))
		return
	}

	// Write audit log for success
	action := "MIND_MAP_EXPORTED_PNG"
	if req.Format == "PDF" {
		action = "MIND_MAP_EXPORTED_PDF"
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"mindMapId":      uuidToString(id),
		"format":         req.Format,
		"organizationId": uuidToString(orgID),
	})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    userID,
		OrganizationID: orgID,
		Action:         action,
		EntityType:     "mind_maps",
		EntityID:       id,
		Metadata:       meta,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusOK, "Exportação autorizada", gin.H{
		"authorized": true,
	})
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
