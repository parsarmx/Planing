package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"planningpoker/internal/config"
	"planningpoker/internal/database/postgres/sqlc"
	"planningpoker/internal/domain"
	"reflect"
	"time"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func New(c *config.Config) (Store, error) { return NewPostgres(c) }
func NewPostgres(c *config.Config) (*PostgresStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pc, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL configuration")
	}
	if c.DB.MaxConns > 0 {
		pc.MaxConns = c.DB.MaxConns
		pc.MinConns = c.DB.MinConns
	}
	p, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, err
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("PostgreSQL is unavailable: %w", err)
	}
	// Serve never creates tables or silently starts without applied migrations.
	if _, err = sqlc.New(p).CountHostedRooms(ctx, ""); err != nil {
		p.Close()
		return nil, fmt.Errorf("database schema is unavailable; run pg_migrate: %w", err)
	}
	return &PostgresStore{p}, nil
}
func (s *PostgresStore) Close() { s.pool.Close() }
func (s *PostgresStore) Read(ctx context.Context, fn func(Transaction) error) error {
	return s.transaction(ctx, "", true, fn)
}
func (s *PostgresStore) Write(ctx context.Context, key string, fn func(Transaction) error) error {
	return s.transaction(ctx, key, false, fn)
}
func (s *PostgresStore) transaction(ctx context.Context, key string, readOnly bool, fn func(Transaction) error) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	opts := pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
	if readOnly {
		opts = pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	}
	tx, err := s.pool.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := sqlc.New(tx)
	if !readOnly {
		if err = q.LockTransaction(ctx, key); err != nil {
			return err
		}
	}
	result := fn(&postgresTransaction{ctx: ctx, q: q, original: map[string]domain.Room{}})
	if result != nil {
		var d *domain.Error
		if !errors.As(result, &d) {
			return result
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return result
}

type postgresTransaction struct {
	ctx      context.Context
	q        *sqlc.Queries
	original map[string]domain.Room
}

func stamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()} }
func missing(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
func decodeUser(u sqlc.User) (domain.User, error) {
	out := domain.User{ID: u.ID, Phone: u.Phone, Name: u.Name, Avatar: u.Avatar, Role: u.Role, CreatedAt: u.CreatedAt.Time}
	if len(u.AvatarConfig) > 0 && string(u.AvatarConfig) != "null" {
		if err := json.Unmarshal(u.AvatarConfig, &out.AvatarConfig); err != nil {
			return out, err
		}
	}
	return out, nil
}
func (t *postgresTransaction) User(id string) (domain.User, error) {
	u, e := t.q.GetUser(t.ctx, id)
	if e != nil {
		return domain.User{}, missing(e)
	}
	return decodeUser(u)
}
func (t *postgresTransaction) UserByPhone(phone string) (domain.User, error) {
	u, e := t.q.GetUserByPhone(t.ctx, phone)
	if e != nil {
		return domain.User{}, missing(e)
	}
	return decodeUser(u)
}
func (t *postgresTransaction) Users(ids []string) (map[string]domain.User, error) {
	rows, e := t.q.GetUsers(t.ctx, ids)
	if e != nil {
		return nil, e
	}
	out := map[string]domain.User{}
	for _, v := range rows {
		u, e := decodeUser(v)
		if e != nil {
			return nil, e
		}
		out[u.ID] = u
	}
	return out, nil
}
func (t *postgresTransaction) SaveUser(u domain.User) error {
	var a []byte
	if u.AvatarConfig != nil {
		var e error
		a, e = json.Marshal(u.AvatarConfig)
		if e != nil {
			return e
		}
	}
	return t.q.SaveUser(t.ctx, sqlc.SaveUserParams{ID: u.ID, Phone: u.Phone, Name: u.Name, Avatar: u.Avatar, AvatarConfig: a, Role: u.Role, CreatedAt: stamp(u.CreatedAt)})
}
func (t *postgresTransaction) OTP(phone string) (domain.OTP, error) {
	o, e := t.q.GetOTP(t.ctx, phone)
	if e != nil {
		return domain.OTP{}, missing(e)
	}
	return domain.OTP{Hash: o.Hash, Salt: o.Salt, ExpiresAt: o.ExpiresAt.Time, SentAt: o.SentAt.Time, Attempts: int(o.Attempts), WindowStart: o.WindowStart.Time, Sends: int(o.Sends)}, nil
}
func (t *postgresTransaction) SaveOTP(phone string, o domain.OTP) error {
	return t.q.SaveOTP(t.ctx, sqlc.SaveOTPParams{Phone: phone, Hash: o.Hash, Salt: o.Salt, ExpiresAt: stamp(o.ExpiresAt), SentAt: stamp(o.SentAt), Attempts: int32(o.Attempts), WindowStart: stamp(o.WindowStart), Sends: int32(o.Sends)})
}
func (t *postgresTransaction) Session(hash string) (domain.Session, error) {
	s, e := t.q.GetSession(t.ctx, hash)
	if e != nil {
		return domain.Session{}, missing(e)
	}
	return domain.Session{UserID: s.UserID, ExpiresAt: s.ExpiresAt.Time}, nil
}
func (t *postgresTransaction) SaveSession(hash string, s domain.Session) error {
	return t.q.SaveSession(t.ctx, sqlc.SaveSessionParams{TokenHash: hash, UserID: s.UserID, ExpiresAt: stamp(s.ExpiresAt)})
}
func (t *postgresTransaction) DeleteSession(hash string) error { return t.q.DeleteSession(t.ctx, hash) }
func (t *postgresTransaction) PruneAuth(now time.Time) error {
	if e := t.q.PruneSessions(t.ctx, stamp(now)); e != nil {
		return e
	}
	return t.q.PruneOTPs(t.ctx, stamp(now.Add(-24*time.Hour)))
}
func (t *postgresTransaction) Room(id string) (domain.Room, error) {
	r, e := t.q.GetRoom(t.ctx, id)
	if e != nil {
		return domain.Room{}, missing(e)
	}
	return t.hydrate(r)
}
func (t *postgresTransaction) hydrate(r sqlc.Room) (domain.Room, error) {
	out := domain.Room{ID: r.ID, Name: r.Name, HostID: r.HostID, Deck: r.Deck, Theme: r.Theme, Visibility: r.Visibility, ActiveRoundID: r.ActiveRoundID, CreatedAt: r.CreatedAt.Time, LastActivityAt: r.LastActivityAt.Time, Closed: r.Closed, Revision: r.Revision, Members: []domain.Member{}, Rounds: []domain.Round{}}
	ms, e := t.q.ListMembers(t.ctx, r.ID)
	if e != nil {
		return out, e
	}
	for _, m := range ms {
		out.Members = append(out.Members, domain.Member{UserID: m.UserID, Observer: m.Observer})
	}
	rs, e := t.q.ListRounds(t.ctx, r.ID)
	if e != nil {
		return out, e
	}
	indexes := map[string]int{}
	for _, v := range rs {
		indexes[v.ID] = len(out.Rounds)
		out.Rounds = append(out.Rounds, domain.Round{ID: v.ID, Title: v.Title, Description: v.Description, Status: v.Status, Estimate: v.Estimate, StartedAt: v.StartedAt.Time, Votes: map[string]string{}})
	}
	vs, e := t.q.ListVotes(t.ctx, r.ID)
	if e != nil {
		return out, e
	}
	for _, v := range vs {
		out.Rounds[indexes[v.RoundID]].Votes[v.UserID] = v.Value
	}
	// Callers mutate votes in place; retain an independent copy for differential writes.
	b, e := json.Marshal(out)
	if e != nil {
		return out, e
	}
	var original domain.Room
	if e = json.Unmarshal(b, &original); e != nil {
		return out, e
	}
	t.original[r.ID] = original
	return out, nil
}
func (t *postgresTransaction) SaveRoom(r domain.Room) error {
	if e := t.q.SaveRoom(t.ctx, sqlc.SaveRoomParams{ID: r.ID, Name: r.Name, HostID: r.HostID, Deck: r.Deck, Theme: domain.RoomTheme(r.Theme), Visibility: domain.RoomVisibility(r.Visibility), ActiveRoundID: r.ActiveRoundID, CreatedAt: stamp(r.CreatedAt), LastActivityAt: stamp(domain.RoomActivity(r)), Closed: r.Closed, Revision: r.Revision}); e != nil {
		return e
	}
	old := t.original[r.ID]
	members := map[string]domain.Member{}
	for _, m := range old.Members {
		members[m.UserID] = m
	}
	for i, m := range r.Members {
		prev, ok := members[m.UserID]
		if !ok || prev != m {
			if e := t.q.SaveMember(t.ctx, sqlc.SaveMemberParams{RoomID: r.ID, UserID: m.UserID, Observer: m.Observer, Position: int32(i)}); e != nil {
				return e
			}
		}
	}
	rounds := map[string]domain.Round{}
	for _, v := range old.Rounds {
		rounds[v.ID] = v
	}
	for i, v := range r.Rounds {
		prev, ok := rounds[v.ID]
		if ok && reflect.DeepEqual(prev, v) {
			continue
		}
		if e := t.q.SaveRound(t.ctx, sqlc.SaveRoundParams{ID: v.ID, RoomID: r.ID, Title: v.Title, Description: v.Description, Status: v.Status, Estimate: v.Estimate, StartedAt: stamp(v.StartedAt), Position: int32(i)}); e != nil {
			return e
		}
		for uid := range prev.Votes {
			if _, ok := v.Votes[uid]; !ok {
				if e := t.q.DeleteVote(t.ctx, sqlc.DeleteVoteParams{RoundID: v.ID, UserID: uid}); e != nil {
					return e
				}
			}
		}
		for uid, value := range v.Votes {
			if prev.Votes[uid] != value {
				if e := t.q.SaveVote(t.ctx, sqlc.SaveVoteParams{RoomID: r.ID, RoundID: v.ID, UserID: uid, Value: value}); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
func (t *postgresTransaction) UserRooms(id string) ([]domain.Room, error) {
	rows, e := t.q.ListUserRooms(t.ctx, id)
	if e != nil {
		return nil, e
	}
	out := []domain.Room{}
	for _, r := range rows {
		v, e := t.hydrate(r)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}
func (t *postgresTransaction) UserRoomIDs(id string) ([]string, error) {
	return t.q.UserRoomIDs(t.ctx, id)
}
func (t *postgresTransaction) CountHostedRooms(id string) (int, error) {
	n, e := t.q.CountHostedRooms(t.ctx, id)
	return int(n), e
}
func (t *postgresTransaction) PublicRooms(limit, offset int) ([]domain.PublicRoom, error) {
	rows, e := t.q.DiscoverRooms(t.ctx, sqlc.DiscoverRoomsParams{Limit: int32(limit), Offset: int32(offset)})
	if e != nil {
		return nil, e
	}
	out := []domain.PublicRoom{}
	for _, r := range rows {
		out = append(out, domain.PublicRoom{ID: r.ID, Name: r.Name, Deck: r.Deck, Theme: r.Theme, HostName: r.HostName, MemberCount: int(r.MemberCount), Capacity: 20, CreatedAt: r.CreatedAt.Time})
	}
	return out, nil
}
