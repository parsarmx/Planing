package services

import (
	"context"
	"errors"
	"planningpoker/internal/domain"
	"planningpoker/internal/realtime"
	"planningpoker/internal/repositories"
	"planningpoker/pkg"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Rooms struct {
	store repositories.Store
	hub   *realtime.Hub
	now   func() time.Time
}

func NewRooms(s repositories.Store, h *realtime.Hub) *Rooms { return &Rooms{s, h, time.Now} }

type Participant struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Avatar       string               `json:"avatar"`
	AvatarConfig *domain.AvatarConfig `json:"avatar_config,omitempty"`
	Role         string               `json:"role"`
	Observer     bool                 `json:"observer"`
	Online       bool                 `json:"online"`
	HasVoted     bool                 `json:"has_voted"`
	Vote         string               `json:"vote,omitempty"`
}
type RoundView struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Estimate    string    `json:"estimate"`
	StartedAt   time.Time `json:"started_at"`
}
type Snapshot struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	HostID         string         `json:"host_id"`
	Deck           string         `json:"deck"`
	Theme          string         `json:"theme"`
	Visibility     string         `json:"visibility"`
	LastActivityAt time.Time      `json:"last_activity_at"`
	Cards          []string       `json:"cards"`
	Members        []Participant  `json:"members"`
	Rounds         []RoundView    `json:"rounds"`
	ActiveRoundID  string         `json:"active_round_id"`
	CreatedAt      time.Time      `json:"created_at"`
	Closed         bool           `json:"closed"`
	Revision       int64          `json:"revision"`
	MyVote         string         `json:"my_vote"`
	Average        float64        `json:"average"`
	Consensus      bool           `json:"consensus"`
	Distribution   map[string]int `json:"distribution"`
}

