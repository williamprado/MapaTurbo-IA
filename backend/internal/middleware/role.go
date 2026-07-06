package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/pkg/response"
)

func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		globalRole, exists := c.Get("global_role")
		if !exists || globalRole != "SUPER_ADMIN" {
			response.Forbidden(c, "Super Admin privilege required")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireTenantRole(dbPool *pgxpool.Pool, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		globalRole, _ := c.Get("global_role")
		if globalRole == "SUPER_ADMIN" {
			c.Next()
			return
		}

		userID, uExists := c.Get("user_id")
		orgID, oExists := c.Get("org_id")
		if !uExists || !oExists {
			response.Unauthorized(c, "Authentication and active organization context required")
			c.Abort()
			return
		}

		var userRole string
		err := dbPool.QueryRow(context.Background(),
			"SELECT role FROM organization_users WHERE organization_id = $1 AND user_id = $2",
			orgID, userID,
		).Scan(&userRole)

		if err != nil {
			response.Forbidden(c, "Access denied: you do not belong to this organization")
			c.Abort()
			return
		}

		if requiredRole == "ORG_ADMIN" && userRole != "ORG_ADMIN" {
			response.Forbidden(c, "Organization Admin role required")
			c.Abort()
			return
		}

		c.Set("tenant_role", userRole)
		c.Next()
	}
}
