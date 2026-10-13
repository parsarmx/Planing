package services

import (
	"context"
	"path/filepath"
	"planningpoker/internal/config"
	"planningpoker/internal/domain"
	"planningpoker/internal/producers"
	"planningpoker/internal/realtime"
	"planningpoker/internal/repositories"
	"testing"
	"time"
)

func avatarExample() domain.AvatarConfig {
	return domain.AvatarConfig{Version: 1, Face: "round", Skin: "warm", Hair: "curls", HairColor: "brown", Expression: "wink", Glasses: "round", Shirt: "pink"}
}

func TestAvatarProfilePersistsAndBroadcasts(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := repositories.NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hub := realtime.New()
	a := NewAuth(store, producers.DevSMS{}, &config.Config{Environment: "development"}, hub)
	r := NewRooms(store, hub)
	u, token := login(t, a, "09120007700", "Avatar QA")
	room, err := r.Create(ctx, u.ID, "Avatar room", "fibonacci", "pink")
	if err != nil {
		t.Fatal(err)
	}
	events, cancel := hub.Subscribe(room.ID, u.ID)
	defer cancel()
	<-events // Initial presence notification.
	appearance := avatarExample()
	want := appearance
	updated, err := a.Profile(ctx, u.ID, u.Name, "blue", u.Role, &appearance)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-events:
	case <-time.After(time.Second):
		t.Fatal("profile change did not notify room subscribers")
	}
	// Input and returned profiles must not alias the committed repository state.
	appearance.Hair = "bald"
	updated.AvatarConfig.Skin = "rich"
	snapshot, err := r.Snapshot(ctx, room.ID, u.ID)
	if err != nil || snapshot.Members[0].AvatarConfig == nil || *snapshot.Members[0].AvatarConfig != want {
		t.Fatalf("room avatar mismatch: %+v, %v", snapshot, err)
	}
	if _, err = a.Profile(ctx, u.ID, "Changed name", "pink", "", nil); err != nil {
		t.Fatal(err)
	}
	reopened, err := repositories.NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := NewAuth(reopened, producers.DevSMS{}, &config.Config{}, realtime.New()).Resolve(ctx, token)
	if err != nil || resolved.AvatarConfig == nil || *resolved.AvatarConfig != want || resolved.Avatar != "pink" {
		t.Fatalf("avatar lost after legacy edit/reopen: %+v, %v", resolved, err)
	}
}

func TestInvalidAvatarDoesNotChangeProfile(t *testing.T) {
	a, _, _ := fixture(t)
	u, token := login(t, a, "09120007701", "Original name")
	cases := map[string]func(*domain.AvatarConfig){
		"version":    func(c *domain.AvatarConfig) { c.Version = 2 },
		"face":       func(c *domain.AvatarConfig) { c.Face = "unknown" },
		"skin":       func(c *domain.AvatarConfig) { c.Skin = "#fff" },
		"hair":       func(c *domain.AvatarConfig) { c.Hair = "https://example.test/avatar.svg" },
		"hair color": func(c *domain.AvatarConfig) { c.HairColor = "<svg>" },
		"expression": func(c *domain.AvatarConfig) { c.Expression = "unknown" },
		"glasses":    func(c *domain.AvatarConfig) { c.Glasses = "unknown" },
		"shirt":      func(c *domain.AvatarConfig) { c.Shirt = "unknown" },
		"partial":    func(c *domain.AvatarConfig) { *c = domain.AvatarConfig{Version: 1} },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			appearance := avatarExample()
			change(&appearance)
			_, err := a.Profile(context.Background(), u.ID, "Changed name", "pink", "", &appearance)
			assertCode(t, err, "invalid_avatar_config")
			current, err := a.Resolve(context.Background(), token)
			if err != nil || current.Name != u.Name || current.Avatar != u.Avatar || current.AvatarConfig != nil {
				t.Fatalf("invalid input changed stored profile: %+v, %v", current, err)
			}
		})
	}
}
