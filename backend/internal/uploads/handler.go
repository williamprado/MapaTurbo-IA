package uploads

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/pkg/response"
	"mapaturbo-ia/pkg/storage"
)

type Handler struct {
	db       *pgxpool.Pool
	queries  *database.Queries
	s3Client *storage.S3Client
}

func NewHandler(db *pgxpool.Pool, s3Client *storage.S3Client) *Handler {
	return &Handler{
		db:       db,
		queries:  database.New(db),
		s3Client: s3Client,
	}
}

func (h *Handler) Upload(c *gin.Context) {
	// 1. Get organization and user context
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

	// 2. Parse file from multi-part form
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "File is required", err.Error())
		return
	}

	src, err := file.Open()
	if err != nil {
		response.InternalServerError(c, "Failed to open upload source file")
		return
	}
	defer src.Close()

	// 3. Generate key and upload to MinIO
	randStr := uniqueID()
	storageKey := "orgs/" + uuidToString(orgID) + "/uploads/" + randStr + "-" + file.Filename

	var uploadURL string
	if h.s3Client != nil {
		_, err = h.s3Client.UploadFile(c.Request.Context(), storageKey, src, file.Size, file.Header.Get("Content-Type"))
		if err != nil {
			response.InternalServerError(c, "Failed to store file on storage server: "+err.Error())
			return
		}
		// Generate temporary signed URL (e.g. expires in 2 hours)
		uploadURL, _ = h.s3Client.GetFileURL(c.Request.Context(), storageKey, 2*time.Hour)
	} else {
		// Placeholder URL if s3 client is not set
		uploadURL = "/static/placeholders/" + file.Filename
	}

	// 4. Save metadata to database
	upload, err := h.queries.CreateUpload(c.Request.Context(), database.CreateUploadParams{
		OrganizationID:  orgID,
		UserID:          userID,
		Filename:        file.Filename,
		OriginalName:    file.Filename,
		MimeType:        file.Header.Get("Content-Type"),
		Size:            file.Size,
		StorageProvider: "S3",
		StorageKey:      storageKey,
		Status:          "UPLOADED",
		Metadata:        []byte("{}"),
	})
	if err != nil {
		response.InternalServerError(c, "Failed to record upload in database: "+err.Error())
		return
	}

	// 5. Create Audit Log
	meta, _ := json.Marshal(map[string]interface{}{
		"filename":    file.Filename,
		"size":        file.Size,
		"storage_key": storageKey,
	})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID:    userID,
		OrganizationID: orgID,
		Action:         "UPLOAD_CREATED",
		EntityType:     "uploads",
		EntityID:       upload.ID,
		Metadata:       meta,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	response.Success(c, http.StatusCreated, "File uploaded successfully", gin.H{
		"upload": upload,
		"url":    uploadURL,
	})
}

func (h *Handler) List(c *gin.Context) {
	orgIDVal, exists := c.Get("org_id")
	if !exists {
		response.BadRequest(c, "Organization context required", nil)
		return
	}
	orgID := orgIDVal.(pgtype.UUID)

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

	uploads, err := h.queries.ListUploadsByOrganization(c.Request.Context(), database.ListUploadsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          int32(limit),
		Offset:         int32(offset),
	})
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve uploads list")
		return
	}

	count, err := h.queries.CountUploadsByOrganization(c.Request.Context(), orgID)
	if err != nil {
		count = 0
	}

	response.Success(c, http.StatusOK, "Uploads list", gin.H{
		"uploads": uploads,
		"total":   count,
	})
}

func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	upload, err := h.queries.GetUploadByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Upload not found")
		return
	}

	// Respect organization isolation (unless Super Admin)
	roleVal, exists := c.Get("global_role")
	isSuperAdmin := exists && roleVal == "SUPER_ADMIN"

	if !isSuperAdmin {
		orgIDVal, exists := c.Get("org_id")
		if !exists {
			response.Forbidden(c, "Access denied")
			return
		}
		orgID := orgIDVal.(pgtype.UUID)
		if uuidToString(upload.OrganizationID) != uuidToString(orgID) {
			response.Forbidden(c, "Access denied: file belongs to another organization")
			return
		}
	}

	var downloadURL string
	if h.s3Client != nil {
		downloadURL, _ = h.s3Client.GetFileURL(c.Request.Context(), upload.StorageKey, 2*time.Hour)
	}

	response.Success(c, http.StatusOK, "Upload details", gin.H{
		"upload": upload,
		"url":    downloadURL,
	})
}

// Helpers
func uniqueID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	var str string
	u.Scan(&str)
	return str
}
