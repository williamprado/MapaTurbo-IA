package audit

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/pkg/response"
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

func (h *Handler) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	logs, err := h.queries.ListAuditLogsDetailed(c.Request.Context(), database.ListAuditLogsDetailedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve audit logs")
		return
	}

	count, err := h.queries.CountAuditLogs(c.Request.Context())
	if err != nil {
		count = 0
	}

	response.Success(c, http.StatusOK, "Audit logs list", gin.H{
		"logs":  logs,
		"total": count,
	})
}
