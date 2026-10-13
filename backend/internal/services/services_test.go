package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"planningpoker/internal/config"
	"planningpoker/internal/domain"
	"planningpoker/internal/producers"
	"planningpoker/internal/realtime"
	"planningpoker/internal/repositories"
	"planningpoker/pkg"
	"strings"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Auth, *Rooms, *repositories.FileStore) {
	t.Helper()
	s, err := repositories.NewFile("")
	if err != nil {
		t.Fatal(err)
	}
	h := realtime.New()
	return NewAuth(s, producers.DevSMS{}, &config.Config{Environment: "development"}, h), NewRooms(s, h), s
}
func login(t *testing.T, a *Auth, phone, name string) (domain.User, string) {
	t.Helper()
	ctx := context.Background()
	ch, err := a.Request(ctx, phone)
	if err != nil {
		t.Fatal(err)
	}
	u, token, err := a.Verify(ctx, phone, ch.DevCode)
	if err != nil {
		t.Fatal(err)
	}
	u, err = a.Profile(ctx, u.ID, name, "violet", "developer", nil)
	if err != nil {
		t.Fatal(err)
	}
	return u, token
}
func connect(t *testing.T, r *Rooms, room, user string) func() {
	t.Helper()
	ch, cancel := r.hub.Subscribe(room, user)
	if ch == nil {
		t.Fatal("presence connection rejected")
	}
	t.Cleanup(cancel)
	return cancel
}
func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	var e *domain.Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
}
func TestOTPReplayExpiryAndAttemptLimit(t *testing.T) {
	ctx := context.Background()
	a, _, s := fixture(t)
	ch, err := a.Request(ctx, "۰۹۱۲۳۴۵۶۷۸۹")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Phone != "+989123456789" {
		t.Fatal(ch.Phone)
	}
	_, err = a.Request(ctx, ch.Phone)
	assertCode(t, err, "resend_wait")
	for i := 0; i < 5; i++ {
		wrong := "000000"
		if ch.DevCode == wrong {
			wrong = "111111"
		}
		_, _, err = a.Verify(ctx, ch.Phone, wrong)
		assertCode(t, err, "invalid_otp")
	}
	_, _, err = a.Verify(ctx, ch.Phone, ch.DevCode)
	assertCode(t, err, "invalid_otp")
	ch, err = a.Request(ctx, "09123456788")
	if err != nil {
		t.Fatal(err)
	}
	u, token, err := a.Verify(ctx, ch.Phone, ch.DevCode)
	if err != nil || u.ID == "" {
		t.Fatal(err)
	}
	_, _, err = a.Verify(ctx, ch.Phone, ch.DevCode)
	assertCode(t, err, "invalid_otp")
	if _, err = a.Resolve(ctx, token); err != nil {
		t.Fatal(err)
	}
	_ = s.Update(ctx, func(st *domain.State) error {
		v := st.Sessions[pkg.Hash(token)]
		v.ExpiresAt = time.Now().Add(-time.Second)
		st.Sessions[pkg.Hash(token)] = v
		return nil
	})
	_, err = a.Resolve(ctx, token)
	assertCode(t, err, "unauthenticated")
	ch, err = a.Request(ctx, "09123456787")
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Update(ctx, func(st *domain.State) error {
		o := st.OTPs[ch.Phone]
		o.ExpiresAt = time.Now().Add(-time.Second)
		st.OTPs[ch.Phone] = o
		return nil
	})
	_, _, err = a.Verify(ctx, ch.Phone, ch.DevCode)
	assertCode(t, err, "invalid_otp")
}
func TestOTPConcurrentVerificationOnlyOneSession(t *testing.T) {
	a, _, _ := fixture(t)
	ctx := context.Background()
	ch, err := a.Request(ctx, "09120000001")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := a.Verify(ctx, ch.Phone, ch.DevCode)
			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Fatalf("got %d successful OTP verifications", success)
	}
}
func TestRoomPermissionsSecretVotesAndRoundLifecycle(t *testing.T) {
	ctx := context.Background()
	a, r, _ := fixture(t)
	host, _ := login(t, a, "09120000002", "Host")
	guest, _ := login(t, a, "09120000003", "Guest")
	outsider, _ := login(t, a, "09120000004", "Outsider")
	observer, _ := login(t, a, "09120000005", "Observer")
	room, err := r.Create(ctx, host.ID, "Sprint", "fibonacci", "")
	if err != nil {
		t.Fatal(err)
	}
	id := room.ID
	connect(t, r, id, host.ID)
	connect(t, r, id, guest.ID)
	_, err = r.Snapshot(ctx, id, outsider.ID)
	assertCode(t, err, "not_member")
	if _, err = r.Join(ctx, id, guest.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Join(ctx, id, observer.ID, true); err != nil {
		t.Fatal(err)
	}
	_, err = r.Command(ctx, id, guest.ID, Command{Action: "add_round", Title: "Forbidden"})
	assertCode(t, err, "host_required")
	room, err = r.Command(ctx, id, host.ID, Command{Action: "add_round", Title: "Implement OTP"})
	if err != nil {
		t.Fatal(err)
	}
	roundID := room.ActiveRoundID
	_, err = r.Command(ctx, id, observer.ID, Command{Action: "vote", Value: "5", RoundID: roundID})
	assertCode(t, err, "observer")
	_, err = r.Command(ctx, id, guest.ID, Command{Action: "vote", Value: "5", RoundID: "old-round"})
	assertCode(t, err, "stale_round")
	_, err = r.Command(ctx, id, guest.ID, Command{Action: "vote", Value: "999", RoundID: roundID})
	assertCode(t, err, "invalid_card")
	_, err = r.Command(ctx, id, guest.ID, Command{Action: "vote", Value: "8", RoundID: roundID})
	if err != nil {
		t.Fatal(err)
	}
	room, err = r.Snapshot(ctx, id, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range room.Members {
		if m.Vote != "" {
			t.Fatal("host can see a hidden vote")
		}
	}
	if room.Average != 0 || len(room.Distribution) != 0 {
		t.Fatal("hidden aggregate leak")
	}
	b, _ := json.Marshal(room)
	if strings.Contains(string(b), "phone") || strings.Contains(string(b), guest.Phone) {
		t.Fatal("participant phone exposed")
	}
	room, err = r.Snapshot(ctx, id, guest.ID)
	if err != nil || room.MyVote != "8" {
		t.Fatal("own vote should be available")
	}
	_, err = r.Command(ctx, id, guest.ID, Command{Action: "reveal"})
	assertCode(t, err, "host_required")
	room, err = r.Command(ctx, id, host.ID, Command{Action: "reveal"})
	if err != nil || room.Average != 8 || room.Distribution["8"] != 1 {
		t.Fatalf("reveal result: %+v %v", room, err)
	}
	_, err = r.Command(ctx, id, guest.ID, Command{Action: "vote", Value: "5", RoundID: roundID})
	assertCode(t, err, "not_voting")
	room, err = r.Command(ctx, id, host.ID, Command{Action: "reset"})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range room.Members {
		if m.Vote != "" || m.HasVoted {
			t.Fatal("reset did not clear votes")
		}
	}
	_, err = r.Command(ctx, id, guest.ID, Command{Action: "vote", Value: "5", RoundID: roundID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Command(ctx, id, host.ID, Command{Action: "reveal"})
	if err != nil {
		t.Fatal(err)
	}
	room, err = r.Command(ctx, id, host.ID, Command{Action: "estimate", Value: "5"})
	if err != nil || room.Rounds[0].Estimate != "5" {
		t.Fatal(err)
	}
	room, err = r.Command(ctx, id, host.ID, Command{Action: "add_round", Title: "Next issue"})
	if err != nil {
		t.Fatal(err)
	}
	next := room.Rounds[1].ID
	room, err = r.Command(ctx, id, host.ID, Command{Action: "start_round", RoundID: next})
	if err != nil {
		t.Fatal(err)
	}
	if room.MyVote != "" || room.Average != 0 {
		t.Fatal("new round leaked prior results")
	}
	_, err = r.Command(ctx, id, guest.ID, Command{Action: "vote", Value: "5", RoundID: roundID})
	assertCode(t, err, "stale_round")
	_, err = r.Command(ctx, id, host.ID, Command{Action: "close"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Command(ctx, id, guest.ID, Command{Action: "vote", Value: "5", RoundID: next})
	assertCode(t, err, "room_closed")
	_, err = r.Join(ctx, id, outsider.ID, false)
	assertCode(t, err, "room_closed")
}
func TestConcurrentVotesNoLostUpdates(t *testing.T) {
	ctx := context.Background()
	a, r, s := fixture(t)
	host, _ := login(t, a, "09120000006", "Host")
	room, err := r.Create(ctx, host.ID, "Concurrent", "fibonacci", "")
	if err != nil {
		t.Fatal(err)
	}
	room, err = r.Command(ctx, room.ID, host.ID, Command{Action: "add_round", Title: "Concurrent task"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for i := 0; i < 12; i++ {
		u := domain.User{ID: fmt.Sprint(i), Name: "User", Avatar: "violet"}
		_ = s.Update(ctx, func(st *domain.State) error { st.Users[u.ID] = u; return nil })
		if _, err = r.Join(ctx, room.ID, u.ID, false); err != nil {
			t.Fatal(err)
		}
		connect(t, r, room.ID, u.ID)
		ids = append(ids, u.ID)
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(uid string) {
			defer wg.Done()
			if _, err := r.Command(ctx, room.ID, uid, Command{Action: "vote", Value: "13", RoundID: room.ActiveRoundID}); err != nil {
				t.Error(err)
			}
		}(id)
	}
	wg.Wait()
	revealed, err := r.Command(ctx, room.ID, host.ID, Command{Action: "reveal"})
	if err != nil || revealed.Distribution["13"] != 12 {
		t.Fatalf("lost votes: %+v %v", revealed, err)
	}
}
