-- +goose Up
CREATE TABLE users (
    id text PRIMARY KEY,
    phone text NOT NULL UNIQUE,
    name text NOT NULL DEFAULT '',
    avatar text NOT NULL DEFAULT 'violet',
    avatar_config jsonb,
    role text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL
);
CREATE TABLE otp_challenges (
    phone text PRIMARY KEY,
    hash text NOT NULL,
    salt text NOT NULL,
    expires_at timestamptz NOT NULL,
    sent_at timestamptz NOT NULL,
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    window_start timestamptz NOT NULL,
    sends integer NOT NULL CHECK (sends >= 0)
);
CREATE TABLE sessions (
    token_hash text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions(expires_at);
CREATE TABLE rooms (
    id text PRIMARY KEY,
    name text NOT NULL,
    host_id text NOT NULL REFERENCES users(id),
    deck text NOT NULL CHECK (deck IN ('fibonacci', 'tshirt')),
    theme text NOT NULL DEFAULT 'violet' CHECK (theme IN ('violet','pink','blue','mint','amber','coral')),
    visibility text NOT NULL DEFAULT 'private' CHECK (visibility IN ('private','public')),
    active_round_id text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    last_activity_at timestamptz NOT NULL,
    closed boolean NOT NULL DEFAULT false,
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0)
);
CREATE INDEX rooms_public_idx ON rooms(created_at DESC, id) WHERE visibility = 'public' AND NOT closed;
CREATE INDEX rooms_host_idx ON rooms(host_id) WHERE NOT closed;
CREATE TABLE room_members (
    room_id text NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    observer boolean NOT NULL DEFAULT false,
    position integer NOT NULL CHECK (position >= 0),
    PRIMARY KEY (room_id,user_id),
    UNIQUE (room_id,position)
);
CREATE INDEX room_members_user_idx ON room_members(user_id,room_id);
CREATE TABLE rounds (
    id text PRIMARY KEY,
    room_id text NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    status text NOT NULL CHECK (status IN ('pending','voting','revealed','completed')),
    estimate text NOT NULL DEFAULT '',
    started_at timestamptz,
    position integer NOT NULL CHECK (position >= 0),
    UNIQUE (room_id,id),
    UNIQUE (room_id,position)
);
CREATE TABLE votes (
    room_id text NOT NULL,
    round_id text NOT NULL,
    user_id text NOT NULL,
    value text NOT NULL,
    PRIMARY KEY (round_id,user_id),
    FOREIGN KEY (room_id,round_id) REFERENCES rounds(room_id,id) ON DELETE CASCADE,
    FOREIGN KEY (room_id,user_id) REFERENCES room_members(room_id,user_id) ON DELETE CASCADE
);
CREATE TABLE legacy_imports (
    source_hash text PRIMARY KEY,
    imported_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE legacy_imports;
DROP TABLE votes;
DROP TABLE rounds;
DROP TABLE room_members;
DROP TABLE rooms;
DROP TABLE sessions;
DROP TABLE otp_challenges;
DROP TABLE users;
