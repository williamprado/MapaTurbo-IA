package mindmaps

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"mapaturbo-ia/internal/ai/domain"
	"mapaturbo-ia/internal/ai/providers/openai"
	"mapaturbo-ia/internal/database"
	cryptoPkg "mapaturbo-ia/pkg/crypto"
	"mapaturbo-ia/pkg/logger"
	"go.uber.org/zap"
)

type Worker struct {
	db            *pgxpool.Pool
	queries       *database.Queries
	encryptionKey string
}

func NewWorker(db *pgxpool.Pool, encryptionKey string) *Worker {
	return &Worker{
		db:            db,
		queries:       database.New(db),
		encryptionKey: encryptionKey,
	}
}

func getAESKey(rawKey string) []byte {
	h := sha256.Sum256([]byte(rawKey))
	return h[:]
}

func (w *Worker) ProcessGenerationTask(ctx context.Context, t *asynq.Task) error {
	var payload map[string]string
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Log.Error("Failed to parse generation task payload", zap.Error(err))
		return nil
	}

	jobIDStr, exists := payload["id"]
	if !exists || jobIDStr == "" {
		logger.Log.Error("Missing job ID in task payload")
		return nil
	}

	var jobID pgtype.UUID
	if err := jobID.Scan(jobIDStr); err != nil {
		logger.Log.Error("Invalid job ID UUID format", zap.Error(err))
		return nil
	}

	// 1. Load job details
	job, err := w.queries.GetGenerationJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Log.Error("Generation job not found in DB", zap.String("id", jobIDStr))
			return nil
		}
		return err
	}

	// Idempotency: skip if completed
	if job.Status == "COMPLETED" {
		logger.Log.Info("Job already completed (idempotent)", zap.String("id", jobIDStr))
		return nil
	}

	// 2. Set status to PROCESSING
	_, _ = w.queries.UpdateGenerationJobStatus(ctx, database.UpdateGenerationJobStatusParams{
		ID:     jobID,
		Status: "PROCESSING",
	})

	// Log Audit - AI Generation Started
	_, _ = w.db.Exec(ctx,
		"INSERT INTO audit_logs (actor_user_id, organization_id, action, entity_type, entity_id) VALUES ($1, $2, 'AI_GENERATION_STARTED', 'generation_jobs', $3)",
		job.UserID, job.OrganizationID, job.ID,
	)

	// 3. Resolve default AI Provider
	dbProvider, err := w.queries.GetDefaultAiProvider(ctx)
	if err != nil {
		w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Nenhum provedor de IA ativo configurado como padrão.")
		return nil
	}

	// Decrypt API key
	key := getAESKey(w.encryptionKey)
	var decryptedApiKey string
	if dbProvider.ApiKeySecure != "" {
		dec, err := cryptoPkg.Decrypt(dbProvider.ApiKeySecure, key)
		if err != nil {
			w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Falha técnica ao descriptografar chave de API do provedor.")
			return nil
		}
		decryptedApiKey = dec
	}

	if decryptedApiKey == "" {
		w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Chave de API do provedor padrão vazia ou inválida.")
		return nil
	}

	// 4. Parse request parameters
	var genTitle string
	var genOptions GenerateOptions
	var promptContent string
	var sourceType string
	var sourceUploadID pgtype.UUID

	var genReq GenerateRequest

	if job.Type == "GENERATE_MAP_PDF" {
		type PdfGenerationInput struct {
			UploadID string          `json:"uploadId"`
			Query    string          `json:"query"`
			Options  GenerateOptions `json:"options"`
		}
		var pdfInput PdfGenerationInput
		if err := json.Unmarshal(job.Input, &pdfInput); err != nil {
			w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Erro ao analisar parâmetros de PDF do job.")
			return nil
		}
		genOptions = pdfInput.Options
		sourceType = "PDF"

		var upUUID pgtype.UUID
		if err := upUUID.Scan(pdfInput.UploadID); err != nil {
			w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "ID do upload associado inválido.")
			return nil
		}
		sourceUploadID = upUUID

		// Check document source status
		docSrc, err := w.queries.GetDocumentSourceByUpload(ctx, upUUID)
		if err != nil {
			w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Fonte de documento RAG não encontrada.")
			return nil
		}
		if docSrc.Status != "READY" {
			w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "O documento PDF ainda está sendo processado.")
			return nil
		}

		genTitle = docSrc.Title

		var chunksContext []string
		openaiProv := openai.NewProvider(decryptedApiKey, dbProvider.BaseUrl.String)

		if pdfInput.Query != "" {
			// Search similar chunks using PGVector
			emb, err := openaiProv.GetEmbedding(ctx, pdfInput.Query)
			if err != nil {
				w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Falha ao gerar embedding para consulta: "+err.Error())
				return nil
			}
			similarChunks, err := w.queries.SearchSimilarChunks(ctx, database.SearchSimilarChunksParams{
				QueryEmbedding: pgvector.NewVector(emb),
				OrganizationID: job.OrganizationID,
				UploadID:       upUUID,
				Limit:          10,
			})
			if err != nil {
				w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Falha ao buscar trechos relevantes no banco vetorial.")
				return nil
			}

			for _, c := range similarChunks {
				chunksContext = append(chunksContext, c.Content)
			}
		} else {
			// Retrieve first 12 chunks
			allChunks, err := w.queries.GetDocumentChunksBySource(ctx, database.GetDocumentChunksBySourceParams{
				DocumentSourceID: docSrc.ID,
				Limit:            12,
			})
			if err != nil {
				w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Falha ao ler blocos do documento.")
				return nil
			}

			for _, c := range allChunks {
				chunksContext = append(chunksContext, c.Content)
			}
		}

		if len(chunksContext) == 0 {
			w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "O documento PDF está vazio ou não possui blocos vetoriais válidos.")
			return nil
		}

		contextText := strings.Join(chunksContext, "\n---\n")
		if pdfInput.Query != "" {
			promptContent = fmt.Sprintf("Você é um especialista em estruturação de mapas mentais. Analise o seguinte conteúdo extraído de um documento PDF filtrado pelo tema '%s':\n\n%s\n\nCrie um mapa mental focado no tema '%s' usando as informações fornecidas.", pdfInput.Query, contextText, pdfInput.Query)
		} else {
			promptContent = fmt.Sprintf("Você é um especialista em estruturação de mapas mentais. Analise o seguinte conteúdo extraído de um documento PDF:\n\n%s\n\nCrie um mapa mental focado no tema central e nos tópicos principais abordados nesse texto.", contextText)
		}
	} else {
		if err := json.Unmarshal(job.Input, &genReq); err != nil {
			w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Erro ao analisar parâmetros de entrada do job.")
			return nil
		}
		genTitle = genReq.Title
		genOptions = genReq.Options
		sourceType = genReq.Type
		promptContent = genReq.Content
	}

	// 5. Invoke AI generation method
	// In Phase 3A, we support OpenAI Chat completions. Gemini/Grok/Anthropic are placeholders
	var aiProv domain.AIProvider
	if dbProvider.Slug == "openai" {
		aiProv = openai.NewProvider(decryptedApiKey, dbProvider.BaseUrl.String)
	} else {
		w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, fmt.Sprintf("Provedor '%s' não suporta geração real na Fase 3A.", dbProvider.Slug))
		return nil
	}

	input := domain.GenerateMindMapInput{
		Type:     sourceType,
		Title:    genTitle,
		Content:  promptContent,
		Depth:    genOptions.Depth,
		Language: genOptions.Language,
		Style:    genOptions.Style,
	}

	logger.Log.Info("Starting AI generation with provider", 
		zap.String("job_id", jobIDStr), 
		zap.String("provider", dbProvider.Slug), 
		zap.String("model", dbProvider.DefaultModel),
	)

	startTime := time.Now()
	aiResult, err := aiProv.GenerateMindMap(ctx, input)
	duration := time.Since(startTime)

	if err != nil {
		w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Falha na chamada da API de IA: "+err.Error())
		return nil
	}

	// 6. Rigid JSON validation
	valErr := validateMindMapJSON(aiResult, genReq.Options.Depth)
	if valErr != nil {
		logger.Log.Warn("AI generated JSON validation failed. Initiating single correction attempt...", zap.Error(valErr))
		// Single correction attempt
		aiResult, err = w.retryWithCorrection(ctx, aiProv, input, valErr.Error(), aiResult.RawPayload)
		if err != nil {
			w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Falha na tentativa de correção automática da IA: "+err.Error())
			return nil
		}
		// Validate again
		valErr = validateMindMapJSON(aiResult, genReq.Options.Depth)
		if valErr != nil {
			w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "JSON gerado pela IA inválido após correção: "+valErr.Error())
			return nil
		}
	}

	// 7. DB Transaction with ROW LOCK for concurrent-safe credit balance subtraction
	tx, err := w.db.Begin(ctx)
	if err != nil {
		w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Falha ao iniciar transação de faturamento.")
		return nil
	}
	defer tx.Rollback(ctx)

	// SELECT balance FOR UPDATE
	var balance int32
	err = tx.QueryRow(ctx,
		"SELECT balance FROM ai_credit_balances WHERE organization_id = $1 FOR UPDATE",
		job.OrganizationID,
	).Scan(&balance)
	if err != nil {
		w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Erro ao recuperar saldo da organização sob bloqueio.")
		return nil
	}

	if balance < job.CreditsCost {
		w.markJobFailed(ctx, jobID, job.UserID, job.OrganizationID, "Saldo de créditos insuficiente no encerramento da geração.")
		return nil
	}

	// Check if already debited (idempotency check against duplicate retry runs)
	var txExists bool
	err = tx.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM ai_credit_transactions WHERE metadata->>'generationJobId' = $1)",
		jobIDStr,
	).Scan(&txExists)
	if err == nil && txExists {
		logger.Log.Info("Credits already debited for this job (idempotent skip)", zap.String("job_id", jobIDStr))
		_ = tx.Commit(ctx)
		return nil
	}

	// A. Create Mind Map record
	jsonBytes, _ := json.Marshal(aiResult)
	qtx := w.queries.WithTx(tx)
	mindMap, err := qtx.CreateMindMap(ctx, database.CreateMindMapParams{
		OrganizationID: job.OrganizationID,
		UserID:         job.UserID,
		Title:          aiResult.Title,
		SourceType:     sourceType,
		SourceUploadID: sourceUploadID,
		Status:         "READY",
		JsonData:       jsonBytes,
		IsPublic:       false,
	})
	if err != nil {
		return err
	}
	mindMapID := mindMap.ID

	// B. Subtract credits balance
	_, err = tx.Exec(ctx,
		"UPDATE ai_credit_balances SET balance = balance - $2, updated_at = NOW() WHERE organization_id = $1",
		job.OrganizationID, job.CreditsCost,
	)
	if err != nil {
		return err
	}

	// C. Create Credit Transaction
	metaBytes, _ := json.Marshal(map[string]string{
		"generationJobId": jobIDStr,
		"mindMapId":       uuidToString(mindMapID),
		"action":          job.Type,
	})
	_, err = tx.Exec(ctx,
		"INSERT INTO ai_credit_transactions (organization_id, amount, type, description, metadata) VALUES ($1, $2, 'SUB', $3, $4)",
		job.OrganizationID, job.CreditsCost, "Débito por geração de mapa mental com IA", metaBytes,
	)
	if err != nil {
		return err
	}

	// D. Complete Generation Job
	resultMeta, _ := json.Marshal(map[string]interface{}{
		"provider":       dbProvider.Slug,
		"model":          dbProvider.DefaultModel,
		"duration_ms":    duration.Milliseconds(),
		"mindMapId":      uuidToString(mindMapID),
	})
	_, err = tx.Exec(ctx,
		"UPDATE generation_jobs SET status = 'COMPLETED', result = $2, mind_map_id = $3, finished_at = NOW() WHERE id = $1",
		jobID, resultMeta, mindMapID,
	)
	if err != nil {
		return err
	}

	// E. Audit logs inside same transaction
	// Audit - Mind Map Created
	_, _ = tx.Exec(ctx,
		"INSERT INTO audit_logs (actor_user_id, organization_id, action, entity_type, entity_id, metadata) VALUES ($1, $2, 'MIND_MAP_CREATED', 'mind_maps', $3, $4)",
		job.UserID, job.OrganizationID, mindMapID, metaBytes,
	)

	// Audit - AI Credits Debited
	_, _ = tx.Exec(ctx,
		"INSERT INTO audit_logs (actor_user_id, organization_id, action, entity_type, entity_id, metadata) VALUES ($1, $2, 'AI_CREDITS_DEBITED', 'ai_credit_balances', $3, $4)",
		job.UserID, job.OrganizationID, job.OrganizationID, metaBytes,
	)

	// Audit - AI Generation Completed
	_, _ = tx.Exec(ctx,
		"INSERT INTO audit_logs (actor_user_id, organization_id, action, entity_type, entity_id, metadata) VALUES ($1, $2, 'AI_GENERATION_COMPLETED', 'generation_jobs', $3, $4)",
		job.UserID, job.OrganizationID, job.ID, metaBytes,
	)

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	logger.Log.Info("AI Generation job completed successfully", zap.String("job_id", jobIDStr), zap.Duration("duration", duration))
	return nil
}

