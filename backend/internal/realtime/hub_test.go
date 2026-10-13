package realtime

import (
	"fmt"
	"testing"
)

func TestConnectionLimitPresenceAndShutdown(t *testing.T) {
	h := New()
	var cleanup []func()
	for i := 0; i < 6; i++ {
		ch, done := h.Subscribe("room", "user")
		if ch == nil {
			t.Fatal("connection rejected")
		}
		cleanup = append(cleanup, done)
	}
	if ch, _ := h.Subscribe("room", "user"); ch != nil {
		t.Fatal("excess connections accepted")
	}
	if !h.Online("room", "user") {
		t.Fatal("missing presence")
	}
	cleanup[0]()
	ch, done := h.Subscribe("another", "user")
	if ch == nil {
		t.Fatal("released slot was not reusable")
	}
	done()
	for _, fn := range cleanup {
		fn()
	}
	if h.Online("room", "user") {
		t.Fatal("presence was not released")
	}
	h.Stop()
	h.Stop()
	select {
	case <-h.Done():
	default:
		t.Fatal("shutdown not signalled")
	}
	if ch, _ := h.Subscribe("room", "new"); ch != nil {
		t.Fatal("subscription after shutdown")
	}
}

func TestRoomCapacityCountsPeopleRatherThanTabs(t *testing.T) {
	h := New()
	var release []func()
	for i := 0; i < 20; i++ {
		ch, cancel := h.Subscribe("room", fmt.Sprint(i))
		if ch == nil {
			t.Fatal("capacity rejected too early")
		}
		release = append(release, cancel)
	}
	if h.Count("room") != 20 {
		t.Fatal("wrong active count")
	}
	ch, same := h.Subscribe("room", "0")
	if ch == nil || h.Count("room") != 20 {
		t.Fatal("another tab occupied a seat")
	}
	if ch, _ := h.Subscribe("room", "new"); ch != nil {
		t.Fatal("over capacity")
	}
	release[0]()
	if ch, _ := h.Subscribe("room", "new"); ch != nil {
		t.Fatal("seat released while another tab is active")
	}
	same()
	ch, done := h.Subscribe("room", "new")
	if ch == nil {
		t.Fatal("last disconnect did not release seat")
	}
	done()
	for _, cancel := range release {
		cancel()
	}
}
