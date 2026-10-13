package services

import (
	"context"
	"fmt"
	"planningpoker/internal/config"
	"planningpoker/internal/domain"
	"planningpoker/internal/producers"
	"planningpoker/internal/realtime"
	"planningpoker/internal/repositories"
	"planningpoker/internal/testutil"
	"sync"
	"testing"
)

func TestPostgresOTPConcurrencyVotesAndProfilePersistence(t *testing.T) {
	dsn := testutil.Postgres(t)
	cfg := &config.Config{Environment: "development", DatabaseURL: dsn}
	ctx := context.Background()
	s, err := repositories.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s2, err := repositories.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	hub := realtime.New()
	a := NewAuth(s, producers.DevSMS{}, cfg, hub)
	r := NewRooms(s, hub)
	r2 := NewRooms(s2, hub)
	challenge, err := a.Request(ctx, "09120006400")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	var host domain.User
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, _, err := a.Verify(ctx, challenge.Phone, challenge.DevCode)
			if err == nil {
				mu.Lock()
				host = u
				success++
				mu.Unlock()
			} else {
				assertCode(t, err, "invalid_otp")
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Fatalf("OTP consumed %d times", success)
	}
	appearance := avatarExample()
	host, err = a.Profile(ctx, host.ID, "PG Host", "pink", "QA", &appearance)
	if err != nil {
		t.Fatal(err)
	}
	room, err := r.Create(ctx, host.ID, "Postgres public", "fibonacci", "pink", "public")
	if err != nil {
		t.Fatal(err)
	}
	users := []domain.User{host}
	for i := 1; i < 20; i++ {
		u, _ := login(t, a, fmt.Sprintf("091200064%02d", i), fmt.Sprintf("Voter %d", i))
		users = append(users, u)
		if _, err = r.Join(ctx, room.ID, u.ID, false); err != nil {
			t.Fatal(err)
		}
	}
	room, err = r.Command(ctx, room.ID, host.ID, Command{Action: "add_round", Title: "Concurrent SQL votes"})
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range users {
		connect(t, r, room.ID, u.ID)
	}
	for i, u := range users {
		wg.Add(1)
		go func(i int, u domain.User) {
			defer wg.Done()
			service := r
			if i%2 == 1 {
				service = r2
			}
			if _, err := service.Command(ctx, room.ID, u.ID, Command{Action: "vote", RoundID: room.ActiveRoundID, Value: "8"}); err != nil {
				t.Error(err)
			}
		}(i, u)
	}
	wg.Wait()
	view, err := r2.Snapshot(ctx, room.ID, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Members) != 20 || view.Average != 0 || len(view.Distribution) != 0 {
		t.Fatal("pre-reveal aggregation leak")
	}
	for _, m := range view.Members {
		if !m.HasVoted || m.Vote != "" {
			t.Fatal("lost or leaked vote")
		}
	}
	view, err = r.Command(ctx, room.ID, host.ID, Command{Action: "reveal"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Distribution["8"] != 20 || view.Average != 8 || !view.Consensus {
		t.Fatalf("lost concurrent votes: %+v", view.Distribution)
	}
	if view.Members[0].AvatarConfig == nil || *view.Members[0].AvatarConfig != appearance {
		t.Fatal("avatar not persisted")
	}
	list, err := r2.Public(ctx, 12, 0)
	if err != nil || len(list) != 1 || list[0].MemberCount != 20 {
		t.Fatal("public room count incorrect", err)
	}
	if _, err = r.Command(ctx, room.ID, host.ID, Command{Action: "reset"}); err != nil {
		t.Fatal(err)
	}
	view, _ = r2.Snapshot(ctx, room.ID, host.ID)
	for _, m := range view.Members {
		if m.HasVoted {
			t.Fatal("reset did not delete SQL vote")
		}
	}
	if _, err = r.Command(ctx, room.ID, host.ID, Command{Action: "close"}); err != nil {
		t.Fatal(err)
	}
	list, _ = r2.Public(ctx, 12, 0)
	if len(list) != 0 {
		t.Fatal("closed SQL room remains public")
	}
}
