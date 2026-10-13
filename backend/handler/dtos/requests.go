package dtos

import "planningpoker/internal/domain"

type OTPRequest struct {
	Phone string `json:"phone"`
}
type OTPVerify struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}
type Profile struct {
	Name         string               `json:"name"`
	Avatar       string               `json:"avatar"`
	AvatarConfig *domain.AvatarConfig `json:"avatar_config"`
	Role         string               `json:"role"`
}
type CreateRoom struct {
	Name       string `json:"name"`
	Deck       string `json:"deck"`
	Theme      string `json:"theme"`
	Visibility string `json:"visibility"`
}
type JoinRoom struct {
	Observer bool `json:"observer"`
}
