package repositories

import (
	"context"
	"encoding/json"
	"planningpoker/internal/config"
	"planningpoker/internal/domain"
	"planningpoker/internal/testutil"
	"testing"
	"time"
)

func TestLegacyImportIsAtomicIdempotentAndPrivate(t *testing.T) {
	dsn := testutil.Postgres(t)
	store, err := NewPostgres(&config.Config{DatabaseURL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	st := domain.NewState()
	now := time.Now().UTC().Truncate(time.Microsecond)
	st.Users["host"] = domain.User{ID: "host", Phone: "+989120006500", Name: "Legacy Host", Avatar: "pink", CreatedAt: now}
	st.Rooms["room"] = domain.Room{ID: "room", Name: "Legacy room", HostID: "host", Deck: "fibonacci", Theme: "pink", Members: []domain.Member{{UserID: "host"}}, Rounds: []domain.Round{{ID: "round", Title: "Legacy task", Status: "revealed", Votes: map[string]string{"host": "8"}, StartedAt: now}}, ActiveRoundID: "round", CreatedAt: now, Revision: 3}
	st.Sessions["token-hash"] = domain.Session{UserID: "host", ExpiresAt: now.Add(time.Hour)}
	st.OTPs["+989120006500"] = domain.OTP{Hash: "hash", Salt: "salt", SentAt: now, ExpiresAt: now.Add(time.Minute), WindowStart: now, Sends: 1}
	bad, _ := json.Marshal(st)
	st.Users["other"] = domain.User{ID: "other", Phone: "+989120006500", Name: "duplicate", CreatedAt: now}
	invalid, _ := json.Marshal(st)
	if _, err = store.Import(ctx, invalid); err == nil {
		t.Fatal("invalid import accepted")
	}
	if err = store.Read(ctx, func(tx Transaction) error {
		_, err := tx.User("host")
		if err != ErrNotFound {
			t.Fatal("partial import was committed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	result, err := store.Import(ctx, bad)
	if err != nil || result.Users != 1 || result.Rooms != 1 || result.Sessions != 1 || result.OTPs != 1 {
		t.Fatal("import failed", result, err)
	}
	result, err = store.Import(ctx, bad)
	if err != nil || !result.AlreadyImported {
		t.Fatal("repeated import not idempotent", err)
	}
	if err = store.Read(ctx, func(tx Transaction) error {
		room, e := tx.Room("room")
		if e != nil {
			return e
		}
		if room.Visibility != "private" || room.Rounds[0].Votes["host"] != "8" || !room.LastActivityAt.Equal(now) {
			t.Fatal("legacy room changed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Import(ctx, invalid); err == nil {
		t.Fatal("import into existing database accepted")
	}
}
