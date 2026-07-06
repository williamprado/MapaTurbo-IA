package users

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
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

type UpdateUserRequest struct {
	Name       *string `json:"name"`
	GlobalRole *string `json:"global_role"`
	Status     *string `json:"status"`
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

	users, err := h.queries.ListUsers(c.Request.Context(), database.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve users")
		return
	}

	count, err := h.queries.CountUsers(c.Request.Context())
	if err != nil {
		count = 0
	}

	response.Success(c, http.StatusOK, "Users list", gin.H{
		"users": users,
		"total": count,
	})
}

func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	user, err := h.queries.GetUserByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, http.StatusOK, "User details", gin.H{
		"id":          user.ID,
		"email":       user.Email,
		"name":        user.Name,
		"global_role": user.GlobalRole,
		"status":      user.Status,
		"created_at":  user.CreatedAt,
		"updated_at":  user.UpdatedAt,
	})
}

func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		response.BadRequest(c, "Invalid UUID format", nil)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input", err.Error())
		return
	}

	existing, err := h.queries.GetUserByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	params := database.UpdateUserParams{
		ID:              id,
		Name:            existing.Name,
		PasswordHash:    existing.PasswordHash,
		Status:          existing.Status,
		GlobalRole:      existing.GlobalRole,
		LastLoginAt:     existing.LastLoginAt,
		EmailVerifiedAt: existing.EmailVerifiedAt,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.GlobalRole != nil {
		params.GlobalRole = *req.GlobalRole
	}
	if req.Status != nil {
		params.Status = *req.Status
	}

	user, err := h.queries.UpdateUser(c.Request.Context(), params)
	if err != nil {
		response.InternalServerError(c, "Failed to update user")
		return
	}

	// Create Audit Log
	actorUserIDStr, _ := c.Get("user_id")
	var actorUserID pgtype.UUID
	_ = actorUserID.Scan(actorUserIDStr)

	// Check if status changed
	if req.Status != nil && *req.Status != existing.Status {
		action := "USER_UNBLOCKED"
		if *req.Status == "BLOCKED" {
			action = "USER_BLOCKED"
			// Revoke all refresh tokens
			_ = h.queries.RevokeAllUserRefreshTokens(c.Request.Context(), id)
		}
		_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
			ActorUserID: actorUserID,
			Action:      action,
			EntityType:  "users",
			EntityID:    id,
			Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
			UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
		})
	} else {
		// Log standard profile update
		_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
			ActorUserID: actorUserID,
			Action:      "USER_UPDATED",
			EntityType:  "users",
			EntityID:    id,
			Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
			UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
		})
	}

	response.Success(c, http.StatusOK, "User updated successfully", user)
}
