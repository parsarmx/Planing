package rest

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"planningpoker/internal/config"
	"planningpoker/internal/producers"
	"planningpoker/internal/realtime"
	"planningpoker/internal/repositories"
	"planningpoker/internal/services"
	"strings"
	"testing"
	"time"
)

type streamOutput struct {
	bytes.Buffer
	ready  chan struct{}
	broken bool
}

func (w *streamOutput) Write(b []byte) (int, error) {
	if w.broken {
		return 0, io.ErrClosedPipe
	}
	n, e := w.Buffer.Write(b)
	select {
	case w.ready <- struct{}{}:
	default:
	}
	return n, e
}
func TestStreamTerminatesAndReleasesSubscriptions(t *testing.T) {
	for _, reason := range []string{"closed", "idle", "expired", "disconnected", "shutdown"} {
		t.Run(reason, func(t *testing.T) {
			ctx := context.Background()
			store, _ := repositories.NewFile("")
			hub := realtime.New()
			cfg := &config.Config{Environment: "development", Realtime: config.RealtimeConfig{Heartbeat: 20 * time.Millisecond, IdleTimeout: 150 * time.Millisecond}}
			auth := services.NewAuth(store, producers.DevSMS{}, cfg, hub)
			rooms := services.NewRooms(store, hub)
			controller := New(auth, rooms, hub, cfg)
			ch, err := auth.Request(ctx, "09120006300")
			if err != nil {
				t.Fatal(err)
			}
			u, token, err := auth.Verify(ctx, ch.Phone, ch.DevCode)
			if err != nil {
				t.Fatal(err)
			}
			u, err = auth.Profile(ctx, u.ID, "Stream Host", "pink", "QA", nil)
			if err != nil {
				t.Fatal(err)
			}
			room, err := rooms.Create(ctx, u.ID, "Stream room", "fibonacci", "pink")
			if err != nil {
				t.Fatal(err)
			}
			events, unsubscribe := hub.Subscribe(room.ID, u.ID)
			output := &streamOutput{ready: make(chan struct{}, 1), broken: reason == "disconnected"}
			done := make(chan struct{})
			go func() {
				defer close(done)
				defer unsubscribe()
				controller.stream(bufio.NewWriter(output), nil, room.ID, u.ID, token, events)
			}()
			if reason != "disconnected" {
				select {
				case <-output.ready:
				case <-time.After(time.Second):
					t.Fatal("initial snapshot not flushed")
				}
			}
			switch reason {
			case "closed":
				_, err = rooms.Command(ctx, room.ID, u.ID, services.Command{Action: "close"})
			case "expired":
				err = auth.Logout(ctx, token)
			case "shutdown":
				hub.Stop()
			}
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				hub.Stop()
				t.Fatal("stream never closed")
			}
			if hub.Online(room.ID, u.ID) {
				t.Fatal("subscription/presence leaked after termination")
			}
			if reason == "closed" {
				if !strings.Contains(output.String(), `"closed":true`) || !strings.Contains(output.String(), "event: closed") {
					t.Fatal("final room snapshot was lost")
				}
			}
			if reason == "idle" || reason == "expired" {
				if !strings.Contains(output.String(), "event: "+reason) {
					t.Fatalf("missing terminal event: %s", output.String())
				}
			}
			if reason == "idle" {
				snapshot, e := rooms.Snapshot(ctx, room.ID, u.ID)
				if e != nil || snapshot.Closed {
					t.Fatal("idle disconnect must preserve an open room")
				}
				if !snapshot.LastActivityAt.Equal(room.LastActivityAt) {
					t.Fatal("heartbeat extended idle deadline")
				}
			}
			if errors.Is(err, io.ErrClosedPipe) {
				t.Fatal(err)
			}
		})
	}
}
