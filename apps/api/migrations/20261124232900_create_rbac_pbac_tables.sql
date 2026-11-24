-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id VARCHAR(50) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(100) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

-- Seed Enterprise Default Roles
INSERT INTO roles (id, name, description) VALUES
    ('admin', 'Administrator', 'Full system administrative access and operational privileges'),
    ('member', 'Standard Member', 'Verified member with access to personal resources')
ON CONFLICT (id) DO NOTHING;

-- Seed Enterprise Granular Permissions
INSERT INTO permissions (id, name, description) VALUES
    ('users:read', 'Read Users', 'Can list, search, and view all registered users'),
    ('users:write', 'Write Users', 'Can create and modify user records'),
    ('users:delete', 'Delete Users', 'Can deactivate or delete user accounts'),
    ('audit_logs:read', 'Read Audit Logs', 'Can inspect immutable security compliance audit logs'),
    ('profile:read', 'Read Profile', 'Can view own personal profile data'),
    ('profile:write', 'Write Profile', 'Can update own personal profile and preferences')
ON CONFLICT (id) DO NOTHING;

-- Map Default Permissions to Roles
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('admin', 'users:read'),
    ('admin', 'users:write'),
    ('admin', 'users:delete'),
    ('admin', 'audit_logs:read'),
    ('admin', 'profile:read'),
    ('admin', 'profile:write'),
    ('member', 'profile:read'),
    ('member', 'profile:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
-- +goose StatementEnd
