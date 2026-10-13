package repositories

import (
	"context"
	"errors"
	"planningpoker/internal/domain"
	"sort"
	"time"
)

var ErrNotFound = errors.New("record not found")

// Write serializes related changes by key. Domain errors still commit validated
// side effects (OTP attempt counters); storage failures roll back the transaction.
type Store interface {
	Read(context.Context, func(Transaction) error) error
	Write(context.Context, string, func(Transaction) error) error
	Close()
}
type Transaction interface {
	User(string) (domain.User, error)
	UserByPhone(string) (domain.User, error)
	Users([]string) (map[string]domain.User, error)
	SaveUser(domain.User) error
	OTP(string) (domain.OTP, error)
	SaveOTP(string, domain.OTP) error
	Session(string) (domain.Session, error)
	SaveSession(string, domain.Session) error
	DeleteSession(string) error
	PruneAuth(time.Time) error
	Room(string) (domain.Room, error)
	SaveRoom(domain.Room) error
	UserRooms(string) ([]domain.Room, error)
	UserRoomIDs(string) ([]string, error)
	CountHostedRooms(string) (int, error)
	PublicRooms(int, int) ([]domain.PublicRoom, error)
}

// FileStore is retained for unit-test fixtures and legacy import, never for serve.
type memoryTransaction struct{ state *domain.State }

func (t *memoryTransaction) User(id string) (domain.User, error) {
	u, ok := t.state.Users[id]
	if !ok {
		return u, ErrNotFound
	}
	return u, nil
}
func (t *memoryTransaction) UserByPhone(phone string) (domain.User, error) {
	for _, u := range t.state.Users {
		if u.Phone == phone {
			return u, nil
		}
	}
	return domain.User{}, ErrNotFound
}
func (t *memoryTransaction) Users(ids []string) (map[string]domain.User, error) {
	out := map[string]domain.User{}
	for _, id := range ids {
		if u, ok := t.state.Users[id]; ok {
			out[id] = u
		}
	}
	return out, nil
}
func (t *memoryTransaction) SaveUser(u domain.User) error { t.state.Users[u.ID] = u; return nil }
func (t *memoryTransaction) OTP(phone string) (domain.OTP, error) {
	o, ok := t.state.OTPs[phone]
	if !ok {
		return o, ErrNotFound
	}
	return o, nil
}
func (t *memoryTransaction) SaveOTP(phone string, o domain.OTP) error {
	t.state.OTPs[phone] = o
	return nil
}
func (t *memoryTransaction) Session(hash string) (domain.Session, error) {
	s, ok := t.state.Sessions[hash]
	if !ok {
		return s, ErrNotFound
	}
	return s, nil
}
func (t *memoryTransaction) SaveSession(hash string, s domain.Session) error {
	t.state.Sessions[hash] = s
	return nil
}
func (t *memoryTransaction) DeleteSession(hash string) error {
	delete(t.state.Sessions, hash)
	return nil
}
func (t *memoryTransaction) PruneAuth(now time.Time) error {
	for k, s := range t.state.Sessions {
		if now.After(s.ExpiresAt) {
			delete(t.state.Sessions, k)
		}
	}
	for k, o := range t.state.OTPs {
		if now.Sub(o.SentAt) > 24*time.Hour {
			delete(t.state.OTPs, k)
		}
	}
	return nil
}
func (t *memoryTransaction) Room(id string) (domain.Room, error) {
	r, ok := t.state.Rooms[id]
	if !ok {
		return r, ErrNotFound
	}
	return r, nil
}
func (t *memoryTransaction) SaveRoom(r domain.Room) error { t.state.Rooms[r.ID] = r; return nil }
func (t *memoryTransaction) UserRooms(id string) ([]domain.Room, error) {
	out := []domain.Room{}
	for _, r := range t.state.Rooms {
		for _, m := range r.Members {
			if m.UserID == id {
				out = append(out, r)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
func (t *memoryTransaction) UserRoomIDs(id string) ([]string, error) {
	rooms, _ := t.UserRooms(id)
	out := []string{}
	for _, r := range rooms {
		out = append(out, r.ID)
	}
	return out, nil
}
func (t *memoryTransaction) CountHostedRooms(id string) (int, error) {
	n := 0
	for _, r := range t.state.Rooms {
		if r.HostID == id && !r.Closed {
			n++
		}
	}
	return n, nil
}
func (t *memoryTransaction) PublicRooms(limit, offset int) ([]domain.PublicRoom, error) {
	rooms := []domain.Room{}
	for _, r := range t.state.Rooms {
		if r.Visibility == "public" && !r.Closed {
			rooms = append(rooms, r)
		}
	}
	sort.Slice(rooms, func(i, j int) bool {
		if rooms[i].CreatedAt.Equal(rooms[j].CreatedAt) {
			return rooms[i].ID < rooms[j].ID
		}
		return rooms[i].CreatedAt.After(rooms[j].CreatedAt)
	})
	out := []domain.PublicRoom{}
	for i := offset; i < len(rooms) && i < offset+limit; i++ {
		r := rooms[i]
		out = append(out, domain.PublicRoom{ID: r.ID, Name: r.Name, Deck: r.Deck, Theme: domain.RoomTheme(r.Theme), HostName: t.state.Users[r.HostID].Name, MemberCount: len(r.Members), Capacity: 20, CreatedAt: r.CreatedAt})
	}
	return out, nil
}
