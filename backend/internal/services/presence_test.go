package services

import (
	"context"
	"testing"
)

func TestLiveVotesFollowLastConnectionAndHistorySurvives(t *testing.T) {
	a, r, _ := fixture(t)
	ctx := context.Background()
	host, _ := login(t, a, "09120006200", "Host")
	guest, _ := login(t, a, "09120006201", "Guest")
	room, err := r.Create(ctx, host.ID, "Presence", "fibonacci", "", "public")
	if err != nil {
		t.Fatal(err)
	}
	connect(t, r, room.ID, host.ID)
	if _, err = r.Join(ctx, room.ID, guest.ID, false); err != nil {
		t.Fatal(err)
	}
	room, err = r.Command(ctx, room.ID, host.ID, Command{Action: "add_round", Title: "Presence task"})
	if err != nil {
		t.Fatal(err)
	}
	vote := Command{Action: "vote", Value: "8", RoundID: room.ActiveRoundID}
	_, err = r.Command(ctx, room.ID, guest.ID, vote)
	assertCode(t, err, "disconnected")
	first := connect(t, r, room.ID, guest.ID)
	last := connect(t, r, room.ID, guest.ID)
	if _, err = r.Command(ctx, room.ID, guest.ID, vote); err != nil {
		t.Fatal(err)
	}
	first()
	if !r.hub.Online(room.ID, guest.ID) {
		t.Fatal("closing one tab removed another active tab")
	}
	last()
	if r.hub.Online(room.ID, guest.ID) {
		t.Fatal("last tab did not release presence")
	}
	_, err = r.Command(ctx, room.ID, guest.ID, vote)
	assertCode(t, err, "disconnected")
	_, err = r.Command(ctx, room.ID, host.ID, Command{Action: "reveal"})
	assertCode(t, err, "no_votes")
	if _, err = r.Command(ctx, room.ID, host.ID, Command{Action: "vote", Value: "5", RoundID: room.ActiveRoundID}); err != nil {
		t.Fatal(err)
	}
	room, err = r.Command(ctx, room.ID, host.ID, Command{Action: "reveal"})
	if err != nil || room.Average != 5 || room.Distribution["8"] != 0 || room.Consensus {
		t.Fatalf("offline vote counted: %+v %v", room, err)
	}
	list, err := r.Public(ctx, 12, 0)
	if err != nil || list[0].MemberCount != 1 {
		t.Fatal("offline member occupied capacity", err)
	}
	back := connect(t, r, room.ID, guest.ID)
	room, err = r.Snapshot(ctx, room.ID, guest.ID)
	if err != nil || room.MyVote != "8" || room.Distribution["8"] != 1 || len(room.Members) != 2 {
		t.Fatalf("reconnection lost membership or vote: %+v %v", room, err)
	}
	room, err = r.Command(ctx, room.ID, host.ID, Command{Action: "estimate", Value: "5"})
	if err != nil {
		t.Fatal(err)
	}
	back()
	room, err = r.Snapshot(ctx, room.ID, host.ID)
	if err != nil || room.Distribution["8"] != 1 {
		t.Fatal("completed results changed on disconnect", err)
	}
}