func member(r domain.Room, id string) bool {
	for _, m := range r.Members {
		if m.UserID == id {
			return true
		}
	}
	return false
}
func (r *Rooms) Snapshot(ctx context.Context, id, user string) (*Snapshot, error) {
	var out *Snapshot
	err := r.store.Read(ctx, func(tx repositories.Transaction) error {
		room, err := tx.Room(id)
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.Fail(404, "room_not_found", "اتاق پیدا نشد. لینک دعوت را بررسی کنید.")
		}
		if err != nil {
			return err
		}
		if !member(room, user) {
			return domain.Fail(403, "not_member", "ابتدا به این اتاق بپیوندید.")
		}
		out, err = r.presentTransaction(tx, room, user)
		return err
	})
	return out, err
}
func (r *Rooms) present(s *domain.State, room domain.Room, user string) *Snapshot {
	out := &Snapshot{ID: room.ID, Name: room.Name, HostID: room.HostID, Deck: room.Deck, Theme: domain.RoomTheme(room.Theme), Visibility: domain.RoomVisibility(room.Visibility), LastActivityAt: domain.RoomActivity(room), Cards: domain.Cards(room.Deck), Members: []Participant{}, Rounds: []RoundView{}, ActiveRoundID: room.ActiveRoundID, CreatedAt: room.CreatedAt, Closed: room.Closed, Revision: room.Revision, Distribution: map[string]int{}}
	var current domain.Round
	for _, round := range room.Rounds {
		out.Rounds = append(out.Rounds, RoundView{round.ID, round.Title, round.Description, round.Status, round.Estimate, round.StartedAt})
		if round.ID == room.ActiveRoundID {
			current = round
		}
	}
	revealed := current.Status == "revealed" || current.Status == "completed"
	out.MyVote = current.Votes[user]
	var sum float64
	var count, voteCount, voterCount int
	first := ""
	same := true
	for _, m := range room.Members {
		u := s.Users[m.UserID]
		v := current.Votes[m.UserID]
		p := Participant{ID: u.ID, Name: u.Name, Avatar: u.Avatar, AvatarConfig: u.AvatarConfig, Role: u.Role, Observer: m.Observer, Online: r.hub.Online(room.ID, u.ID), HasVoted: v != ""}
		// Completed estimates retain their historical votes. Live rounds count only connected voters.
		eligible := !m.Observer && (current.Status == "completed" || room.Closed || p.Online)
		if eligible {
			voterCount++
		}
		if revealed {
			p.Vote = v
		}
		out.Members = append(out.Members, p)
		if revealed && eligible && v != "" {
			out.Distribution[v]++
			voteCount++
			if first == "" {
				first = v
			} else if first != v {
				same = false
			}
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				sum += f
				count++
			}
		}
	}
	if count > 0 {
		out.Average = sum / float64(count)
	}
	out.Consensus = voteCount > 1 && voteCount == voterCount && same && first != "?" && first != "☕"
	return out
}
func (r *Rooms) List(ctx context.Context, user string) ([]*Snapshot, error) {
	out := []*Snapshot{}
	err := r.store.Read(ctx, func(tx repositories.Transaction) error {
		rooms, err := tx.UserRooms(user)
		if err != nil {
			return err
		}
		for _, room := range rooms {
			view, err := r.presentTransaction(tx, room, user)
			if err != nil {
				return err
			}
			out = append(out, view)
		}
		return nil
	})
	return out, err
}
func (r *Rooms) Create(ctx context.Context, user, name, deck, theme string, requestedVisibility ...string) (*Snapshot, error) {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 80 {
		return nil, domain.Fail(400, "invalid_room", "نام اتاق باید بین ۲ تا ۸۰ کاراکتر باشد.")
	}
	if deck != "fibonacci" && deck != "tshirt" {
		return nil, domain.Fail(400, "invalid_deck", "دسته کارت معتبر نیست.")
	}
	theme = domain.RoomTheme(theme)
	if !domain.ValidRoomTheme(theme) {
		return nil, domain.Fail(400, "invalid_theme", "تم اتاق معتبر نیست.")
	}
	visibility := "private"
	if len(requestedVisibility) > 0 {
		visibility = domain.RoomVisibility(requestedVisibility[0])
	}
	if !domain.ValidRoomVisibility(visibility) {
		return nil, domain.Fail(400, "invalid_visibility", "نوع دسترسی اتاق معتبر نیست.")
	}
	id := pkg.ID()
	err := r.store.Write(ctx, "host:"+user, func(tx repositories.Transaction) error {
		u, err := tx.User(user)
		if err != nil {
			return err
		}
		if u.Name == "" {
			return domain.Fail(400, "profile_required", "ابتدا پروفایل خود را تکمیل کنید.")
		}
		n, err := tx.CountHostedRooms(user)
		if err != nil {
			return err
		}
		if n >= 30 {
			return domain.Fail(400, "room_limit", "حداکثر ۳۰ اتاق فعال می‌توانید داشته باشید.")
		}
		now := r.now()
		return tx.SaveRoom(domain.Room{ID: id, Name: name, HostID: user, Deck: deck, Theme: theme, Visibility: visibility, Members: []domain.Member{{UserID: user}}, Rounds: []domain.Round{}, CreatedAt: now, LastActivityAt: now, Revision: 1})
	})
	if err != nil {
		return nil, err
	}
	return r.Snapshot(ctx, id, user)
}
func (r *Rooms) Join(ctx context.Context, id, user string, observer bool) (*Snapshot, error) {
	err := r.store.Write(ctx, "room:"+id, func(tx repositories.Transaction) error {
		room, err := tx.Room(id)
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.Fail(404, "room_not_found", "اتاق پیدا نشد.")
		}
		if err != nil {
			return err
		}
		if room.Closed {
			return domain.Fail(409, "room_closed", "این جلسه پایان یافته است.")
		}
		if member(room, user) {
			room.LastActivityAt = r.now()
			return tx.SaveRoom(room)
		}
		u, err := tx.User(user)
		if err != nil {
			return err
		}
		if u.Name == "" {
			return domain.Fail(400, "profile_required", "ابتدا پروفایل خود را تکمیل کنید.")
		}
		if r.hub.Count(id) >= 20 {
			return domain.Fail(409, "room_full", "ظرفیت اتاق ۲۰ نفر است.")
		}
		room.Members = append(room.Members, domain.Member{UserID: user, Observer: observer})
		room.Revision++
		room.LastActivityAt = r.now()
		return tx.SaveRoom(room)
	})
	if err != nil {
		return nil, err
	}
	r.hub.Publish(id)
	return r.Snapshot(ctx, id, user)
}

