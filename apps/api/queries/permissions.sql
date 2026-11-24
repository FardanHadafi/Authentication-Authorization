-- name: GetRolePermissions :many
SELECT permission_id
FROM role_permissions
WHERE role_id = $1
ORDER BY permission_id ASC;

-- name: CheckRoleHasPermission :one
SELECT EXISTS (
    SELECT 1
    FROM role_permissions
    WHERE role_id = $1 AND permission_id = $2
);

-- name: ListRoles :many
SELECT id, name, description, created_at
FROM roles
ORDER BY id ASC;

-- name: ListPermissions :many
SELECT id, name, description, created_at
FROM permissions
ORDER BY id ASC;
