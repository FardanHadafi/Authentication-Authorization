package shared

import (
	"log/slog"
	"os"
)

func InitLogger() *slog.Logger {
	options := slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}

	handler := slog.NewJSONHandler(os.Stdout, &options)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
