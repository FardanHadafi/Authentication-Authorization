-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    actor_id,
    actor_email,
    actor_role,
    action,
    resource_type,
    resource_id,
    status,
    ip_address,
    user_agent,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING id, actor_id, actor_email, actor_role, action, resource_type, resource_id, status, ip_address, user_agent, metadata, created_at;

-- name: ListAuditLogs :many
SELECT id, actor_id, actor_email, actor_role, action, resource_type, resource_id, status, ip_address, user_agent, metadata, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs;
