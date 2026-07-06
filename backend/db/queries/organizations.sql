-- name: CreateOrganization :one
INSERT INTO organizations (name, slug, status)
VALUES ($1, $2, $3)
RETURNING id, name, slug, status, created_at, updated_at;

-- name: GetOrganizationBySlug :one
SELECT id, name, slug, status, created_at, updated_at
FROM organizations
WHERE slug = $1 LIMIT 1;

-- name: GetOrganizationByID :one
SELECT id, name, slug, status, created_at, updated_at
FROM organizations
WHERE id = $1 LIMIT 1;

-- name: UpdateOrganization :one
UPDATE organizations
SET name = COALESCE($2, name),
    slug = COALESCE($3, slug),
    status = COALESCE($4, status)
WHERE id = $1
RETURNING id, name, slug, status, created_at, updated_at;

-- name: ListOrganizations :many
SELECT id, name, slug, status, created_at
FROM organizations
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountOrganizations :one
SELECT COUNT(*) FROM organizations;

-- name: CreateOrganizationUser :one
INSERT INTO organization_users (organization_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING id, organization_id, user_id, role, created_at, updated_at;

-- name: GetOrganizationUser :one
SELECT id, organization_id, user_id, role, created_at, updated_at
FROM organization_users
WHERE organization_id = $1 AND user_id = $2 LIMIT 1;

-- name: GetUserOrganizations :many
SELECT ou.organization_id, o.name AS organization_name, o.slug AS organization_slug, o.status AS organization_status, ou.role
FROM organization_users ou
JOIN organizations o ON o.id = ou.organization_id
WHERE ou.user_id = $1;

-- name: UpdateOrganizationUserRole :one
UPDATE organization_users
SET role = $3
WHERE organization_id = $1 AND user_id = $2
RETURNING id, organization_id, user_id, role, created_at, updated_at;

-- name: RemoveOrganizationUser :exec
DELETE FROM organization_users
WHERE organization_id = $1 AND user_id = $2;

-- name: ListOrganizationUsers :many
SELECT ou.id, ou.organization_id, ou.user_id, ou.role, ou.created_at,
       u.email, u.name as user_name
FROM organization_users ou
JOIN users u ON ou.user_id = u.id
WHERE ou.organization_id = $1;
