package main

import (
	"log/slog"
	"os"
	"strings"
)

// Env holds environment variables exposed to the lambda.
type Env struct {
	NonceTableName       string
	ReleaseUploadRoleARN string
	ReleaseBucketName    string
	LogLevel             slog.Level
}

// LoadEnv parses an [Env] from the environment.
func LoadEnv() Env {
	env := Env{}

	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			// Malformed entry somehow
			continue
		}

		switch key {
		case "LOG_LEVEL":
			env.LogLevel = parseLogLevel(value)
		case "NONCE_TABLE_NAME":
			env.NonceTableName = value
		case "RELEASE_UPLOAD_ROLE_ARN":
			env.ReleaseUploadRoleARN = value
		case "RELEASE_BUCKET_NAME":
			env.ReleaseBucketName = value
		default:
			continue
		}
	}

	return env
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
