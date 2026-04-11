-- name: GetUserByEmail :one
SELECT id, approved_user_id, email, password_hash, is_active, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, approved_user_id, email, password_hash, is_active, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (approved_user_id, email, password_hash, is_active)
VALUES ($1, $2, $3, $4)
RETURNING id, approved_user_id, email, password_hash, is_active, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET email = $2, password_hash = $3, is_active = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, approved_user_id, email, password_hash, is_active, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: GetUserRoles :many
SELECT r.id, r.name, r.description, r.created_at
FROM roles r
INNER JOIN user_roles ur ON r.id = ur.role_id
WHERE ur.user_id = $1
ORDER BY r.name;

-- name: GetRolesByNames :many
SELECT id, name, description, created_at
FROM roles
WHERE name = ANY($1::TEXT[]);

-- name: AssignRole :exec
INSERT INTO user_roles (user_id, role_id)
VALUES ($1, $2)
ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: RemoveRoleFromUser :exec
DELETE FROM user_roles
WHERE user_id = $1 AND role_id = $2;
