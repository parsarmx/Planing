package domain

import "time"

type User struct {
	ID           string        `json:"id"`
	Phone        string        `json:"phone"`
	Name         string        `json:"name"`
	Avatar       string        `json:"avatar"`
	AvatarConfig *AvatarConfig `json:"avatar_config,omitempty"`
	Role         string        `json:"role"`
	CreatedAt    time.Time     `json:"created_at"`
}
type OTP struct {
	Hash        string    `json:"hash"`
	Salt        string    `json:"salt"`
	ExpiresAt   time.Time `json:"expires_at"`
	SentAt      time.Time `json:"sent_at"`
	Attempts    int       `json:"attempts"`
	WindowStart time.Time `json:"window_start"`
	Sends       int       `json:"sends"`
}
type Session struct {
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
type Member struct {
	UserID   string `json:"user_id"`
	Observer bool   `json:"observer"`
}
type Round struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Votes       map[string]string `json:"votes"`
	Status      string            `json:"status"`
	Estimate    string            `json:"estimate"`
	StartedAt   time.Time         `json:"started_at"`
}
type Room struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	HostID         string    `json:"host_id"`
	Deck           string    `json:"deck"`
	Theme          string    `json:"theme"`
	Visibility     string    `json:"visibility"`
	Members        []Member  `json:"members"`
	Rounds         []Round   `json:"rounds"`
	ActiveRoundID  string    `json:"active_round_id"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
	Closed         bool      `json:"closed"`
	Revision       int64     `json:"revision"`
}
type State struct {
	Users    map[string]User    `json:"users"`
	OTPs     map[string]OTP     `json:"otps"`
	Sessions map[string]Session `json:"sessions"`
	Rooms    map[string]Room    `json:"rooms"`
}

func NewState() *State {
	return &State{Users: map[string]User{}, OTPs: map[string]OTP{}, Sessions: map[string]Session{}, Rooms: map[string]Room{}}
}
func Cards(deck string) []string {
	if deck == "tshirt" {
		return []string{"XS", "S", "M", "L", "XL", "?", "☕"}
	}
	return []string{"0", "1", "2", "3", "5", "8", "13", "21", "34", "?", "☕"}
}

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string                    { return e.Message }
func Fail(status int, code, message string) error { return &Error{status, code, message} }
