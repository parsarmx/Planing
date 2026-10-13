package domain

const DefaultRoomTheme = "violet"

func ValidRoomTheme(theme string) bool {
	switch theme {
	case "violet", "pink", "blue", "mint", "amber", "coral":
		return true
	default:
		return false
	}
}

// Rooms saved before color themes were introduced keep their original appearance.
func RoomTheme(theme string) string {
	if theme == "" {
		return DefaultRoomTheme
	}
	return theme
}
