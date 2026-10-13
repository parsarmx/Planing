package rest

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"net"
	"planningpoker/handler/dtos"
	"planningpoker/internal/config"
	"planningpoker/internal/domain"
	"planningpoker/internal/realtime"
	"planningpoker/internal/services"
	"time"
)

type Controller struct {
	Auth   *services.Auth
	Rooms  *services.Rooms
	Hub    *realtime.Hub
	Config *config.Config
}

func New(a *services.Auth, r *services.Rooms, h *realtime.Hub, c *config.Config) *Controller {
	return &Controller{a, r, h, c}
}
func parse(c *fiber.Ctx, v any) error {
	if err := c.BodyParser(v); err != nil {
		return domain.Fail(400, "invalid_body", "اطلاعات درخواست معتبر نیست.")
	}
	return nil
}
func user(c *fiber.Ctx) domain.User { return c.Locals("user").(domain.User) }
func (c *Controller) Authenticate(ctx *fiber.Ctx) error {
	u, err := c.Auth.Resolve(ctx.UserContext(), ctx.Cookies("poker_session"))
	if err != nil {
		return err
	}
	ctx.Locals("user", u)
	return ctx.Next()
}
func (c *Controller) RequestOTP(ctx *fiber.Ctx) error {
	var d dtos.OTPRequest
	if err := parse(ctx, &d); err != nil {
		return err
	}
	ch, err := c.Auth.Request(ctx.UserContext(), d.Phone)
	if err != nil {
		return err
	}
	return ctx.JSON(ch)
}
func (c *Controller) VerifyOTP(ctx *fiber.Ctx) error {
	var d dtos.OTPVerify
	if err := parse(ctx, &d); err != nil {
		return err
	}
	u, token, err := c.Auth.Verify(ctx.UserContext(), d.Phone, d.Code)
	if err != nil {
		return err
	}
	ctx.Cookie(&fiber.Cookie{Name: "poker_session", Value: token, Path: "/", HTTPOnly: true, Secure: c.Config.SecureCookies, SameSite: "Strict", MaxAge: 604800, Expires: time.Now().Add(7 * 24 * time.Hour)})
	return ctx.JSON(u)
}
func (c *Controller) Me(ctx *fiber.Ctx) error { return ctx.JSON(user(ctx)) }
func (c *Controller) Logout(ctx *fiber.Ctx) error {
	if err := c.Auth.Logout(ctx.UserContext(), ctx.Cookies("poker_session")); err != nil {
		return err
	}
	ctx.Cookie(&fiber.Cookie{Name: "poker_session", Value: "", Path: "/", HTTPOnly: true, Secure: c.Config.SecureCookies, SameSite: "Strict", Expires: time.Now().Add(-time.Hour), MaxAge: -1})
	return ctx.SendStatus(204)
}
func (c *Controller) Profile(ctx *fiber.Ctx) error {
	var d dtos.Profile
	if err := parse(ctx, &d); err != nil {
		return err
	}
	u, err := c.Auth.Profile(ctx.UserContext(), user(ctx).ID, d.Name, d.Avatar, d.Role, d.AvatarConfig)
	if err != nil {
		return err
	}
	return ctx.JSON(u)
}
func (c *Controller) List(ctx *fiber.Ctx) error {
	out, err := c.Rooms.List(ctx.UserContext(), user(ctx).ID)
	if err != nil {
		return err
	}
	return ctx.JSON(out)
}
func (c *Controller) PublicRooms(ctx *fiber.Ctx) error {
	limit, offset := ctx.QueryInt("limit", 12), ctx.QueryInt("offset", 0)
	out, err := c.Rooms.Public(ctx.UserContext(), limit, offset)
	if err != nil {
		return err
	}
	return ctx.JSON(out)
}
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var d dtos.CreateRoom
	if err := parse(ctx, &d); err != nil {
		return err
	}
	out, err := c.Rooms.Create(ctx.UserContext(), user(ctx).ID, d.Name, d.Deck, d.Theme, d.Visibility)
	if err != nil {
		return err
	}
	return ctx.Status(201).JSON(out)
}
func (c *Controller) Join(ctx *fiber.Ctx) error {
	var d dtos.JoinRoom
	if err := parse(ctx, &d); err != nil {
		return err
	}
	out, err := c.Rooms.Join(ctx.UserContext(), ctx.Params("id"), user(ctx).ID, d.Observer)
	if err != nil {
		return err
	}
	return ctx.JSON(out)
}
func (c *Controller) Get(ctx *fiber.Ctx) error {
	out, err := c.Rooms.Snapshot(ctx.UserContext(), ctx.Params("id"), user(ctx).ID)
	if err != nil {
		return err
	}
	return ctx.JSON(out)
}
func (c *Controller) Command(ctx *fiber.Ctx) error {
	var d services.Command
	if err := parse(ctx, &d); err != nil {
		return err
	}
	out, err := c.Rooms.Command(ctx.UserContext(), ctx.Params("id"), user(ctx).ID, d)
	if err != nil {
		return err
	}
	return ctx.JSON(out)
}
func (c *Controller) Events(ctx *fiber.Ctx) error {
	id, uid, token := ctx.Params("id"), user(ctx).ID, ctx.Cookies("poker_session")
	snapshot, err := c.Rooms.Snapshot(ctx.UserContext(), id, uid)
	if err != nil {
		return err
	}
	// HTTP 204 instructs EventSource clients to stop reconnecting to a closed room.
	if snapshot.Closed {
		return ctx.SendStatus(204)
	}
	if err := c.Rooms.Touch(ctx.UserContext(), id, uid); err != nil {
		return err
	}
	ch, unsubscribe := c.Hub.Subscribe(id, uid)
	if ch == nil {
		if !c.Hub.Online(id, uid) && c.Hub.Count(id) >= 20 {
			return domain.Fail(409, "room_full", "ظرفیت اتاق ۲۰ نفر حاضر است.")
		}
		return domain.Fail(429, "connection_limit", "حداکثر ۶ اتصال هم‌زمان برای هر کاربر مجاز است.")
	}
	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache, no-transform")
	ctx.Set("X-Accel-Buffering", "no")
	connection := ctx.Context().Conn()
	ctx.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer unsubscribe()
		c.stream(w, connection, id, uid, token, ch)
	})
	return nil
}