func (w *Worker) markJobFailed(ctx context.Context, jobID pgtype.UUID, userID pgtype.UUID, orgID pgtype.UUID, errMsg string) {
	_, err := w.queries.UpdateGenerationJobError(ctx, database.UpdateGenerationJobErrorParams{
		ID:    jobID,
		Error: pgtype.Text{String: errMsg, Valid: true},
	})
	if err != nil {
		logger.Log.Error("Failed to update job status to FAILED", zap.Error(err))
	}

	// Audit Log
	meta, _ := json.Marshal(map[string]string{"error": errMsg})
	_, _ = w.db.Exec(ctx,
		"INSERT INTO audit_logs (actor_user_id, organization_id, action, entity_type, entity_id, metadata) VALUES ($1, $2, 'AI_GENERATION_FAILED', 'generation_jobs', $3, $4)",
		userID, orgID, jobID, meta,
	)
}

func (w *Worker) retryWithCorrection(ctx context.Context, prov domain.AIProvider, input domain.GenerateMindMapInput, valError string, rawJSON string) (*domain.MindMapAIResult, error) {
	// Request correction of the faulty JSON format
	correctionPrompt := fmt.Sprintf("O JSON gerado anteriormente continha erros de validação: %s. Aqui está o JSON incorreto: %s. Por favor, re-gere e corrija o JSON para corresponder exatamente ao formato especificado.", valError, rawJSON)
	
	// Create correction input (we temporary override content with the correction request instructions)
	correctedInput := input
	correctedInput.Content = correctionPrompt

	return prov.GenerateMindMap(ctx, correctedInput)
}

