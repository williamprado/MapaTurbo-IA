package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/internal/database"
	"mapaturbo-ia/pkg/crypto"
	"mapaturbo-ia/pkg/response"
	"mapaturbo-ia/pkg/validator"
)

type Handler struct {
	db        *pgxpool.Pool
	queries   *database.Queries
	jwtSecret string
}

func NewHandler(db *pgxpool.Pool, jwtSecret string) *Handler {
	return &Handler{
		db:        db,
		queries:   database.New(db),
		jwtSecret: jwtSecret,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
	OrgName  string `json:"org_name" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	User         database.CreateUserRow `json:"user"`
}

func generateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	// Clean email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Check if email exists
	_, err := h.queries.GetUserByEmail(c.Request.Context(), req.Email)
	if err == nil {
		response.BadRequest(c, "Email already registered", nil)
		return
	}

	passwordHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		response.InternalServerError(c, "Failed to process security parameters")
		return
	}

	// Execute inside a transaction
	tx, err := h.db.Begin(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to start transaction")
		return
	}
	defer tx.Rollback(c.Request.Context())

	txQueries := h.queries.WithTx(tx)

	// 1. Create User
	user, err := txQueries.CreateUser(c.Request.Context(), database.CreateUserParams{
		Email:        req.Email,
		PasswordHash: passwordHash,
		Name:         req.Name,
		GlobalRole:   "USER",
		Status:       "ACTIVE",
	})
	if err != nil {
		response.InternalServerError(c, "Failed to create user account")
		return
	}

	// Generate Slug for Organization
	orgSlug := strings.ToLower(strings.ReplaceAll(req.OrgName, " ", "-"))
	// Make sure slug has no weird characters
	orgSlug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, orgSlug)

	// Check slug conflict and make it unique if needed
	var count int
	err = h.db.QueryRow(c.Request.Context(), "SELECT COUNT(*) FROM organizations WHERE slug = $1", orgSlug).Scan(&count)
	if err == nil && count > 0 {
		orgSlug = orgSlug + "-" + string(time.Now().Format("05"))
	}

	// 2. Create Organization
	org, err := txQueries.CreateOrganization(c.Request.Context(), database.CreateOrganizationParams{
		Name:   req.OrgName,
		Slug:   orgSlug,
		Status: "ACTIVE",
	})
	if err != nil {
		response.InternalServerError(c, "Failed to create tenant organization")
		return
	}

	// 3. Link User as ORG_ADMIN
	_, err = txQueries.CreateOrganizationUser(c.Request.Context(), database.CreateOrganizationUserParams{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           "ORG_ADMIN",
	})
	if err != nil {
		response.InternalServerError(c, "Failed to link user to organization")
		return
	}

	// 4. Initialize Credit Balance (100 credits for free trial signups)
	_, err = txQueries.InitializeCreditBalance(c.Request.Context(), database.InitializeCreditBalanceParams{
		OrganizationID: org.ID,
		Balance:        100,
	})
	if err != nil {
		response.InternalServerError(c, "Failed to initialize organization balance")
		return
	}

	// 5. Create Audit Log
	meta, _ := json.Marshal(map[string]interface{}{
		"email: ": user.Email,
		"org: ":   org.Slug,
	})
	_, _ = txQueries.CreateAuditLog(ctxOrFallback(c.Request.Context()), database.CreateAuditLogParams{
		ActorUserID:    user.ID,
		OrganizationID: org.ID,
		Action:         "USER_CREATED",
		EntityType:     "users",
		EntityID:       user.ID,
		Metadata:       meta,
		Ip:             pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:      pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	if err := tx.Commit(c.Request.Context()); err != nil {
		response.InternalServerError(c, "Failed to finalize registration")
		return
	}

	// Generate tokens
	accessToken, err := GenerateToken(uuidToString(user.ID), user.Email, user.GlobalRole, h.jwtSecret, 15*time.Minute)
	if err != nil {
		response.InternalServerError(c, "Account created, but token generation failed")
		return
	}

	rawRefreshToken, err := generateRandomToken()
	if err != nil {
		response.InternalServerError(c, "Account created, but refresh token generation failed")
		return
	}

	// Save hashed refresh token to database
	hashedToken := hashToken(rawRefreshToken)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err = h.queries.CreateRefreshToken(c.Request.Context(), database.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hashedToken,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		response.InternalServerError(c, "Failed to initialize session parameters")
		return
	}

	response.Success(c, http.StatusCreated, "User registered successfully", AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		User:         user,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := h.queries.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Unauthorized(c, "Invalid credentials")
			return
		}
		response.InternalServerError(c, "Database query failed")
		return
	}

	if user.Status == "BLOCKED" {
		response.Forbidden(c, "Your account has been blocked")
		return
	}

	if !crypto.CheckPasswordHash(req.Password, user.PasswordHash) {
		// Log failed login audit
		meta, _ := json.Marshal(map[string]string{"email": req.Email})
		_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
			Action:     "LOGIN_FAILED",
			EntityType: "users",
			Metadata:  meta,
			Ip:        pgtype.Text{String: c.ClientIP(), Valid: true},
			UserAgent: pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
		})

		response.Unauthorized(c, "Invalid credentials")
		return
	}

	// Update last_login_at (dedicated query; UpdateUser would blank other
	// string fields since Go sends "" which COALESCE treats as non-NULL).
	now := time.Now()
	_ = h.queries.UpdateLastLogin(c.Request.Context(), database.UpdateLastLoginParams{
		ID:          user.ID,
		LastLoginAt: pgtype.Timestamptz{Time: now, Valid: true},
	})

	// Generate tokens
	accessToken, err := GenerateToken(uuidToString(user.ID), user.Email, user.GlobalRole, h.jwtSecret, 15*time.Minute)
	if err != nil {
		response.InternalServerError(c, "Token generation failed")
		return
	}

	rawRefreshToken, err := generateRandomToken()
	if err != nil {
		response.InternalServerError(c, "Refresh token generation failed")
		return
	}

	// Save hashed refresh token to database
	hashedToken := hashToken(rawRefreshToken)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err = h.queries.CreateRefreshToken(c.Request.Context(), database.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hashedToken,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		response.InternalServerError(c, "Session persistence initialization failed")
		return
	}

	// Log successful login audit
	meta, _ := json.Marshal(map[string]string{"email": user.Email})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: user.ID,
		Action:      "LOGIN_SUCCESS",
		EntityType:  "users",
		EntityID:    user.ID,
		Metadata:    meta,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	// Format user for response
	resUser := database.CreateUserRow{
		ID:         user.ID,
		Email:      user.Email,
		Name:       user.Name,
		GlobalRole: user.GlobalRole,
		Status:     user.Status,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}

	response.Success(c, http.StatusOK, "Login successful", AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		User:         resUser,
	})
}

func (h *Handler) Me(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	var userID pgtype.UUID
	if err := userID.Scan(userIDStr); err != nil {
		response.Unauthorized(c, "User session not found")
		return
	}

	user, err := h.queries.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response.NotFound(c, "User profile not found")
		return
	}

	// Fetch user's organizations
	orgs, err := h.queries.GetUserOrganizations(c.Request.Context(), userID)
	if err != nil {
		orgs = []database.GetUserOrganizationsRow{}
	}

	response.Success(c, http.StatusOK, "Profile retrieved successfully", gin.H{
		"id":            uuidToString(user.ID),
		"email":         user.Email,
		"name":          user.Name,
		"global_role":   user.GlobalRole,
		"status":        user.Status,
		"last_login_at": user.LastLoginAt,
		"organizations": orgs,
	})
}

// Helpers to handle pgtype.UUID conversion cleanly
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	// Scan UUID bytes to string representation
	var str string
	u.Scan(&str)
	return str
}

func ctxOrFallback(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	hashedToken := hashToken(req.RefreshToken)
	dbToken, err := h.queries.GetRefreshToken(c.Request.Context(), hashedToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Unauthorized(c, "Invalid or expired refresh token")
			return
		}
		response.InternalServerError(c, "Database query failed")
		return
	}

	user, err := h.queries.GetUserByID(c.Request.Context(), dbToken.UserID)
	if err != nil {
		response.Unauthorized(c, "Associated user account not found")
		return
	}

	if user.Status == "BLOCKED" {
		response.Forbidden(c, "Your account has been blocked")
		return
	}

	// Token rotation: revoke old token
	_ = h.queries.RevokeRefreshToken(c.Request.Context(), hashedToken)

	// Generate new tokens
	accessToken, err := GenerateToken(uuidToString(user.ID), user.Email, user.GlobalRole, h.jwtSecret, 15*time.Minute)
	if err != nil {
		response.InternalServerError(c, "Failed to generate access token")
		return
	}

	rawRefreshToken, err := generateRandomToken()
	if err != nil {
		response.InternalServerError(c, "Failed to generate refresh token")
		return
	}

	// Save new refresh token hash to database
	newHashedToken := hashToken(rawRefreshToken)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err = h.queries.CreateRefreshToken(c.Request.Context(), database.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: newHashedToken,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		response.InternalServerError(c, "Session persistence registration failed")
		return
	}

	// Create Audit Log
	meta, _ := json.Marshal(map[string]string{"email": user.Email})
	_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
		ActorUserID: user.ID,
		Action:      "REFRESH_TOKEN_CREATED",
		EntityType:  "users",
		EntityID:    user.ID,
		Metadata:    meta,
		Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
		UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
	})

	// Format user for response
	resUser := database.CreateUserRow{
		ID:         user.ID,
		Email:      user.Email,
		Name:       user.Name,
		GlobalRole: user.GlobalRole,
		Status:     user.Status,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}

	response.Success(c, http.StatusOK, "Token refreshed successfully", AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		User:         resUser,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid input data", err.Error())
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		response.BadRequest(c, "Validation failed", validator.FormatValidationError(err))
		return
	}

	hashedToken := hashToken(req.RefreshToken)
	
	// Try to get token first to check user ID for audit log
	dbToken, err := h.queries.GetRefreshToken(c.Request.Context(), hashedToken)
	var actorUserID pgtype.UUID
	if err == nil {
		actorUserID = dbToken.UserID
	}

	// Revoke token in database
	err = h.queries.RevokeRefreshToken(c.Request.Context(), hashedToken)
	if err != nil {
		response.InternalServerError(c, "Failed to revoke token")
		return
	}

	// Create Audit Log
	if actorUserID.Valid {
		_, _ = h.queries.CreateAuditLog(c.Request.Context(), database.CreateAuditLogParams{
			ActorUserID: actorUserID,
			Action:      "LOGOUT",
			EntityType:  "users",
			EntityID:    actorUserID,
			Ip:          pgtype.Text{String: c.ClientIP(), Valid: true},
			UserAgent:   pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
		})
	}

	response.Success(c, http.StatusOK, "Logged out successfully", nil)
}
