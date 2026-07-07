package uploads

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/dslipak/pdf"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
	"mapaturbo-ia/internal/ai/providers/openai"
	"mapaturbo-ia/internal/database"
	cryptoPkg "mapaturbo-ia/pkg/crypto"
	"mapaturbo-ia/pkg/logger"
	"mapaturbo-ia/pkg/storage"
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

func (w *Worker) getAESKey() []byte {
	h := sha256.Sum256([]byte(w.encryptionKey))
	return h[:]
}

func (w *Worker) ProcessPdfUploadTask(ctx context.Context, t *asynq.Task) error {
	var payload map[string]string
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Log.Error("Failed to parse process PDF upload task payload", zap.Error(err))
		return nil
	}

	uploadIDStr, exists := payload["id"]
	if !exists || uploadIDStr == "" {
		logger.Log.Error("Missing upload ID in task payload")
		return nil
	}

	var uploadID pgtype.UUID
	if err := uploadID.Scan(uploadIDStr); err != nil {
		logger.Log.Error("Invalid upload UUID in task payload", zap.Error(err))
		return nil
	}

	// 1. Fetch upload details
	upload, err := w.queries.GetUploadByID(ctx, uploadID)
	if err != nil {
		logger.Log.Error("Failed to retrieve upload from DB", zap.String("id", uploadIDStr), zap.Error(err))
		return nil
	}

	if upload.Status != "UPLOADED" {
		logger.Log.Warn("Upload is already processed or processing", zap.String("id", uploadIDStr), zap.String("status", upload.Status))
		return nil
	}

	// Set status to PROCESSING
	_, _ = w.queries.UpdateUploadStatus(ctx, database.UpdateUploadStatusParams{
		ID:     uploadID,
		Status: "PROCESSING",
	})

	logger.Log.Info("Starting processing of PDF upload", zap.String("upload_id", uploadIDStr), zap.String("filename", upload.Filename))

	// 2. Create document_sources row
	docSource, err := w.queries.CreateDocumentSource(ctx, database.CreateDocumentSourceParams{
		UploadID:       uploadID,
		OrganizationID: upload.OrganizationID,
		Title:          upload.OriginalName,
		Status:         "CHUNKING",
		WordCount:      0,
		ChunkCount:     0,
	})
	if err != nil {
		logger.Log.Error("Failed to create document source", zap.Error(err))
		w.failUpload(ctx, uploadID, "Falha ao inicializar fonte de documento no banco.")
		return nil
	}

	// 3. Download file from MinIO
	if storage.Client == nil {
		logger.Log.Error("MinIO storage client is not initialized")
		w.failDocSource(ctx, docSource.ID, uploadID, "Serviço de armazenamento inativo.")
		return nil
	}

	reader, err := storage.Client.GetObject(ctx, upload.StorageKey)
	if err != nil {
		logger.Log.Error("Failed to download PDF object from storage", zap.Error(err))
		w.failDocSource(ctx, docSource.ID, uploadID, "Falha ao baixar arquivo do storage.")
		return nil
	}
	defer reader.Close()

	// Write to temporary local file
	tempFile, err := os.CreateTemp("", "mapaturbo-*.pdf")
	if err != nil {
		logger.Log.Error("Failed to create temp file", zap.Error(err))
		w.failDocSource(ctx, docSource.ID, uploadID, "Falha interna ao criar arquivo temporário.")
		return nil
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, reader); err != nil {
		logger.Log.Error("Failed to copy PDF download to temp file", zap.Error(err))
		w.failDocSource(ctx, docSource.ID, uploadID, "Erro ao gravar dados temporários do arquivo.")
		return nil
	}

	// 4. Extract text from PDF
	pdfReader, err := pdf.Open(tempFile.Name())
	if err != nil {
		logger.Log.Error("Failed to open PDF local temp file", zap.Error(err))
		w.failDocSource(ctx, docSource.ID, uploadID, "Arquivo PDF corrompido ou formato inválido.")
		return nil
	}

	var buf strings.Builder
	plainTextReader, err := pdfReader.GetPlainText()
	if err != nil {
		logger.Log.Error("Failed to get plain text reader from PDF", zap.Error(err))
		w.failDocSource(ctx, docSource.ID, uploadID, "PDF protegido por senha ou sem permissão de leitura.")
		return nil
	}

	_, _ = io.Copy(&buf, plainTextReader)
	extractedText := buf.String()

	// Handle scanned PDF / No text extracted
	trimmed := strings.TrimSpace(extractedText)
	if len(trimmed) == 0 {
		logger.Log.Warn("Scanned or empty text PDF uploaded", zap.String("upload_id", uploadIDStr))
		w.failDocSource(ctx, docSource.ID, uploadID, "O PDF parece estar em branco ou foi escaneado (sem texto selecionável).")
		return nil
	}

	wordCount := len(strings.Fields(trimmed))

	// 5. Chunking text (750 chars size, 150 overlap)
	chunks := chunkText(trimmed, 750, 150)
	if len(chunks) == 0 {
		w.failDocSource(ctx, docSource.ID, uploadID, "Falha ao gerar blocos de leitura do documento.")
		return nil
	}

	// 6. Get default OpenAI API Key & init OpenAI provider
	aiProv, err := w.queries.GetDefaultAiProvider(ctx)
	if err != nil {
		logger.Log.Error("Failed to get default AI Provider", zap.Error(err))
		w.failDocSource(ctx, docSource.ID, uploadID, "Nenhum conector de IA configurado para gerar embeddings.")
		return nil
	}

	decryptedKey, err := cryptoPkg.Decrypt(aiProv.ApiKeySecure, w.getAESKey())
	if err != nil {
		logger.Log.Error("Failed to decrypt API Key", zap.Error(err))
		w.failDocSource(ctx, docSource.ID, uploadID, "Falha interna ao descriptografar chave do provedor.")
		return nil
	}

	openAIProvider := openai.NewProvider(decryptedKey, aiProv.BaseUrl.String)

	// 7. Embed & Save chunks
	logger.Log.Info("Generating embeddings for PDF chunks", zap.String("upload_id", uploadIDStr), zap.Int("chunks_count", len(chunks)))

	for idx, chunk := range chunks {
		// Generate Embedding vector
		emb, err := openAIProvider.GetEmbedding(ctx, chunk)
		if err != nil {
			logger.Log.Error("OpenAI Embedding generation failed", zap.Int("chunk_index", idx), zap.Error(err))
			w.failDocSource(ctx, docSource.ID, uploadID, "Falha na chamada de geração de embeddings com OpenAI.")
			return nil
		}

		// Approximate token count: chars / 4
		tokenCount := len(chunk) / 4

		_, err = w.queries.CreateDocumentChunk(ctx, database.CreateDocumentChunkParams{
			DocumentSourceID: docSource.ID,
			OrganizationID:   upload.OrganizationID,
			Content:          chunk,
			ChunkIndex:       int32(idx),
			TokenCount:       int32(tokenCount),
			Embedding:        pgvector.NewVector(emb),
			Metadata:         []byte("{}"),
		})
		if err != nil {
			logger.Log.Error("Failed to save chunk to database", zap.Int("chunk_index", idx), zap.Error(err))
			w.failDocSource(ctx, docSource.ID, uploadID, "Falha ao salvar blocos vetoriais no banco.")
			return nil
		}
	}

	// 8. Update document_sources to READY
	_, _ = w.queries.UpdateDocumentSourceStatus(ctx, database.UpdateDocumentSourceStatusParams{
		ID:         docSource.ID,
		Status:     "READY",
		ChunkCount: int32(len(chunks)),
		WordCount:  int32(wordCount),
	})

	// 9. Update upload to PROCESSED
	_, _ = w.queries.UpdateUploadStatus(ctx, database.UpdateUploadStatusParams{
		ID:     uploadID,
		Status: "PROCESSED",
	})

	// 10. Audit Log UPLOAD_PROCESSED
	meta, _ := json.Marshal(map[string]interface{}{
		"upload_id":   uploadIDStr,
		"chunksCount": len(chunks),
		"wordCount":   wordCount,
	})
	_, _ = w.queries.CreateAuditLog(ctx, database.CreateAuditLogParams{
		OrganizationID: upload.OrganizationID,
		Action:         "UPLOAD_PROCESSED",
		EntityType:     "uploads",
		EntityID:       uploadID,
		Metadata:       meta,
	})

	logger.Log.Info("PDF processed and RAG database populated successfully", zap.String("upload_id", uploadIDStr))
	return nil
}

