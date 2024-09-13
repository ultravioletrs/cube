package middleware

import (
	"context"
	"log/slog"
	"time"

	proxy "github.com/ultraviolet/vault-proxy"
)

var _ proxy.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    proxy.Service
}

func NewLoggingMiddleware(logger *slog.Logger, svc proxy.Service) proxy.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm *loggingMiddleware) Identify(ctx context.Context, token string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Identify user failed", args...)
			return

		}
		lm.logger.Info("Identify user completed successfully", args...)
	}(time.Now())

	return lm.svc.Identify(ctx, token)
}
