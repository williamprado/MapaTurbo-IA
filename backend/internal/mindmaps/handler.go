package mindmaps

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

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

// Helpers
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	var str string
	u.Scan(&str)
	return str
}