// No Fiber context escapes into the writer: fasthttp reuses it after the handler.
func (c *Controller) stream(w *bufio.Writer, connection net.Conn, id, uid, token string, ch <-chan struct{}) {
	heartbeat, idle := c.Config.Realtime.Heartbeat, c.Config.Realtime.IdleTimeout
	if heartbeat <= 0 {
		heartbeat = 10 * time.Second
	}
	if idle <= 0 {
		idle = 24 * time.Hour
	}
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()
	idleTimer := time.NewTimer(idle)
	defer idleTimer.Stop()
	write := func(event string, data []byte) bool {
		if connection != nil {
			_ = connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
		}
		_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		if err == nil {
			err = w.Flush()
		}
		if connection != nil {
			_ = connection.SetWriteDeadline(time.Time{})
		}
		return err == nil
	}
	terminal := func(event string) bool { write(event, []byte("{}")); return false }
	send := func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := c.Auth.Resolve(ctx, token); err != nil {
			var authError *domain.Error
			if errors.As(err, &authError) && authError.Status == 401 {
				return terminal("expired")
			}
			return terminal("unavailable")
		}
		snapshot, err := c.Rooms.Snapshot(ctx, id, uid)
		if err != nil {
			return terminal("unavailable")
		}
		if !snapshot.Closed && time.Since(snapshot.LastActivityAt) >= idle {
			return terminal("idle")
		}
		b, err := json.Marshal(snapshot)
		if err != nil || !write("room", b) {
			return false
		}
		if snapshot.Closed {
			return terminal("closed")
		}
		if !idleTimer.Stop() {
			select {
			case <-idleTimer.C:
			default:
			}
		}
		idleTimer.Reset(time.Until(snapshot.LastActivityAt.Add(idle)))
		return true
	}
	if !send() {
		return
	}
	for {
		select {
		case <-c.Hub.Done():
			return
		case <-ch:
			if !send() {
				return
			}
		case <-ticker.C:
			if !send() {
				return
			}
		case <-idleTimer.C:
			if !send() {
				return
			}
		}
	}
}
