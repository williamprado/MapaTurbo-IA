package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"mapaturbo-ia/pkg/response"
)

func TenantMiddleware(dbPool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgIDStr := c.GetHeader("X-Organization-ID")
		if orgIDStr == "" {
			orgIDStr = c.Query("organization_id")
		}

		userID, uExists := c.Get("user_id")
		globalRole, rExists := c.Get("global_role")

		if rExists && globalRole == "SUPER_ADMIN" {
			if orgIDStr != "" {
				var orgID pgtype.UUID
				if err := orgID.Scan(orgIDStr); err == nil {
					c.Set("org_id", orgID)
				}
			}
			c.Next()
			return
		}

		if !uExists {
			response.Unauthorized(c, "Authentication context required")
			c.Abort()
			return
		}

		var orgID pgtype.UUID

		if orgIDStr == "" {
			err := dbPool.QueryRow(context.Background(),
				"SELECT organization_id FROM organization_users WHERE user_id = $1 LIMIT 1",
				userID,
			).Scan(&orgID)
			if err != nil {
				response.BadRequest(c, "Active organization context required. Please select an organization.", nil)
				c.Abort()
				return
			}
		} else {
			if err := orgID.Scan(orgIDStr); err != nil {
				response.BadRequest(c, "Invalid organization ID format", nil)
				c.Abort()
				return
			}

			var exists bool
			err := dbPool.QueryRow(context.Background(),
				"SELECT EXISTS(SELECT 1 FROM organization_users WHERE organization_id = $1 AND user_id = $2)",
				orgID, userID,
			).Scan(&exists)
			if err != nil || !exists {
				response.Forbidden(c, "Access denied: you do not belong to this organization")
				c.Abort()
				return
			}
		}

		c.Set("org_id", orgID)
		c.Next()
	}
}