func validateMindMapJSON(res *domain.MindMapAIResult, maxDepth int) error {
	if res.Title == "" {
		return errors.New("title is required")
	}
	if res.CentralTopic == "" {
		return errors.New("centralTopic is required")
	}
	if len(res.Nodes) == 0 {
		return errors.New("nodes array is empty")
	}
	if len(res.Nodes) > 80 {
		return fmt.Errorf("nodes limit exceeded: got %d nodes, max is 80", len(res.Nodes))
	}

	// Validate node root and fields
	nodeMap := make(map[string]domain.AINode)
	rootCount := 0
	var rootNode domain.AINode

	for _, n := range res.Nodes {
		if n.ID == "" {
			return errors.New("found node with empty id")
		}
		if len(n.Title) > 120 {
			return fmt.Errorf("node title too long: %s", n.Title)
		}
		if len(n.Content) > 1000 {
			return fmt.Errorf("node content too long: %s", n.Title)
		}
		if n.Level > maxDepth {
			return fmt.Errorf("node level exceeds max depth: level=%d, max=%d", n.Level, maxDepth)
		}

		if _, exists := nodeMap[n.ID]; exists {
			return fmt.Errorf("duplicate node ID: %s", n.ID)
		}
		nodeMap[n.ID] = n

		if n.ParentID == nil || *n.ParentID == "" {
			rootCount++
			rootNode = n
		}
	}

	if rootCount != 1 {
		return fmt.Errorf("must contain exactly one root node, found %d", rootCount)
	}

	if rootNode.ID != "root" {
		return fmt.Errorf("root node ID must be 'root', got '%s'", rootNode.ID)
	}

	if rootNode.Level != 0 {
		return fmt.Errorf("root node level must be 0, got %d", rootNode.Level)
	}

	// Validate parent connections
	for _, n := range res.Nodes {
		if n.ID == "root" {
			continue
		}
		if n.ParentID == nil || *n.ParentID == "" {
			return fmt.Errorf("non-root node %s has empty parentId", n.ID)
		}
		parent, exists := nodeMap[*n.ParentID]
		if !exists {
			return fmt.Errorf("node %s references non-existing parentId '%s'", n.ID, *n.ParentID)
		}
		if n.Level != parent.Level+1 {
			return fmt.Errorf("node %s level (%d) does not match parent's level (%d) + 1", n.ID, n.Level, parent.Level)
		}
	}

	// Validate edge connections
	for _, e := range res.Edges {
		if _, exists := nodeMap[e.Source]; !exists {
			return fmt.Errorf("edge references non-existing source node: %s", e.Source)
		}
		if _, exists := nodeMap[e.Target]; !exists {
			return fmt.Errorf("edge references non-existing target node: %s", e.Target)
		}
	}

	return nil
}
