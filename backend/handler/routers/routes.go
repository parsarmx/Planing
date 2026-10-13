package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"planningpoker/handler/controllers/rest"
	"planningpoker/internal/config"
	"planningpoker/internal/domain"
	"strings"
	"time"
)

func Register(a *fiber.App, c *rest.Controller, cfg *config.Config) {
	a.Use(func(ctx *fiber.Ctx) error {
		ctx.Set("X-Content-Type-Options", "nosniff")
		ctx.Set("Referrer-Policy", "same-origin")
		ctx.Set("X-Frame-Options", "DENY")
		if strings.HasPrefix(ctx.Path(), "/api/") {
			ctx.Set("Cache-Control", "no-store")
		}
		if ctx.Method() != "GET" && ctx.Method() != "HEAD" && ctx.Method() != "OPTIONS" {
			origin := ctx.Get("Origin")
			if origin != "" {
				allowed := false
				for _, o := range cfg.Origins {
					if origin == strings.TrimSpace(o) {
						allowed = true
					}
				}
				if !allowed {
					return domain.Fail(403, "invalid_origin", "مبدأ درخواست مجاز نیست.")
				}
			}
			if !strings.HasPrefix(ctx.Get("Content-Type"), "application/json") {
				return domain.Fail(415, "content_type", "درخواست باید JSON باشد.")
			}
		}
		return ctx.Next()
	})
	a.Get("/api/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok", "environment": cfg.Environment}) })
	auth := a.Group("/api/auth", limiter.New(limiter.Config{Max: 20, Expiration: time.Minute, LimitReached: func(c *fiber.Ctx) error {
		return domain.Fail(429, "rate_limit", "تعداد درخواست‌ها زیاد است؛ کمی صبر کنید.")
	}}))
	auth.Post("/otp", c.RequestOTP)
	auth.Post("/verify", c.VerifyOTP)
	a.Get("/api/rooms/public", c.PublicRooms)
	api := a.Group("/api", c.Authenticate)
	api.Get("/me", c.Me)
	api.Patch("/me", c.Profile)
	api.Post("/auth/logout", c.Logout)
	api.Get("/rooms", c.List)
	api.Post("/rooms", c.Create)
	api.Post("/rooms/:id/join", c.Join)
	api.Get("/rooms/:id/events", c.Events)
	api.Get("/rooms/:id", c.Get)
	api.Post("/rooms/:id/commands", c.Command)
	a.All("/api/*", func(*fiber.Ctx) error { return domain.Fail(404, "not_found", "مسیر پیدا نشد.") })
	a.Static("/", cfg.FrontendPath, fiber.Static{Index: "index.html"})
	a.Get("*", func(c *fiber.Ctx) error { return c.SendFile(cfg.FrontendPath + "/index.html") })
}
