package app

import (
	"context"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"net"
	"planningpoker/handler/controllers/rest"
	"planningpoker/handler/routers"
	"planningpoker/internal/config"
	"planningpoker/internal/domain"
	"planningpoker/internal/producers"
	"planningpoker/internal/realtime"
	"planningpoker/internal/repositories"
	"planningpoker/internal/services"
)

func Framework(logger *zap.Logger) *fiber.App {
	a := fiber.New(fiber.Config{Immutable: true, DisableStartupMessage: true, BodyLimit: 64 * 1024, ErrorHandler: func(c *fiber.Ctx, err error) error {
		var e *domain.Error
		if errors.As(err, &e) {
			return c.Status(e.Status).JSON(fiber.Map{"error": fiber.Map{"code": e.Code, "message": e.Message}})
		}
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fiber.Map{"code": "http_error", "message": "درخواست قابل پردازش نیست."}})
		}
		logger.Error("request failed", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": "خطایی رخ داد. دوباره تلاش کنید."}})
	}})
	a.Use(recover.New())
	return a
}
func New(c *config.Config) *fx.App {
	return fx.New(
		fx.Supply(c),
		fx.Provide(zap.NewProduction, repositories.New, producers.NewSMS, realtime.New, services.NewAuth, services.NewRooms, rest.New, Framework,),
		fx.Invoke(routers.Register, func(lc fx.Lifecycle, a *fiber.App, c *config.Config, s repositories.Store, h *realtime.Hub, l *zap.Logger, shutdown fx.Shutdowner) {
			lc.Append(fx.Hook{OnStart: func(context.Context) error {
				listener, err := net.Listen("tcp", c.Addr)
				if err != nil {
					return err
				}
				l.Info("Planning Poker ready", zap.String("url", "http://"+listener.Addr().String()), zap.String("environment", c.Environment))
				go func() {
					if err := a.Listener(listener); err != nil {
						l.Error("server stopped", zap.Error(err))
						_ = shutdown.Shutdown(fx.ExitCode(1))
					}
				}()
				return nil
			}, OnStop: func(ctx context.Context) error {
				h.Stop()
				err := a.ShutdownWithContext(ctx)
				s.Close()
				_ = l.Sync()
				return err
			}})
		}),
	)
}
