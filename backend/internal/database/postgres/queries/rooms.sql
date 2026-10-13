-- name: LockTransaction :exec
SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0));
-- name: GetRoom :one
SELECT * FROM rooms WHERE id=$1;
-- name: ListUserRooms :many
SELECT rooms.* FROM rooms JOIN room_members ON rooms.id=room_members.room_id
WHERE room_members.user_id=$1 ORDER BY rooms.created_at DESC,rooms.id;
-- name: UserRoomIDs :many
SELECT room_id FROM room_members WHERE user_id=$1;
-- name: CountHostedRooms :one
SELECT count(*) FROM rooms WHERE host_id=$1 AND NOT closed;
-- name: DiscoverRooms :many
SELECT r.id,r.name,r.deck,r.theme,u.name AS host_name,r.created_at,
       (SELECT count(*) FROM room_members m WHERE m.room_id=r.id) AS member_count
FROM rooms r JOIN users u ON u.id=r.host_id
WHERE r.visibility='public' AND NOT r.closed
ORDER BY r.created_at DESC,r.id LIMIT $1 OFFSET $2;
-- name: SaveRoom :exec
INSERT INTO rooms (id,name,host_id,deck,theme,visibility,active_round_id,created_at,last_activity_at,closed,revision)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,theme=EXCLUDED.theme,visibility=EXCLUDED.visibility,
active_round_id=EXCLUDED.active_round_id,last_activity_at=EXCLUDED.last_activity_at,closed=EXCLUDED.closed,revision=EXCLUDED.revision;
-- name: ListMembers :many
SELECT * FROM room_members WHERE room_id=$1 ORDER BY position;
-- name: SaveMember :exec
INSERT INTO room_members (room_id,user_id,observer,position) VALUES ($1,$2,$3,$4)
ON CONFLICT (room_id,user_id) DO UPDATE SET observer=EXCLUDED.observer;
-- name: ListRounds :many
SELECT * FROM rounds WHERE room_id=$1 ORDER BY position;
-- name: SaveRound :exec
INSERT INTO rounds (id,room_id,title,description,status,estimate,started_at,position)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title,description=EXCLUDED.description,status=EXCLUDED.status,estimate=EXCLUDED.estimate,started_at=EXCLUDED.started_at;
-- name: ListVotes :many
SELECT * FROM votes WHERE room_id=$1;
-- name: SaveVote :exec
INSERT INTO votes (room_id,round_id,user_id,value) VALUES ($1,$2,$3,$4)
ON CONFLICT (round_id,user_id) DO UPDATE SET value=EXCLUDED.value;
-- name: DeleteVote :exec
DELETE FROM votes WHERE round_id=$1 AND user_id=$2;
-- name: HasLegacyImport :one
SELECT EXISTS(SELECT 1 FROM legacy_imports WHERE source_hash=$1);
-- name: RecordLegacyImport :exec
INSERT INTO legacy_imports (source_hash) VALUES ($1);
-- name: DatabaseEmpty :one
SELECT NOT EXISTS(SELECT 1 FROM users) AND NOT EXISTS(SELECT 1 FROM rooms) AS empty;
