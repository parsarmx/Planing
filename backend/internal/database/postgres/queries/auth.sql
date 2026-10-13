-- name: GetUser :one
SELECT * FROM users WHERE id = $1;
-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = $1;
-- name: GetUsers :many
SELECT * FROM users WHERE id = ANY($1::text[]);
-- name: SaveUser :exec
INSERT INTO users (id,phone,name,avatar,avatar_config,role,created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, avatar=EXCLUDED.avatar, avatar_config=EXCLUDED.avatar_config, role=EXCLUDED.role;
-- name: GetOTP :one
SELECT * FROM otp_challenges WHERE phone = $1;
-- name: SaveOTP :exec
INSERT INTO otp_challenges (phone,hash,salt,expires_at,sent_at,attempts,window_start,sends)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (phone) DO UPDATE SET hash=EXCLUDED.hash,salt=EXCLUDED.salt,expires_at=EXCLUDED.expires_at,sent_at=EXCLUDED.sent_at,attempts=EXCLUDED.attempts,window_start=EXCLUDED.window_start,sends=EXCLUDED.sends;
-- name: GetSession :one
SELECT * FROM sessions WHERE token_hash = $1;
-- name: SaveSession :exec
INSERT INTO sessions (token_hash,user_id,expires_at) VALUES ($1,$2,$3)
ON CONFLICT (token_hash) DO UPDATE SET user_id=EXCLUDED.user_id,expires_at=EXCLUDED.expires_at;
-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;
-- name: PruneSessions :exec
DELETE FROM sessions WHERE expires_at < $1;
-- name: PruneOTPs :exec
DELETE FROM otp_challenges WHERE sent_at < $1;
