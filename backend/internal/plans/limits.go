package plans

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"mapaturbo-ia/internal/database"
)

type OrganizationLimits struct {
	MaxMaps         int32
	MaxFiles        int32
	MaxUsers        int32
	MaxStorageBytes int64
	Features        map[string]bool
}

// Fallback / default free tier limits
var DefaultFreeLimits = OrganizationLimits{
	MaxMaps:         5,
	MaxFiles:        3,
	MaxUsers:        2, // Admin + 1 User
	MaxStorageBytes: 10 * 1024 * 1024, // 10MB
	Features: map[string]bool{
		"generateTopic":  true,
		"generateText":   true,
		"generatePdf":    false,
		"visualEditor":   true,
		"uploadPdf":      false,
		"exportPng":      false,
		"exportPdf":      false,
		"creditsHistory": true,
	},
}

type LimitService struct {
	queries *database.Queries
}

func NewLimitService(queries *database.Queries) *LimitService {
	return &LimitService{queries: queries}
}

func (s *LimitService) GetLimits(ctx context.Context, orgID pgtype.UUID) (OrganizationLimits, error) {
	sub, err := s.queries.GetSubscriptionByOrg(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DefaultFreeLimits, nil
		}
		return DefaultFreeLimits, err
	}

	// Statuses that grant features: ACTIVE, TRIALING
	// Statuses that block premium features: PENDING, PAST_DUE, CANCELED, SUSPENDED, FAILED
	if sub.Status != "ACTIVE" && sub.Status != "TRIALING" {
		return DefaultFreeLimits, nil
	}

	plan, err := s.queries.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DefaultFreeLimits, nil
		}
		return DefaultFreeLimits, err
	}

	var features map[string]bool
	if len(plan.Features) > 0 {
		_ = json.Unmarshal(plan.Features, &features)
	}
	if features == nil {
		features = make(map[string]bool)
	}

	return OrganizationLimits{
		MaxMaps:         plan.MaxMaps,
		MaxFiles:        plan.MaxFiles,
		MaxUsers:        plan.MaxUsers,
		MaxStorageBytes: plan.MaxStorageBytes,
		Features:        features,
	}, nil
}

func (s *LimitService) CanCreateMindMap(ctx context.Context, orgID pgtype.UUID) (bool, int32, int32, error) {
	limits, err := s.GetLimits(ctx, orgID)
	if err != nil {
		return false, 0, 0, err
	}

	// Count existing maps of organization
	count, err := s.queries.CountMindMapsByOrganization(ctx, orgID)
	if err != nil {
		return false, 0, 0, err
	}

	currentCount := int32(count)

	// If limit is 0, interpret as blocked unless max_maps is negative (representing unlimited) or positive.
	// But according to rules: "se limite for 0, interpretar como ilimitado apenas se essa for a regra já definida. Caso contrário, tratar como bloqueado."
	// Let's treat negative as unlimited. If max_maps <= 0, we can block (meaning 0 maps allowed) unless it's configured as unlimited (e.g. -1).
	// To align with standard, let's treat -1 as unlimited and 0 as blocked.
	if limits.MaxMaps == 0 {
		return false, currentCount, limits.MaxMaps, nil
	}

	if limits.MaxMaps > 0 && currentCount >= limits.MaxMaps {
		return false, currentCount, limits.MaxMaps, nil
	}
	return true, currentCount, limits.MaxMaps, nil
}

func (s *LimitService) CanUploadFile(ctx context.Context, orgID pgtype.UUID, additionalBytes int64) (bool, int32, int32, int64, int64, error) {
	limits, err := s.GetLimits(ctx, orgID)
	if err != nil {
		return false, 0, 0, 0, 0, err
	}

	// Count existing uploads (excluding FAILED status)
	count, err := s.queries.CountUploadsByOrganization(ctx, orgID)
	if err != nil {
		return false, 0, 0, 0, 0, err
	}
	currentCount := int32(count)

	if limits.MaxFiles == 0 {
		return false, currentCount, limits.MaxFiles, 0, limits.MaxStorageBytes, nil
	}
	if limits.MaxFiles > 0 && currentCount >= limits.MaxFiles {
		return false, currentCount, limits.MaxFiles, 0, limits.MaxStorageBytes, nil
	}

	// Sum existing storage size (excluding FAILED status)
	totalSize, err := s.queries.SumUploadSizeByOrganization(ctx, orgID)
	if err != nil {
		return false, currentCount, limits.MaxFiles, 0, limits.MaxStorageBytes, err
	}

	if totalSize+additionalBytes > limits.MaxStorageBytes {
		return false, currentCount, limits.MaxFiles, totalSize, limits.MaxStorageBytes, nil
	}

	return true, currentCount, limits.MaxFiles, totalSize, limits.MaxStorageBytes, nil
}

func (s *LimitService) CanAddUser(ctx context.Context, orgID pgtype.UUID) (bool, int32, int32, error) {
	limits, err := s.GetLimits(ctx, orgID)
	if err != nil {
		return false, 0, 0, err
	}

	// Count members
	count, err := s.queries.CountOrganizationUsers(ctx, orgID)
	if err != nil {
		return false, 0, 0, err
	}
	currentCount := int32(count)

	if limits.MaxUsers == 0 {
		return false, currentCount, limits.MaxUsers, nil
	}
	if limits.MaxUsers > 0 && currentCount >= limits.MaxUsers {
		return false, currentCount, limits.MaxUsers, nil
	}
	return true, currentCount, limits.MaxUsers, nil
}

func (s *LimitService) CanUseFeature(ctx context.Context, orgID pgtype.UUID, featureKey string) (bool, error) {
	limits, err := s.GetLimits(ctx, orgID)
	if err != nil {
		return false, err
	}

	allowed, exists := limits.Features[featureKey]
	return exists && allowed, nil
}

// LogPlanLimitReached writes audit log and helper
func (s *LimitService) LogPlanLimitReached(ctx context.Context, actorID, orgID pgtype.UUID, feature string, limit int32, current int32) {
	meta, _ := json.Marshal(map[string]interface{}{
		"feature":        feature,
		"limit":          limit,
		"currentUsage":   current,
		"organizationId": uuidToString(orgID),
	})
	_, _ = s.queries.CreateAuditLog(ctx, database.CreateAuditLogParams{
		ActorUserID:    actorID,
		OrganizationID: orgID,
		Action:         "PLAN_LIMIT_REACHED",
		EntityType:     "plans",
		EntityID:       orgID, // link to org
		Metadata:       meta,
		Ip:             pgtype.Text{String: "127.0.0.1", Valid: true},
		UserAgent:      pgtype.Text{String: "System Limit Gate", Valid: true},
	})
}

// LogFeatureBlocked writes audit log
func (s *LimitService) LogFeatureBlocked(ctx context.Context, actorID, orgID pgtype.UUID, feature string) {
	meta, _ := json.Marshal(map[string]interface{}{
		"feature":        feature,
		"organizationId": uuidToString(orgID),
	})
	_, _ = s.queries.CreateAuditLog(ctx, database.CreateAuditLogParams{
		ActorUserID:    actorID,
		OrganizationID: orgID,
		Action:         "FEATURE_BLOCKED_BY_PLAN",
		EntityType:     "plans",
		EntityID:       orgID,
		Metadata:       meta,
		Ip:             pgtype.Text{String: "127.0.0.1", Valid: true},
		UserAgent:      pgtype.Text{String: "System Limit Gate", Valid: true},
	})
}

