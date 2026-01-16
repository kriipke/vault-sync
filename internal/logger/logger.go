package logger

import (
	"context"
	"log/slog"
	"os"
	"vault-sync/internal/config"
)

var defaultLogger *slog.Logger

func Init(cfg *config.Config) {
	var handler slog.Handler
	
	opts := &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}
	
	if cfg.Verbose {
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func Debug(msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.Debug(msg, args...)
	}
}

func Info(msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.Error(msg, args...)
	}
}

func DebugCtx(ctx context.Context, msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.DebugContext(ctx, msg, args...)
	}
}

func InfoCtx(ctx context.Context, msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.InfoContext(ctx, msg, args...)
	}
}

func WarnCtx(ctx context.Context, msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.WarnContext(ctx, msg, args...)
	}
}

func ErrorCtx(ctx context.Context, msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.ErrorContext(ctx, msg, args...)
	}
}

func With(args ...any) *slog.Logger {
	if defaultLogger != nil {
		return defaultLogger.With(args...)
	}
	return slog.Default()
}