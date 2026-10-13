package services

import (
	"context"
	"encoding/json"
	"planningpoker/internal/domain"
	"strings"
	"testing"
	"time"
)

func TestPublicDiscoveryVisibilityPermissionsAndClosure(t *testing.T) {
	a, r, s := fixture(t)
	ctx := context.Background()
	host, _ := login(t, a, "09120006100", "Public Host")
	guest, _ := login(t, a, "09120006101", "Guest")
	private, err := r.Create(ctx, host.ID, "Secret room", "fibonacci", "pink")
	if err != nil {
		t.Fatal(err)
	}
	public, err := r.Create(ctx, host.ID, "Community room", "fibonacci", "blue", "public")
	if err != nil {
		t.Fatal(err)
	}
	connect(t, r, public.ID, host.ID)
	list, err := r.Public(ctx, 12, 0)
	if err != nil || len(list) != 1 || list[0].ID != public.ID || list[0].MemberCount != 1 {
		t.Fatalf("unexpected discovery: %+v %v", list, err)
	}
	b, _ := json.Marshal(list)
	for _, secret := range []string{host.Phone, host.ID, "members", "votes", "active_round_id", private.ID} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("discovery leaked %s", secret)
		}
	}
	_, err = r.Snapshot(ctx, public.ID, guest.ID)
	assertCode(t, err, "not_member")
	if _, err = r.Join(ctx, public.ID, guest.ID, false); err != nil {
		t.Fatal(err)
	}
	_, err = r.Command(ctx, public.ID, guest.ID, Command{Action: "set_visibility", Value: "private"})
	assertCode(t, err, "host_required")
	_, err = r.Command(ctx, public.ID, host.ID, Command{Action: "set_visibility", Value: "invalid"})
	assertCode(t, err, "invalid_visibility")
	if _, err = r.Command(ctx, public.ID, host.ID, Command{Action: "set_visibility", Value: "private"}); err != nil {
		t.Fatal(err)
	}
	list, _ = r.Public(ctx, 12, 0)
	if len(list) != 0 {
		t.Fatal("private room is discoverable")
	}
	if _, err = r.Join(ctx, private.ID, guest.ID, false); err != nil {
		t.Fatal("private invitation no longer works", err)
	}
	if _, err = r.Command(ctx, private.ID, host.ID, Command{Action: "set_visibility", Value: "public"}); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Command(ctx, private.ID, host.ID, Command{Action: "close"}); err != nil {
		t.Fatal(err)
	}
	list, _ = r.Public(ctx, 12, 0)
	if len(list) != 0 {
		t.Fatal("closed room remains discoverable")
	}
	_, err = r.Command(ctx, private.ID, host.ID, Command{Action: "set_visibility", Value: "public"})
	assertCode(t, err, "room_closed")
	_ = s.Update(ctx, func(st *domain.State) error {
		room := st.Rooms[public.ID]
		room.Visibility = ""
		st.Rooms[room.ID] = room
		return nil
	})
	snapshot, _ := r.Snapshot(ctx, public.ID, host.ID)
	if snapshot.Visibility != "private" {
		t.Fatal("legacy room must default to private")
	}
}
func TestRoomReadsDoNotRenewIdleDeadline(t *testing.T) {
	a, r, _ := fixture(t)
	ctx := context.Background()
	host, _ := login(t, a, "09120006102", "Idle Host")
	now := time.Now()
	r.now = func() time.Time { return now }
	room, err := r.Create(ctx, host.ID, "Idle room", "fibonacci", "pink")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(25 * time.Hour)
	snapshot, _ := r.Snapshot(ctx, room.ID, host.ID)
	_, _ = r.List(ctx, host.ID)
	_, _ = r.Public(ctx, 12, 0)
	if !snapshot.LastActivityAt.Equal(room.LastActivityAt) {
		t.Fatal("read renewed activity")
	}
	if err = r.Touch(ctx, room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = r.Snapshot(ctx, room.ID, host.ID)
	if !snapshot.LastActivityAt.Equal(now) {
		t.Fatal("manual reopening did not renew activity")
	}
	_, err = r.Command(ctx, room.ID, host.ID, Command{Action: "set_theme", Value: "invalid"})
	assertCode(t, err, "invalid_theme")
	now = now.Add(time.Hour)
	snapshot, _ = r.Snapshot(ctx, room.ID, host.ID)
	if snapshot.LastActivityAt.Equal(now) {
		t.Fatal("failed command renewed activity")
	}
}