type Command struct {
	Action      string `json:"action"`
	Value       string `json:"value"`
	RoundID     string `json:"round_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (r *Rooms) Command(ctx context.Context, id, user string, c Command) (*Snapshot, error) {
	err := r.store.Write(ctx, "room:"+id, func(tx repositories.Transaction) error {
		room, err := tx.Room(id)
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.Fail(404, "room_not_found", "اتاق پیدا نشد.")
		}
		if err != nil {
			return err
		}
		if !member(room, user) {
			return domain.Fail(403, "not_member", "عضو این اتاق نیستید.")
		}
		if room.Closed && c.Action != "set_theme" {
			return domain.Fail(409, "room_closed", "این جلسه پایان یافته است.")
		}
		if c.Action != "vote" && room.HostID != user {
			return domain.Fail(403, "host_required", "این کار فقط برای گرداننده اتاق مجاز است.")
		}
		idx := -1
		for i, round := range room.Rounds {
			if round.ID == room.ActiveRoundID {
				idx = i
			}
		}
		switch c.Action {
		case "set_visibility":
			if !domain.ValidRoomVisibility(c.Value) {
				return domain.Fail(400, "invalid_visibility", "نوع دسترسی اتاق معتبر نیست.")
			}
			room.Visibility = c.Value
		case "set_theme":
			if !domain.ValidRoomTheme(c.Value) {
				return domain.Fail(400, "invalid_theme", "تم اتاق معتبر نیست.")
			}
			room.Theme = c.Value
		case "add_round":
			title := strings.TrimSpace(c.Title)
			if utf8.RuneCountInString(title) < 2 || utf8.RuneCountInString(title) > 160 || utf8.RuneCountInString(c.Description) > 3000 {
				return domain.Fail(400, "invalid_round", "عنوان تسک باید بین ۲ تا ۱۶۰ کاراکتر باشد؛ توضیحات حداکثر ۳۰۰۰ کاراکتر.")
			}
			if len(room.Rounds) >= 200 {
				return domain.Fail(400, "round_limit", "حداکثر ۲۰۰ تسک در هر اتاق مجاز است.")
			}
			round := domain.Round{ID: pkg.ID(), Title: title, Description: strings.TrimSpace(c.Description), Votes: map[string]string{}, Status: "pending"}
			room.Rounds = append(room.Rounds, round)
			if room.ActiveRoundID == "" {
				room.ActiveRoundID = round.ID
				room.Rounds[len(room.Rounds)-1].Status = "voting"
				room.Rounds[len(room.Rounds)-1].StartedAt = r.now()
			}
		case "start_round":
			target := -1
			for i, round := range room.Rounds {
				if round.ID == c.RoundID {
					target = i
				}
			}
			if target < 0 {
				return domain.Fail(404, "round_not_found", "تسک پیدا نشد.")
			}
			if room.Rounds[target].Status != "pending" {
				return domain.Fail(409, "round_started", "این تسک قبلاً شروع شده است.")
			}
			if idx >= 0 && room.Rounds[idx].Status != "completed" {
				return domain.Fail(409, "active_round", "ابتدا امتیاز تسک جاری را ثبت کنید.")
			}
			room.ActiveRoundID = c.RoundID
			room.Rounds[target].Status = "voting"
			room.Rounds[target].StartedAt = r.now()
		case "vote":
			if idx < 0 || room.Rounds[idx].Status != "voting" {
				return domain.Fail(409, "not_voting", "این دور برای رأی‌گیری باز نیست.")
			}
			if c.RoundID != room.ActiveRoundID {
				return domain.Fail(409, "stale_round", "تسک جاری تغییر کرده است.")
			}
			for _, m := range room.Members {
				if m.UserID == user && m.Observer {
					return domain.Fail(403, "observer", "ناظر نمی‌تواند رأی بدهد.")
				}
			}
			valid := c.Value == ""
			for _, card := range domain.Cards(room.Deck) {
				if card == c.Value {
					valid = true
				}
			}
			if !valid {
				return domain.Fail(400, "invalid_card", "کارت معتبر نیست.")
			}
			if !r.hub.Online(id, user) {
				return domain.Fail(409, "disconnected", "برای رأی دادن، ابتدا دوباره به اتاق متصل شوید.")
			}
			if c.Value == "" {
				delete(room.Rounds[idx].Votes, user)
			} else {
				room.Rounds[idx].Votes[user] = c.Value
			}
		case "reveal":
			if idx < 0 || room.Rounds[idx].Status != "voting" {
				return domain.Fail(409, "not_voting", "دور قابل نمایش نیست.")
			}
			hasActiveVote := false
			for _, participant := range room.Members {
				if !participant.Observer && r.hub.Online(id, participant.UserID) && room.Rounds[idx].Votes[participant.UserID] != "" {
					hasActiveVote = true
					break
				}
			}
			if !hasActiveVote {
				return domain.Fail(409, "no_votes", "هنوز کسی رأی نداده است.")
			}
			room.Rounds[idx].Status = "revealed"
		case "reset":
			if idx < 0 || room.Rounds[idx].Status == "completed" {
				return domain.Fail(409, "invalid_round", "دور قابل شروع مجدد نیست.")
			}
			room.Rounds[idx].Votes = map[string]string{}
			room.Rounds[idx].Status = "voting"
			room.Rounds[idx].StartedAt = r.now()
		case "estimate":
			if idx < 0 || room.Rounds[idx].Status != "revealed" {
				return domain.Fail(409, "not_revealed", "ابتدا کارت‌ها را نمایش دهید.")
			}
			valid := false
			for _, card := range domain.Cards(room.Deck) {
				if card == c.Value && card != "?" && card != "☕" {
					valid = true
				}
			}
			if !valid {
				return domain.Fail(400, "invalid_estimate", "امتیاز نهایی معتبر نیست.")
			}
			room.Rounds[idx].Estimate = c.Value
			room.Rounds[idx].Status = "completed"
		case "close":
			room.Closed = true
		default:
			return domain.Fail(400, "invalid_action", "عملیات معتبر نیست.")
		}
		room.Revision++
		room.LastActivityAt = r.now()
		return tx.SaveRoom(room)
	})
	if err != nil {
		return nil, err
	}
	r.hub.Publish(id)
	return r.Snapshot(ctx, id, user)
}

func (r *Rooms) presentTransaction(tx repositories.Transaction, room domain.Room, user string) (*Snapshot, error) {
	ids := make([]string, 0, len(room.Members))
	for _, m := range room.Members {
		ids = append(ids, m.UserID)
	}
	users, err := tx.Users(ids)
	if err != nil {
		return nil, err
	}
	return r.present(&domain.State{Users: users}, room, user), nil
}
func (r *Rooms) Public(ctx context.Context, limit, offset int) ([]domain.PublicRoom, error) {
	if limit < 1 || limit > 50 || offset < 0 || offset > 100000 {
		return nil, domain.Fail(400, "invalid_page", "صفحه درخواستی معتبر نیست.")
	}
	var out []domain.PublicRoom
	err := r.store.Read(ctx, func(tx repositories.Transaction) error { var e error; out, e = tx.PublicRooms(limit, offset); return e })
	for i := range out {
		out[i].MemberCount = r.hub.Count(out[i].ID)
	}
	return out, err
}

// Opening a room is activity. Heartbeats and read-only snapshots never renew it.
func (r *Rooms) Touch(ctx context.Context, id, user string) error {
	return r.store.Write(ctx, "room:"+id, func(tx repositories.Transaction) error {
		room, err := tx.Room(id)
		if err != nil {
			return err
		}
		if !member(room, user) {
			return domain.Fail(403, "not_member", "عضو این اتاق نیستید.")
		}
		if room.Closed {
			return nil
		}
		room.LastActivityAt = r.now()
		return tx.SaveRoom(room)
	})
}
