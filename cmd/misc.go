package main

import (
	"log/slog"
	"os"
)

func initTrace(debugLevel string) *slog.Logger {
	handlerOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		// AddSource: true,
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

// func printConfiguration(c *config.Config) {
// 	if len(c.S3cfg.AccessKey) != 0 {
// 		c.S3cfg.AccessKey = "*****"
// 	}
// 	if len(c.S3cfg.SecretKey) != 0 {
// 		c.S3cfg.SecretKey = "*****"
// 	}
// 	if len(c.GitlabToken) != 0 {
// 		c.GitlabToken = "*****"
// 	}
// 	fmt.Println("Actual configuration (yaml format):")
// 	fmt.Println("---------------------------------")
// 	fmt.Println(c)
// 	fmt.Println("---------------------------------")
// 	fmt.Println("Configuration can be done with environment variables:")
// 	f := cleanenv.Usage(c, nil)
// 	f()
// }