// Helpers for graceful failure
func (w *Worker) failUpload(ctx context.Context, id pgtype.UUID, message string) {
	_, _ = w.queries.UpdateUploadStatus(ctx, database.UpdateUploadStatusParams{
		ID:     id,
		Status: "FAILED",
	})
}

func (w *Worker) failDocSource(ctx context.Context, docSrcID, uploadID pgtype.UUID, message string) {
	_, _ = w.queries.UpdateDocumentSourceStatus(ctx, database.UpdateDocumentSourceStatusParams{
		ID:         docSrcID,
		Status:     "FAILED",
		ChunkCount: 0,
		WordCount:  0,
	})
	w.failUpload(ctx, uploadID, message)

	// Save error details inside metadata update
	meta, _ := json.Marshal(map[string]interface{}{"error": message})
	dbPool, err := w.db.Acquire(ctx)
	if err == nil {
		defer dbPool.Release()
		var uStr string
		uploadID.Scan(&uStr)
		_, _ = dbPool.Exec(ctx, "UPDATE uploads SET metadata = $1 WHERE id = $2", meta, uploadID)
	}
}

func chunkText(text string, chunkSize, overlap int) []string {
	var chunks []string
	runes := []rune(text)
	length := len(runes)

	if length == 0 {
		return chunks
	}

	for i := 0; i < length; {
		end := i + chunkSize
		if end > length {
			end = length
		}

		chunk := string(runes[i:end])
		chunks = append(chunks, chunk)

		if end == length {
			break
		}

		i = end - overlap
		if i < 0 {
			i = 0
		}
		if i >= end {
			i = end
		}
	}

	return chunks
}
