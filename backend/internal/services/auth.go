package services

import (
	"context"
	"crypto/subtle"
	"errors"
	"planningpoker/internal/config"
	"planningpoker/internal/domain"
	"planningpoker/internal/producers"
	"planningpoker/internal/realtime"
	"planningpoker/internal/repositories"
	"planningpoker/pkg"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type Auth struct {
	store  repositories.Store
	sms    producers.SMS
	config *config.Config
	hub    *realtime.Hub
}

func NewAuth(s repositories.Store, sms producers.SMS, c *config.Config, h *realtime.Hub) *Auth {
	return &Auth{s, sms, c, h}
}

var phonePattern = regexp.MustCompile(`^\+989[0-9]{9}$`)

func NormalizePhone(v string) (string, error) {
	v = strings.TrimSpace(v)
	v = strings.NewReplacer("۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9", "٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9", " ", "", "-", "").Replace(v)
	if strings.HasPrefix(v, "09") {
		v = "+98" + v[1:]
	} else if strings.HasPrefix(v, "0098") {
		v = "+" + v[2:]
	}
	if !phonePattern.MatchString(v) {
		return "", domain.Fail(400, "invalid_phone", "شماره موبایل معتبر ایرانی وارد کنید.")
	}
	return v, nil
}

type Challenge struct {
	Phone      string `json:"phone"`
	ExpiresIn  int    `json:"expires_in"`
	RetryAfter int    `json:"retry_after"`
	DevCode    string `json:"dev_code,omitempty"`
}

func (a *Auth) Request(ctx context.Context, input string) (*Challenge, error) {
	phone, err := NormalizePhone(input)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	code, salt := pkg.Code(), pkg.Token()
	err = a.store.Write(ctx, "phone:"+phone, func(tx repositories.Transaction) error {
		if err := tx.PruneAuth(now); err != nil {
			return err
		}
		o, err := tx.OTP(phone)
		if err != nil && !errors.Is(err, repositories.ErrNotFound) {
			return err
		}
		if now.Sub(o.SentAt) < time.Minute {
			return domain.Fail(429, "resend_wait", "برای ارسال دوباره یک دقیقه صبر کنید.")
		}
		if now.Sub(o.WindowStart) >= time.Hour {
			o.WindowStart = now
			o.Sends = 0
		}
		if o.Sends >= 5 {
			return domain.Fail(429, "otp_limit", "تعداد درخواست‌ها زیاد است. یک ساعت دیگر تلاش کنید.")
		}
		return tx.SaveOTP(phone, domain.OTP{Hash: pkg.Hash(salt + code), Salt: salt, ExpiresAt: now.Add(3 * time.Minute), SentAt: now, WindowStart: o.WindowStart, Sends: o.Sends + 1})
	})
	if err != nil {
		return nil, err
	}
	if err = a.sms.Send(ctx, phone, code); err != nil {
		_ = a.store.Write(ctx, "phone:"+phone, func(tx repositories.Transaction) error {
			o, err := tx.OTP(phone)
			if err != nil {
				return err
			}
			if o.Salt == salt {
				o.Hash = ""
				return tx.SaveOTP(phone, o)
			}
			return nil
		})
		return nil, domain.Fail(502, "sms_unavailable", "ارسال پیامک انجام نشد. کمی بعد دوباره تلاش کنید.")
	}
	ch := &Challenge{Phone: phone, ExpiresIn: 180, RetryAfter: 60}
	if a.config.Environment == "development" {
		ch.DevCode = code
	}
	return ch, nil
}
func (a *Auth) Verify(ctx context.Context, input, code string) (domain.User, string, error) {
	phone, err := NormalizePhone(input)
	if err != nil {
		return domain.User{}, "", err
	}
	code = strings.TrimSpace(code)
	var user domain.User
	token := pkg.Token()
	now := time.Now()
	err = a.store.Write(ctx, "phone:"+phone, func(tx repositories.Transaction) error {
		o, err := tx.OTP(phone)
		if err != nil && !errors.Is(err, repositories.ErrNotFound) {
			return err
		}
		if errors.Is(err, repositories.ErrNotFound) || o.Hash == "" || now.After(o.ExpiresAt) || o.Attempts >= 5 {
			return domain.Fail(401, "invalid_otp", "کد نامعتبر یا منقضی شده است. کد جدید بگیرید.")
		}
		o.Attempts++
		if err := tx.SaveOTP(phone, o); err != nil {
			return err
		}
		if len(code) != 6 || subtle.ConstantTimeCompare([]byte(o.Hash), []byte(pkg.Hash(o.Salt+code))) != 1 {
			return domain.Fail(401, "invalid_otp", "کد واردشده درست نیست.")
		}
		o.Hash = ""
		if err := tx.SaveOTP(phone, o); err != nil {
			return err
		}
		user, err = tx.UserByPhone(phone)
		if err != nil && !errors.Is(err, repositories.ErrNotFound) {
			return err
		}
		if errors.Is(err, repositories.ErrNotFound) {
			user = domain.User{ID: pkg.ID(), Phone: phone, Avatar: "violet", CreatedAt: now}
			if err := tx.SaveUser(user); err != nil {
				return err
			}
		}
		return tx.SaveSession(pkg.Hash(token), domain.Session{UserID: user.ID, ExpiresAt: now.Add(7 * 24 * time.Hour)})
	})
	return user, token, err
}
func (a *Auth) Resolve(ctx context.Context, token string) (domain.User, error) {
	var u domain.User
	err := a.store.Read(ctx, func(tx repositories.Transaction) error {
		session, err := tx.Session(pkg.Hash(token))
		if err != nil && !errors.Is(err, repositories.ErrNotFound) {
			return err
		}
		if errors.Is(err, repositories.ErrNotFound) || time.Now().After(session.ExpiresAt) {
			return domain.Fail(401, "unauthenticated", "برای ادامه وارد حساب شوید.")
		}
		u, err = tx.User(session.UserID)
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.Fail(401, "unauthenticated", "حساب معتبر نیست.")
		}
		return err
	})
	return u, err
}
func (a *Auth) Logout(ctx context.Context, token string) error {
	return a.store.Write(ctx, "session:"+pkg.Hash(token), func(tx repositories.Transaction) error { return tx.DeleteSession(pkg.Hash(token)) })
}
func (a *Auth) Profile(ctx context.Context, id, name, avatar, role string, appearance *domain.AvatarConfig) (domain.User, error) {
	name = strings.TrimSpace(name)
	role = strings.TrimSpace(role)
	if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 50 || utf8.RuneCountInString(role) > 60 {
		return domain.User{}, domain.Fail(400, "invalid_profile", "نام باید بین ۲ تا ۵۰ کاراکتر باشد.")
	}
	valid := false
	for _, c := range []string{"violet", "mint", "peach", "blue", "pink", "yellow"} {
		if c == avatar {
			valid = true
		}
	}
	if !valid {
		return domain.User{}, domain.Fail(400, "invalid_avatar", "رنگ پروفایل معتبر نیست.")
	}
	if appearance != nil && !appearance.Valid() {
		return domain.User{}, domain.Fail(400, "invalid_avatar_config", "گزینه‌های آواتار معتبر نیستند.")
	}
	var u domain.User
	var rooms []string
	err := a.store.Write(ctx, "user:"+id, func(tx repositories.Transaction) error {
		var err error
		u, err = tx.User(id)
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.Fail(404, "not_found", "کاربر پیدا نشد.")
		}
		if err != nil {
			return err
		}
		u.Name = name
		u.Avatar = avatar
		u.Role = role
		if appearance != nil {
			copy := *appearance
			u.AvatarConfig = &copy
		}
		if err := tx.SaveUser(u); err != nil {
			return err
		}
		rooms, err = tx.UserRoomIDs(id)
		return err
	})
	if err == nil {
		for _, r := range rooms {
			a.hub.Publish(r)
		}
	}
	if u.AvatarConfig != nil {
		copy := *u.AvatarConfig
		u.AvatarConfig = &copy
	}
	return u, err
}
