package services

import (
	"context"
	"path/filepath"
	"planningpoker/internal/domain"
	"planningpoker/internal/realtime"
	"planningpoker/internal/repositories"
	"testing"
)

func TestRoomThemesCreatorPermissionValidationAndBroadcast(t *testing.T) {
	ctx := context.Background()
	a, r, s := fixture(t)
	host, _ := login(t, a, "09120006001", "Theme creator")
	guest, _ := login(t, a, "09120006002", "Theme guest")
	outsider, _ := login(t, a, "09120006003", "Theme outsider")
	for _, theme := range []string{"violet", "pink", "blue", "mint", "amber", "coral"} {
		room, err := r.Create(ctx, host.ID, "Theme "+theme, "fibonacci", theme)
		if err != nil || room.Theme != theme {
			t.Fatalf("create %s: %+v %v", theme, room, err)
		}
	}
	_, err := r.Create(ctx, host.ID, "Invalid theme", "fibonacci", "unknown")
	assertCode(t, err, "invalid_theme")
	room, err := r.Create(ctx, host.ID, "Room colors", "fibonacci", "")
	if err != nil {
		t.Fatal(err)
	}
	if room.Theme != "violet" {
		t.Fatal("default theme changed")
	}
	if _, err = r.Join(ctx, room.ID, guest.ID, false); err != nil {
		t.Fatal(err)
	}
	room, err = r.Command(ctx, room.ID, host.ID, Command{Action: "add_round", Title: "Theme during voting"})
	if err != nil {
		t.Fatal(err)
	}
	connect(t, r, room.ID, guest.ID)
	room, err = r.Command(ctx, room.ID, guest.ID, Command{Action: "vote", Value: "8", RoundID: room.ActiveRoundID})
	if err != nil {
		t.Fatal(err)
	}
	before := room.Revision
	_, err = r.Command(ctx, room.ID, guest.ID, Command{Action: "set_theme", Value: "pink"})
	assertCode(t, err, "host_required")
	_, err = r.Command(ctx, room.ID, outsider.ID, Command{Action: "set_theme", Value: "pink"})
	assertCode(t, err, "not_member")
	_, err = r.Command(ctx, room.ID, host.ID, Command{Action: "set_theme", Value: "unknown"})
	assertCode(t, err, "invalid_theme")
	_, err = r.Command(ctx, room.ID, host.ID, Command{Action: "set_theme", Value: ""})
	assertCode(t, err, "invalid_theme")
	unchanged, err := r.Snapshot(ctx, room.ID, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Theme != "violet" || unchanged.Revision != before {
		t.Fatal("rejected command mutated room")
	}
	ch, unsub := r.hub.Subscribe(room.ID, guest.ID)
	defer unsub()
	<-ch
	updated, err := r.Command(ctx, room.ID, host.ID, Command{Action: "set_theme", Value: "pink"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Theme != "pink" || updated.Revision != before+1 {
		t.Fatal("theme/revision not updated")
	}
	select {
	case <-ch:
	default:
		t.Fatal("theme change was not broadcast")
	}
	guestView, err := r.Snapshot(ctx, room.ID, guest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if guestView.Theme != "pink" || guestView.MyVote != "8" || guestView.ActiveRoundID != room.ActiveRoundID || guestView.Rounds[0].Status != "voting" {
		t.Fatal("theme changed voting state")
	}
	if len(guestView.Distribution) != 0 || guestView.Average != 0 {
		t.Fatal("theme leaked vote statistics")
	}
	for _, m := range guestView.Members {
		if m.Vote != "" {
			t.Fatal("theme leaked hidden votes")
		}
	}
	_ = s.View(ctx, func(st *domain.State) error {
		if st.Rooms[room.ID].Theme != "pink" {
			t.Fatal("theme not stored")
		}
		return nil
	})
	if _, err = r.Command(ctx, room.ID, host.ID, Command{Action: "close"}); err != nil {
		t.Fatal(err)
	}
	room, err = r.Command(ctx, room.ID, host.ID, Command{Action: "set_theme", Value: "mint"})
	if err != nil || room.Theme != "mint" || !room.Closed {
		t.Fatal("creator could not recolor closed room")
	}
	_, err = r.Command(ctx, room.ID, guest.ID, Command{Action: "vote", Value: "5", RoundID: room.ActiveRoundID})
	assertCode(t, err, "room_closed")
}

func TestRoomThemePersistenceAndLegacyDefault(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := repositories.NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Update(ctx, func(st *domain.State) error { st.Users["host"] = domain.User{ID: "host", Name: "Creator"}; return nil })
	r := NewRooms(s, realtime.New())
	room, err := r.Create(ctx, "host", "Persist colors", "fibonacci", "pink")
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := repositories.NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	view, err := NewRooms(reopened, realtime.New()).Snapshot(ctx, room.ID, "host")
	if err != nil || view.Theme != "pink" {
		t.Fatal("theme did not survive reopen")
	}
	_ = reopened.Update(ctx, func(st *domain.State) error {
		legacy := st.Rooms[room.ID]
		legacy.Theme = ""
		st.Rooms[room.ID] = legacy
		return nil
	})
	legacy, err := NewRooms(reopened, realtime.New()).Snapshot(ctx, room.ID, "host")
	if err != nil || legacy.Theme != domain.DefaultRoomTheme {
		t.Fatal("legacy rooms lost original theme")
	}
}
