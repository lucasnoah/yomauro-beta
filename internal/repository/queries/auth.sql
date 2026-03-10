-- name: GetUserByEmail :one
SELECT id, email, password_hash, display_name, role, active FROM users WHERE email = @email;

-- name: CountActiveOwners :one
SELECT COUNT(*) FROM users WHERE role = 'owner' AND active = true;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, display_name, role)
VALUES (@email, @password_hash, @display_name, @role)
RETURNING *;

-- name: CreateSession :one
INSERT INTO sessions (id, user_id, expires_at)
VALUES (@id, @user_id, @expires_at)
RETURNING *;

-- name: GetSession :one
SELECT s.id, s.user_id, s.created_at, s.expires_at,
       u.email, u.display_name, u.role
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.id = @id AND s.expires_at > NOW() AND u.active = true;

-- name: ExtendSession :exec
UPDATE sessions SET expires_at = @expires_at WHERE id = @id;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = @id;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = @user_id;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < NOW();
