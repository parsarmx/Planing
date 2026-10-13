package domain

// AvatarConfig contains only local illustration choices, never markup or URLs.
type AvatarConfig struct {
	Version    int    `json:"version"`
	Face       string `json:"face"`
	Skin       string `json:"skin"`
	Hair       string `json:"hair"`
	HairColor  string `json:"hair_color"`
	Expression string `json:"expression"`
	Glasses    string `json:"glasses"`
	Shirt      string `json:"shirt"`
}

func (a AvatarConfig) Valid() bool {
	return a.Version == 1 &&
		avatarChoice(a.Face, "round", "square", "oval") &&
		avatarChoice(a.Skin, "porcelain", "warm", "sand", "tan", "deep", "rich") &&
		avatarChoice(a.Hair, "crop", "swoop", "bob", "curls", "bun", "bald", "scarf") &&
		avatarChoice(a.HairColor, "espresso", "brown", "gold", "copper", "silver", "pink") &&
		avatarChoice(a.Expression, "smile", "laugh", "wink", "calm") &&
		avatarChoice(a.Glasses, "none", "round", "square") &&
		avatarChoice(a.Shirt, "violet", "pink", "mint", "blue", "amber", "coral")
}

func avatarChoice(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
