package domain

import "time"

func RoomVisibility(value string) string {
	if value == "" {
		return "private"
	}
	return value
}

func ValidRoomVisibility(value string) bool { return value == "private" || value == "public" }

func RoomActivity(room Room) time.Time {
	if room.LastActivityAt.IsZero() {
		return room.CreatedAt
	}
	return room.LastActivityAt
}

// Discovery never exposes member identities, phone numbers, task content or votes.
type PublicRoom struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Deck        string    `json:"deck"`
	Theme       string    `json:"theme"`
	HostName    string    `json:"host_name"`
	MemberCount int       `json:"member_count"`
	Capacity    int       `json:"capacity"`
	CreatedAt   time.Time `json:"created_at"`
}
