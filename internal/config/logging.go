package config

import (
	"os"
	"strings"
	"time"

	"log/slog"
)

func InitLogger(config LogConfig) {

	// timezone
	location, _ := time.LoadLocation("UTC")
	time.Local = location

	logLevel := strings.ToLower(config.Level)

	var level slog.Level

	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handlerOptions := slog.HandlerOptions{
		Level: level,
	}

	logFormat := strings.ToLower(config.Format)
	var handler slog.Handler

	switch logFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &handlerOptions)
	case "text":
		fallthrough
	default:
		handler = slog.NewTextHandler(os.Stdout, &handlerOptions)
	}

	slog.SetDefault(slog.New(handler))
}
