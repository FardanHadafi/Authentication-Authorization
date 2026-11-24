package shared

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

func InitLogger() *slog.Logger {
	options := slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}

	var writer io.Writer = os.Stdout

	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "logs"
	}

	if err := os.MkdirAll(logDir, 0755); err == nil {
		logFilePath := filepath.Join(logDir, "app.log")
		if logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			writer = io.MultiWriter(os.Stdout, logFile)
		}
	}

	handler := slog.NewJSONHandler(writer, &options)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
