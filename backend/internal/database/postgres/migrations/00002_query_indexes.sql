-- +goose Up
CREATE INDEX votes_room_idx ON votes(room_id);
CREATE INDEX otp_challenges_sent_idx ON otp_challenges(sent_at);

-- +goose Down
DROP INDEX otp_challenges_sent_idx;
DROP INDEX votes_room_idx;
