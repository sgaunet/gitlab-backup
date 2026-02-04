package main

import (
	"log/slog"
	"os"
)

func initTrace(debugLevel string, noLogTime bool) *slog.Logger {
	handlerOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	if noLogTime {
		handlerOptions.ReplaceAttr = func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{} // Remove the time attribute
			}
			return a
		}
	}

	switch debugLevel {
	case "debug":
		handlerOptions.Level = slog.LevelDebug
		handlerOptions.AddSource = true
	case "info":
		handlerOptions.Level = slog.LevelInfo
	case "warn":
		handlerOptions.Level = slog.LevelWarn
	case "error":
		handlerOptions.Level = slog.LevelError
	default:
		handlerOptions.Level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, handlerOptions)
	// handler := slog.NewJSONHandler(os.Stdout, nil) // JSON format
	logger := slog.New(handler)
	return logger
}
